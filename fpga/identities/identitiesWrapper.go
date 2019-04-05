package identities

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/handlers/endorsement/api/identities"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/cache"
	"reflect"
	"unsafe"
)

var (
	logger = flogging.MustGetLogger("fpga")
)

func FpgaEndorserSign(signer endorsement.SigningIdentity, msg []byte) ([]byte, error) {
	if reflect.TypeOf(signer).String() != "*msp.signingidentity" {
		logger.Fatalf(reflect.TypeOf(signer).String())
	}
	fpgaSigner := (*msp.FpgaSigningidentity)(unsafe.Pointer(reflect.ValueOf(signer).Pointer()))
	return fpgaSigner.Sign(msg)
}

func generateFpgaIdentity(id msp.Identity) *msp.FpgaIdentity {
	if reflect.TypeOf(id).String() != "*cache.cachedIdentity" {
		logger.Fatalf(reflect.TypeOf(id).String())
	}
	fpgaCachedIdentity := (*cache.FpgaCachedIdentity)(unsafe.Pointer(reflect.ValueOf(id).Pointer()))

	if reflect.TypeOf(fpgaCachedIdentity.Identity).String() != "*msp.identity" {
		logger.Fatalf(reflect.TypeOf(id).String())
	}

	fpgaIdentity := (*msp.FpgaIdentity)(unsafe.Pointer(reflect.ValueOf(fpgaCachedIdentity.Identity).Pointer()))
	return fpgaIdentity
}

func FpgaVerify4Endorser(id msp.Identity, msg []byte, sig []byte) error {
	fpgaIdentity := generateFpgaIdentity(id)
	return fpgaIdentity.EndorserVerify(msg, sig)
}

func FpgaVerify4Committer(id msp.Identity, msg []byte, sig []byte) error {
	fpgaIdentity := generateFpgaIdentity(id)
	return fpgaIdentity.CommitterVerify(msg, sig)
}