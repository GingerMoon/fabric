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

	logger.Infof("FPGA_MOCK: ", os.Getenv("FPGA_MOCK"))
	logger.Infof("FPGA_SERVER_ADDR: ", serverAddr)
	logger.Infof("FPGA_BATCH_GEN_INTERVAL: (us)", os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	logger.Infof("FPGA_BATCH_SIZE", os.Getenv("FPGA_BATCH_SIZE"))
	logger.Infof("FPGA_RPC_CLIENT_COUNT", os.Getenv("FPGA_RPC_CLIENT_COUNT"))

	var err error
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	logger.Infof("dialed server address: %s", serverAddr)
	conn, err = grpc.Dial(serverAddr, opts...)
	if err != nil {
		logger.Fatalf("grpc.Dial(serverAddr, opts...) failed!")
	}
}