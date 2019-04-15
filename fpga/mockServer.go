package fpga

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"github.com/hyperledger/fabric/protos/fpga"
	"google.golang.org/grpc"
	"io"
	"math/big"
	"net"
	"os"
)

func init() {
	if os.Getenv("FPGA_MOCK") == "1" {
		logger.Infof("going to start the mockserver")
		go start()
	}
}

type fpgaServer struct {
}

func (s *fpgaServer) Sign(stream fpga.BatchRPC_SignServer) error {

	for {
		batchReq, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				fmt.Printf("sign stream.Recv() EOF!, err: %v \n\n", err)
			} else {
				fmt.Printf("sign stream.Recv() failed!, err: %v \n\n", err)
			}
			return err
		}

		reply := &fpga.BatchReply{}

		for _, request := range batchReq.SgRequests {
			d := &big.Int{}
			d.SetBytes(request.D)
			privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			privateKey.D = d

			r, s, err := ecdsa.Sign(rand.Reader, privateKey, request.Hash)
			if err != nil {
				return err
			}
			signature := &fpga.BatchReply_SignGenReply{ReqId: request.ReqId, SignR:r.Bytes(), SignS:s.Bytes()}
			reply.SgReplies = append(reply.SgReplies, signature)
			reply.BatchId = batchReq.BatchId
		}
		logger.Debugf("mock server returned sign rpc. batch_id: %d, ReqCount: %d", reply.BatchId, len(reply.SgReplies))
	}

	return nil
}

func (s *fpgaServer) Verify(stream fpga.BatchRPC_VerifyServer) error {

	for {
		batchReq, err := stream.Recv()
		if err != nil {
			fmt.Printf("verify stream.Recv() failed!, err: %v \n\n", err)
			return err
		}

		reply := &fpga.BatchReply{}

		for _, request := range batchReq.SvRequests {
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
				logger.Infof("grpc server verify result: false")
			}
			result := &fpga.BatchReply_SignVerReply{ReqId:request.ReqId, Verified:valid}
			reply.SvReplies = append(reply.SvReplies, result)
			reply.BatchId = batchReq.BatchId
		}
		logger.Debugf("mock server returned verify rpc. batch_id: %d, ReqCount: %d", reply.BatchId, len(reply.SvReplies))
	}

	return nil
}

func newServer() *fpgaServer {
	s := &fpgaServer{}
	return s
}

func start() {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	fpga.RegisterBatchRPCServer(grpcServer, newServer())
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