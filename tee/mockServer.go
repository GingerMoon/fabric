package tee

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/tee"
	"google.golang.org/grpc"
	"net"
	"os"
)

var logger = flogging.MustGetLogger("tee")

func init() {
	if os.Getenv("TEE_FPGA_MOCK") == "1" {
		go start()
	}
}

type fpgaServer struct {
}

func (s *fpgaServer) Execute(ctx context.Context, args *pb.TeeArgs) (*pb.TeeResp, error) {
	if len(args.Args) < 1 {
		return nil, errors.New("the args of Execute should be more than 1")
	}

	elf := string(args.Args[0])
	if elf == "paymentCCtee" {
		results, err := paymentCCtee(args.Args[1:])
		if err != nil {
			return nil, err
		} else {
			return &pb.TeeResp{Results:results}, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("unsupported function call. ELF: %s", elf))
}

func paymentCCtee(args [][]byte) (resp [][]byte, err error) {
	if len(args) != 3 {
		return nil, errors.New("paymentCCtee need 5 args!")
	}
	valueA := binary.LittleEndian.Uint32(args[0])
	valueB := binary.LittleEndian.Uint32(args[1])
	x := binary.LittleEndian.Uint32(args[2])

	if int(valueA) - int(x) < 0 {
		logger.Infof("not enough balance! valueA: %d, valueB: %d, x: %d", valueA, valueB, x)
		return nil, errors.New("not enough balance!")
	}
	valueA -= x
	bsA := make([]byte, 4)
	binary.LittleEndian.PutUint32(bsA, valueA)
	resp = append(resp, bsA)

	valueB += x
	bsB := make([]byte, 4)
	binary.LittleEndian.PutUint32(bsB, valueB)
	resp = append(resp, bsB)

	return resp, nil
}

func newServer() *fpgaServer {
	s := &fpgaServer{}
	return s
}

func start() {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	//fpga.RegisterFpgaServer(grpcServer, newServer())
	pb.RegisterTeeServer(grpcServer, newServer())
	logger.Infof("start a tee fpga mock server.")
	lis, err := net.Listen("tcp", "0.0.0.0:20000")
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatalf("grpcServer.Serve(lis) failed.")
	}
}