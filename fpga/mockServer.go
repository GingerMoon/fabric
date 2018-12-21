package fpga

import (
	"context"
	"github.com/hyperledger/fabric/protos/fpga"
	"google.golang.org/grpc"
	"net"
)

func init() {
	go start()
}

type fpgaServer struct {

}

func (s *fpgaServer) VerifySig4Vscc(context.Context, *fpga.VsccEnvelope) (*fpga.VsccResponse, error) {
	logger.Infof("mock fpga  receive a VerifySig4Vscc request")
	return &fpga.VsccResponse{Result:true}, nil
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
