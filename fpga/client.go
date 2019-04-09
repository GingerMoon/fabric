package fpga

import (
	"github.com/hyperledger/fabric/common/flogging"
	"google.golang.org/grpc"
	"os"
)

var (
	logger = flogging.MustGetLogger("fpga")
	serverAddr = os.Getenv("FPGA_SERVER_ADDR")
	connSign *grpc.ClientConn
	connEndorserVerify *grpc.ClientConn
	connBlockCommitterVerify *grpc.ClientConn
)

func init() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	connSign, _ = grpc.Dial(serverAddr, opts...)
	connEndorserVerify, _ = grpc.Dial(serverAddr, opts...)
	connBlockCommitterVerify, _ = grpc.Dial(serverAddr, opts...)
}