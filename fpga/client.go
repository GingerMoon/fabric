package fpga

import (
	"github.com/hyperledger/fabric/common/flogging"
	"google.golang.org/grpc"
	"os"
	"os/signal"
)

var (
	sigs chan os.Signal
	logger = flogging.MustGetLogger("fpga")
	serverAddr = os.Getenv("FPGA_SERVER_ADDR")
	conn *grpc.ClientConn
)

func init() {
	sigs = make(chan os.Signal, 1)
	signal.Notify(sigs)
	go func() {
		sig := <-sigs
		logger.Fatalf("-------------received a signal from the system: %v", sig)
	}()

	logger.Infof("FPGA_MOCK: ", os.Getenv("FPGA_MOCK"))
	logger.Infof("FPGA_SERVER_ADDR: ", serverAddr)
	logger.Infof("FPGA_BATCH_GEN_INTERVAL: (microsecond)", os.Getenv("FPGA_BATCH_GEN_INTERVAL"))

	var err error
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	logger.Infof("dialed server address: %s", serverAddr)
	conn, err = grpc.Dial(serverAddr, opts...)
	if err != nil {
		logger.Fatalf("grpc.Dial(serverAddr, opts...) failed!")
	}
}