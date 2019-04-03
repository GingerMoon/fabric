/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/utils"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
	"reflect"
	"time"
	"unsafe"
)

type FpgaIdentity struct {
	// id contains the identifier (MSPID and identity identifier) for this instance
	id *IdentityIdentifier

	// cert contains the x.509 certificate that signs the public key of this instance
	cert *x509.Certificate

	// this is the public key of this instance
	pk bccsp.Key

	// reference to the MSP that "owns" this identity
	msp *bccspmsp
}

// ExpiresAt returns the time at which the Identity expires.
func (id *FpgaIdentity) ExpiresAt() time.Time {
	return id.cert.NotAfter
}

// SatisfiesPrincipal returns null if this instance matches the supplied principal or an error otherwise
func (id *FpgaIdentity) SatisfiesPrincipal(principal *msp.MSPPrincipal) error {
	return id.msp.SatisfiesPrincipal(id, principal)
}

// GetIdentifier returns the identifier (MSPID/IDID) for this instance
func (id *FpgaIdentity) GetIdentifier() *IdentityIdentifier {
	return id.id
}

// GetMSPIdentifier returns the MSP identifier for this instance
func (id *FpgaIdentity) GetMSPIdentifier() string {
	return id.id.Mspid
}

// Validate returns nil if this instance is a valid identity or an error otherwise
func (id *FpgaIdentity) Validate() error {
	return id.msp.Validate(id)
}

// GetOrganizationalUnits returns the OU for this instance
func (id *FpgaIdentity) GetOrganizationalUnits() []*OUIdentifier {
	if id.cert == nil {
		return nil
	}

	cid, err := id.msp.getCertificationChainIdentifier(id)
	if err != nil {
		mspIdentityLogger.Errorf("Failed getting certification chain identifier for [%v]: [%+v]", id, err)

		return nil
	}

	res := []*OUIdentifier{}
	for _, unit := range id.cert.Subject.OrganizationalUnit {
		res = append(res, &OUIdentifier{
			OrganizationalUnitIdentifier: unit,
			CertifiersIdentifier:         cid,
		})
	}

	return res
}

// Anonymous returns true if this identity provides anonymity
func (id *FpgaIdentity) Anonymous() bool {
	return false
}

// Verify checks against a signature and a message
// to determine whether this identity produced the
// signature; it returns nil if so or an error otherwise
func (id *FpgaIdentity) Verify(msg []byte, sig []byte) error {
	// mspIdentityLogger.Infof("Verifying signature")

	// Compute Hash
	hashOpt, err := id.getHashOpt(id.msp.cryptoConfig.SignatureHashFamily)
	if err != nil {
		return errors.WithMessage(err, "failed getting hash function options")
	}

	digest, err := id.msp.bccsp.Hash(msg, hashOpt)
	if err != nil {
		return errors.WithMessage(err, "failed computing digest")
	}

	if mspIdentityLogger.IsEnabledFor(zapcore.DebugLevel) {
		mspIdentityLogger.Debugf("Verify: digest = %s", hex.Dump(digest))
		mspIdentityLogger.Debugf("Verify: sig = %s", hex.Dump(sig))
	}

	valid, err := id.msp.bccsp.Verify(id.pk, sig, digest, nil)
	if err != nil {
		return errors.WithMessage(err, "could not determine the validity of the signature")
	} else if !valid {
		return errors.New("The signature is invalid")
	}

	return nil
}

// Added for Accelor
func (id *FpgaIdentity) GetPublicKey() (*ecdsa.PublicKey, error) {
	pk := id.cert.PublicKey

	switch pk.(type) {
	case *ecdsa.PublicKey:
		return pk.(*ecdsa.PublicKey), nil
	case *rsa.PublicKey:
		return nil, errors.New("Certificate's public key type RSA is not supported for now.")
	default:
		return nil, errors.New("Certificate's public key type not recognized. Supported keys: [ECDSA]")
	}
}

// Serialize returns a byte array representation of this identity
func (id *FpgaIdentity) Serialize() ([]byte, error) {
	// mspIdentityLogger.Infof("Serializing identity %s", id.id)

	pb := &pem.Block{Bytes: id.cert.Raw, Type: "CERTIFICATE"}
	pemBytes := pem.EncodeToMemory(pb)
	if pemBytes == nil {
		return nil, errors.New("encoding of identity failed")
	}

	// We serialize identities by prepending the MSPID and appending the ASN.1 DER content of the cert
	sId := &msp.SerializedIdentity{Mspid: id.id.Mspid, IdBytes: pemBytes}
	idBytes, err := proto.Marshal(sId)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal a SerializedIdentity structure for identity %s", id.id)
	}

	return idBytes, nil
}

func (id *FpgaIdentity) getHashOpt(hashFamily string) (bccsp.HashOpts, error) {
	switch hashFamily {
	case bccsp.SHA2:
		return bccsp.GetHashOpt(bccsp.SHA256)
	case bccsp.SHA3:
		return bccsp.GetHashOpt(bccsp.SHA3_256)
	}
	return nil, errors.Errorf("hash familiy not recognized [%s]", hashFamily)
}

type FpgaSigningidentity struct {
	// we embed everything from a base identity
	FpgaIdentity

	// signer corresponds to the object that can produce signatures from this identity
	Signer crypto.Signer
}

// Sign produces a signature over msg, signed by this instance
func (id *FpgaSigningidentity) Sign(msg []byte) ([]byte, error) {
	//mspIdentityLogger.Infof("Signing message")

	// Compute Hash
	hashOpt, err := id.getHashOpt(id.msp.cryptoConfig.SignatureHashFamily)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting hash function options")
	}

	digest, err := id.msp.bccsp.Hash(msg, hashOpt)
	if err != nil {
		return nil, errors.WithMessage(err, "failed computing digest")
	}

	if len(msg) < 32 {
		mspIdentityLogger.Debugf("Sign: plaintext: %X \n", msg)
	} else {
		mspIdentityLogger.Debugf("Sign: plaintext: %X...%X \n", msg[0:16], msg[len(msg)-16:])
	}
	mspIdentityLogger.Debugf("Sign: digest: %X \n", digest)

	fpgaCryptoSigner := (*fpgaCryptoSigner)(unsafe.Pointer(reflect.ValueOf(id.Signer).Pointer()))
	mspPrikey := (*ecdsaPrivateKey)(unsafe.Pointer(reflect.ValueOf(fpgaCryptoSigner.key).Pointer()))
	prikey := mspPrikey.privKey


	// Sign
	r, s, err := ecdsa.Sign(rand.Reader, prikey, digest)
	if err != nil {
		return nil, err
	}
	s, _, err = utils.ToLowS(&prikey.PublicKey, s)
	if err != nil {
		return nil, err
	}
	return utils.MarshalECDSASignature(r, s)

	//return id.Signer.Sign(rand.Reader, digest, nil)
}