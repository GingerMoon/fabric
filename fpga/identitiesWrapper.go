package fpga

import (
	"github.com/hyperledger/fabric/core/handlers/endorsement/api/identities"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/cache"
	"reflect"
	"unsafe"
)

func FpgaEndorserSign(signer endorsement.SigningIdentity, msg []byte) ([]byte, error) {
	logger.Warningf(reflect.TypeOf(signer).String())
	// sign the concatenation of the proposal response and the serialized endorser identity with this endorser's key
	if reflect.TypeOf(signer).String() != "*msp.signingidentity" {
		logger.Fatalf(reflect.TypeOf(signer).String())
	}
	fpgaSigner := (*msp.FpgaSigningidentity)(unsafe.Pointer(reflect.ValueOf(signer).Pointer()))
	return fpgaSigner.Sign(msg)
}

func FpgaEndorserVerify(id msp.Identity, msg []byte, sig []byte) error {

	if reflect.TypeOf(id).String() != "*cache.cachedIdentity" {
		logger.Fatalf(reflect.TypeOf(id).String())
	}
	fpgaCachedIdentity := (*cache.FpgaCachedIdentity)(unsafe.Pointer(reflect.ValueOf(id).Pointer()))

	if reflect.TypeOf(fpgaCachedIdentity.Identity).String() != "*msp.identity" {
		logger.Fatalf(reflect.TypeOf(id).String())
	}

	fpgaIdentity := (*msp.FpgaIdentity)(unsafe.Pointer(reflect.ValueOf(fpgaCachedIdentity.Identity).Pointer()))
	return fpgaIdentity.Verify(msg, sig)
}