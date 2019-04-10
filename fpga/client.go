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
	err error
)

func init() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	logger.Infof("dialed server address: %s", serverAddr)
	conn, err = grpc.Dial(serverAddr, opts...)
	if err != nil {
		logger.Fatalf("grpc.Dial(serverAddr, opts...) failed!")
	}
}