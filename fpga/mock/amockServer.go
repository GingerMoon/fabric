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
	"sync"
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
	//s.logger.Debugf("Sign rpc invoke")
	reply := &fpga.BatchReply{}
	reply.BatchId = env.BatchId

	ch := make(chan *fpga.BatchReply_SignGenReply)
	wg := &sync.WaitGroup{}
	wg.Add(len(env.SgRequests))
	for _, request := range env.SgRequests {
		go s.sign_(wg, ch, request)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	for signature := range ch {
		reply.SgReplies = append(reply.SgReplies, signature)

	}

	//s.logger.Debugf("mock server returned sign rpc. batch_id: %d, ReqCount: %d", reply.BatchId, len(reply.SgReplies))
	return reply, nil
}

func (s *fpgaServer) sign_(wg *sync.WaitGroup, ch chan *fpga.BatchReply_SignGenReply, request *fpga.BatchRequest_SignGenRequest) {
	defer wg.Done()
	d := &big.Int{}
	d.SetBytes(request.D)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privateKey.D = d

	rr, ss, err := ecdsa.Sign(rand.Reader, privateKey, request.Hash)
	if err != nil {
		s.logger.Fatalf("sign failed! err: %v", err)
	}
	signature := &fpga.BatchReply_SignGenReply{ReqId: request.ReqId, SignR:rr.Bytes(), SignS:ss.Bytes()}
	ch <- signature
}

func (s *fpgaServer) Verify(ctx context.Context, env *fpga.BatchRequest) (*fpga.BatchReply, error) {
	//s.logger.Debugf("Verify rpc invoke")
	reply := &fpga.BatchReply{}
	reply.BatchId = env.BatchId

	ch := make(chan *fpga.BatchReply_SignVerReply)
	wg := &sync.WaitGroup{}
	wg.Add(len(env.SvRequests))
	for _, request := range env.SvRequests {
		go s.verify_(wg, ch, request)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	for result := range ch {
		reply.SvReplies = append(reply.SvReplies, result)

	}

	//s.logger.Debugf("mock server returned verify rpc. batch_id: %d, ReqCount: %d", reply.BatchId, len(reply.SvReplies))

	return reply, nil
}

func (s *fpgaServer) verify_(wg *sync.WaitGroup, ch chan *fpga.BatchReply_SignVerReply, request *fpga.BatchRequest_SignVerRequest) {
	defer wg.Done()
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
	ch <- result
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