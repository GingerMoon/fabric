/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/utils"
	commonerrors "github.com/hyperledger/fabric/common/errors"
	"github.com/hyperledger/fabric/fpga"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
	"math/big"
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

func (id *FpgaIdentity) Verify(msg []byte, sig []byte) error {
	panic("this function should not be invoked!")
}

func (id *FpgaIdentity) GetPublicKey() (*ecdsa.PublicKey, error) {
	if reflect.TypeOf(id.pk).String() != "*sw.ecdsaPublicKey" {
		return nil, errors.Errorf("expected identity's publilc key type: *sw.ecdsaPublicKey, but got %s", reflect.TypeOf(id.pk).String())
	}
	pk := (*ecdsaPublicKey)(unsafe.Pointer(reflect.ValueOf(id.pk).Pointer()))
	pubkey := pk.pubKey
	return pubkey, nil
}

// Verify checks against a signature and a message
// to determine whether this identity produced the
// signature; it returns nil if so or an error otherwise
func (id *FpgaIdentity) prepare4Verify(msg []byte, sig []byte) (*pb.BatchRequest_SignVerRequest, error) {

	// Compute Hash
	hashOpt, err := id.getHashOpt(id.msp.cryptoConfig.SignatureHashFamily)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting hash function options")
	}

	digest, err := id.msp.bccsp.Hash(msg, hashOpt)
	if err != nil {
		return nil, errors.WithMessage(err, "failed computing digest")
	}

	r, s, err := utils.UnmarshalECDSASignature(sig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed UnmarshalECDSASignature")
	}

	if reflect.TypeOf(id.pk).String() != "*sw.ecdsaPublicKey" {
		panic(fmt.Sprintf("expected identity's publilc key type: *sw.ecdsaPublicKey, but got %s", reflect.TypeOf(id.pk).String()))
	}
	pk := (*ecdsaPublicKey)(unsafe.Pointer(reflect.ValueOf(id.pk).Pointer()))

	pubkey := pk.pubKey
	lowS, err := utils.IsLowS(pubkey, s)
	if err != nil {
		return nil, errors.WithMessage(err, "failed IsLowS")
	}
	if !lowS {
		return nil, errors.WithMessage(err, fmt.Sprintf("Invalid S. Must be smaller than half the order [%s][%s].", s, utils.GetCurveHalfOrdersAt(pubkey.Curve)))
	}

	in := &pb.BatchRequest_SignVerRequest{SignR:r.Bytes(), SignS:s.Bytes(), Px:pubkey.X.Bytes(), Hash:digest, Py:pubkey.Y.Bytes()}
	return in, nil
}

func (id *FpgaIdentity) EndorserVerify(msg []byte, sig []byte) error {
	in, err := id.prepare4Verify(msg, sig)
	if err != nil {
		return err
	}

	valid := fpga.EndorserVerify(in)
	if !valid {
		r := &big.Int{}
		r.SetBytes(in.SignR)
		s := &big.Int{}
		s.SetBytes(in.SignS)

		x := &big.Int{}
		x.SetBytes(in.Px)
		y := &big.Int{}
		y.SetBytes(in.Py)
		pubkey := ecdsa.PublicKey{Curve:elliptic.P256(), X:x, Y:y}
		succeed := ecdsa.Verify(&pubkey, in.Hash[:], r, s)
		if succeed {
			panic("The signature is invalid [verified by FPGA], but it is valid by go library.")
		}

		return &commonerrors.VSCCEndorsementPolicyError{
			Err: errors.New("The signature is invalid [verified by FPGA]. "),
		}
	}
	return nil
}

func (id *FpgaIdentity) CommitterVerify(msg []byte, sig []byte) error {
	//in, err := id.prepare4Verify(msg, sig)
	//if err != nil {
	//	return err
	//}
	//
	//valid := fpga.CommitterVerify(in)
	//if !valid {
	//	return errors.New("The signature is invalid")
	//}
	return nil
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

	// get the private key
	if reflect.TypeOf(id.Signer).String() != "*signer.bccspCryptoSigner" {
		mspIdentityLogger.Fatalf(reflect.TypeOf(id.Signer).String())
	}
	fpgaCryptoSigner := (*fpgaCryptoSigner)(unsafe.Pointer(reflect.ValueOf(id.Signer).Pointer()))

	if reflect.TypeOf(fpgaCryptoSigner.key).String() != "*sw.ecdsaPrivateKey" {
		mspIdentityLogger.Fatalf(reflect.TypeOf(fpgaCryptoSigner.key).String())
	}
	mspPrikey := (*ecdsaPrivateKey)(unsafe.Pointer(reflect.ValueOf(fpgaCryptoSigner.key).Pointer()))

	privkey := mspPrikey.privKey

	// Compute Hash
	hashOpt, err := id.getHashOpt(id.msp.cryptoConfig.SignatureHashFamily)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting hash function options")
	}

	hash, err := id.msp.bccsp.Hash(msg, hashOpt)
	if err != nil {
		return nil, errors.WithMessage(err, "failed computing digest")
	}

	// fpga endorser sign
	in := &pb.BatchRequest_SignGenRequest{D: privkey.D.Bytes(), Hash: hash}
	signature := fpga.EndorserSign(in)

	// marshall signature
	s := &big.Int{}
	s.SetBytes(signature.SignS)
	s, _, err = utils.ToLowS(&privkey.PublicKey, s)
	if err != nil {
		return nil, err
	}
	r := &big.Int{}
	r.SetBytes(signature.SignR)
	return utils.MarshalECDSASignature(r, s)
}