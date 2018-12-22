package fpga

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"github.com/hyperledger/fabric/protos/fpga"
	"google.golang.org/grpc"
	"math/big"
	"net"
)

func init() {
	go start()
}

type fpgaServer struct {

}

func (s *fpgaServer) VerifySig4Vscc(cxt context.Context, env *fpga.VsccEnvelope) (*fpga.VsccResponse, error) {
	logger.Infof("mock fpga  receive a VerifySig4Vscc request")

	intR := big.Int{}
	intR.SetString(env.SignR, 10)
	intS := big.Int{}
	intS.SetString(env.SignS, 10)

	x := big.Int{}
	x.SetString(env.PkX, 10)
	y := big.Int{}
	y.SetString(env.PkY, 10)
	pubkey := ecdsa.PublicKey{elliptic.P256(), &x, &y}

	digest, _ := base64.StdEncoding.DecodeString(env.E)
	valid := ecdsa.Verify(&pubkey, digest, &intR, &intS)
	if !valid {
		logger.Warnf("grpc server verify result: false")
	}
	return &fpga.VsccResponse{Result:valid}, nil
}

func (s *fpgaServer) SendBlock4Mvcc(context.Context, *fpga.Block4Mvcc) (*fpga.MvccResponse, error) {
	logger.Infof("mock fpga  receive a SendBlock4Mvcc request")
	return &fpga.MvccResponse{Result:true}, nil
}

func newServer() *fpgaServer {
	s := &fpgaServer{}
	return s
}

func start() {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	fpga.RegisterFpgaServer(grpcServer, newServer())
	logger.Infof("start a fpga mock server.")
	lis, err := net.Listen("tcp", "0.0.0.0:10000")
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatalf("grpcServer.Serve(lis) failed.")
	}
}
