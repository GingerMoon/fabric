package mock

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/protos/fpga"
	"google.golang.org/grpc"
	"math/big"
	"net"
	"os"
)

func init() {
	if os.Getenv("FPGA_MOCK") == "1" {
		go start()
	}
}

type fpgaServer struct {
	logger *flogging.FabricLogger
}

func (s *fpgaServer) Sign(ctx context.Context, env *fpga.BatchRequest) (*fpga.BatchReply, error) {
	s.logger.Debugf("Sign rpc invoke")
	reply := &fpga.BatchReply{}

	for _, request := range env.SgRequests {
		d := &big.Int{}
		d.SetBytes(request.D)
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		privateKey.D = d

		r, s, err := ecdsa.Sign(rand.Reader, privateKey, request.Hash)
		if err != nil {
			return nil, err
		}
		signature := &fpga.BatchReply_SignGenReply{ReqId: request.ReqId, SignR:r.Bytes(), SignS:s.Bytes()}
		reply.SgReplies = append(reply.SgReplies, signature)
		reply.BatchId = env.BatchId
	}
	s.logger.Debugf("mock server returned sign rpc. batch_id: %d, ReqCount: %d", reply.BatchId, len(reply.SgReplies))
	return reply, nil
}

func (s *fpgaServer) Verify(ctx context.Context, env *fpga.BatchRequest) (*fpga.BatchReply, error) {
	s.logger.Debugf("Verify rpc invoke")
	reply := &fpga.BatchReply{}

	for _, request := range env.SvRequests {
		intR := big.Int{}
		intR.SetBytes(request.SignR)
		intS := big.Int{}
		intS.SetBytes(request.SignS)

		x := big.Int{}
		x.SetBytes(request.Px)
		y := big.Int{}
		y.SetBytes(request.Py)
		pubkey := ecdsa.PublicKey{elliptic.P256(), &x, &y}

		valid := ecdsa.Verify(&pubkey, request.Hash, &intR, &intS)
		if !valid {
			s.logger.Warningf("grpc server verify result: false")
		}
		result := &fpga.BatchReply_SignVerReply{ReqId:request.ReqId, Verified:valid}
		reply.SvReplies = append(reply.SvReplies, result)
		reply.BatchId = env.BatchId
	}
	s.logger.Debugf("mock server returned verify rpc. batch_id: %d, ReqCount: %d", reply.BatchId, len(reply.SvReplies))

	return reply, nil
}

func newServer() *fpgaServer {
	s := &fpgaServer{}
	s.logger = flogging.MustGetLogger("fpga.mockserver")
	return s
}

func start() {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	mockserver := newServer()
	fpga.RegisterBatchRPCServer(grpcServer, mockserver)
	mockserver.logger.Infof("start a fpga mock server.")
	lis, err := net.Listen("tcp", "0.0.0.0:10000")
	if err != nil {
		mockserver.logger.Fatalf("failed to listen: %v", err)
	}
	err = grpcServer.Serve(lis)
	if err != nil {
		mockserver.logger.Fatalf("grpcServer.Serve(lis) failed.")
	}
}