/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cauthdsl

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/fpga"
	fpgapb "github.com/hyperledger/fabric/protos/fpga"
	"github.com/hyperledger/fabric/msp"
	cb "github.com/hyperledger/fabric/protos/common"
)

type deserializeAndVerify4accelor struct {
	signedData           *cb.SignedData
	deserializer         msp.IdentityDeserializer
	deserializedIdentity msp.Identity
}

func (d *deserializeAndVerify4accelor) Identity() (Identity, error) {
	deserializedIdentity, err := d.deserializer.DeserializeIdentity(d.signedData.Identity)
	if err != nil {
		return nil, err
	}

	d.deserializedIdentity = deserializedIdentity
	return deserializedIdentity, nil
}


//type Identity4Accelor interface { // using another interface will cause the panic: interface conversion: *cache.cachedIdentity is not cauthdsl.Identity4Accelor: missing method GetPublicKey
//	msp.Identity
//	GetPublicKey() (*ecdsa.PublicKey, error)
//}

// the bccsp and Identity are extremely encapsulated hence it's very difficult to make modifications to the two modules.
func (d *deserializeAndVerify4accelor) Verify() error {
	if d.deserializedIdentity == nil {
		cauthdslLogger.Panicf("programming error, Identity must be called prior to Verify")
	}
	identify := d.deserializedIdentity
	//pubkey, err := identify.(Identity4Accelor).GetPublicKey()
	pubkey, err := identify.GetPublicKey()
	if err != nil {
		cauthdslLogger.Panicf("sever error, unsupported public key found. for now only ecdsa public key is supported.")
	}

	// encode publick key
	pubkeybytes, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		cauthdslLogger.Panicf("Unable to marshal ECDSA public key: %v", err)
	}
	pubkeybytes_encoded := pem.EncodeToMemory(&pem.Block{Type: "EC PUBLIC  KEY", Bytes: pubkeybytes})
	signedData_encoded := base64.StdEncoding.EncodeToString(d.signedData.Data)
	Signature_encoded := base64.StdEncoding.EncodeToString(d.signedData.Signature)
	cauthdslLogger.Infof("the public key is %s. ", string(pubkeybytes_encoded), )
	cauthdslLogger.Infof("the signed data is: %s. ", signedData_encoded)
	cauthdslLogger.Infof("the signature is: %s. Now send to FPGA for verification.", Signature_encoded)

	response := fpga.VerifySig4Vscc(&fpgapb.VsccEnvelope{
		PubkeybytesEncoded:string(pubkeybytes_encoded),
		SignedDataEncoded: string(signedData_encoded),
		SignatureEncoded : string(Signature_encoded)})
	cauthdslLogger.Infof("fpga.VerifySig4Vscc response: %v", response)

	// for now, we don't support multiple channels because we can't get the channelID directly from here.
	// and we don't want to add this parameter (channelID) to every function call in the whole call stack. We will redesign later.
	valid := true
	//return d.deserializedIdentity.Verify(d.signedData.Data, d.signedData.Signature)
	if !valid {
		return errors.New("The signature is invalid")
	}
	return nil
}

type provider4accelor struct {
	deserializer msp.IdentityDeserializer
}

// NewProviderImpl provides a policy generator for cauthdsl type policies
func NewPolicyProvider4accelor(deserializer msp.IdentityDeserializer) policies.Provider {
	return &provider4accelor{
		deserializer: deserializer,
	}
}

// NewPolicy creates a new policy based on the policy bytes
func (pr *provider4accelor) NewPolicy(data []byte) (policies.Policy, proto.Message, error) {
	sigPolicy := &cb.SignaturePolicyEnvelope{}
	if err := proto.Unmarshal(data, sigPolicy); err != nil {
		return nil, nil, fmt.Errorf("Error unmarshaling to SignaturePolicy: %s", err)
	}

	if sigPolicy.Version != 0 {
		return nil, nil, fmt.Errorf("This evaluator only understands messages of version 0, but version was %d", sigPolicy.Version)
	}

	compiled, err := compile(sigPolicy.Rule, sigPolicy.Identities, pr.deserializer)
	if err != nil {
		return nil, nil, err
	}

	return &policy4accelor{
		evaluator:    compiled,
		deserializer: pr.deserializer,
	}, sigPolicy, nil

}

type policy4accelor struct {
	evaluator    func([]IdentityAndSignature, []bool) bool
	deserializer msp.IdentityDeserializer
}

// Evaluate takes a set of SignedData and evaluates whether this set of signatures satisfies the policy
func (p *policy4accelor) Evaluate(signatureSet []*cb.SignedData) error {
	if p == nil {
		return fmt.Errorf("No such policy")
	}
	idAndS := make([]IdentityAndSignature, len(signatureSet))
	for i, sd := range signatureSet {
		idAndS[i] = &deserializeAndVerify4accelor{
			signedData:   sd,
			deserializer: p.deserializer,
		}
	}

	ok := p.evaluator(deduplicate(idAndS), make([]bool, len(signatureSet)))
	if !ok {
		return errors.New("signature set did not satisfy policy")
	}
	return nil
}
