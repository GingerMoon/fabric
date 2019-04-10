package fpga

import (
	"github.com/hyperledger/fabric/common/flogging"
	"google.golang.org/grpc"
	"os"
)

var (
	logger = flogging.MustGetLogger("fpga")
	serverAddr = os.Getenv("FPGA_SERVER_ADDR")
	conn *grpc.ClientConn
)

func init() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, _ = grpc.Dial(serverAddr, opts...)
}