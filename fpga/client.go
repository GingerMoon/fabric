package fpga

import (
	"context"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"github.com/spf13/viper"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"time"
)

var (
	logger = flogging.MustGetLogger("fpga")

	serverAddr = "127.0.0.1:10000"

	/*
		for the time being, we only consider the scenario of 1 channel
		since fabric handles blocks one by one (just as the name "block-chain" implies),
		there is only 1 sendBlockSizeWorker and sendBlock4MvccWorker.
		since fabric handles verify sig of txs in the block concurrently,
		there are multiple verifySigWorkers
	*/
	verifySigWorkers     []pb.FpgaClient
	sendBlock4MvccWorker pb.FpgaClient

	verifySigWorkersSemaphore *semaphore.Weighted // used for verifySigTaskPool

	verifySigTaskPool      chan *verifyTask
	sendBlock4MvccTaskPool chan *sendBlock4MvccTask
)

func init() {
	initVerifySigWorkers()
	startVerifySigTaskPool()

	initSendBlock4MvccWorkerWorker()
	startSendBlock4MvccTaskPool()
}

type verifyTask struct {
	in  *pb.VsccEnvelope
	out chan<- *pb.VsccResponse
}

type sendBlock4MvccTask struct {
	in  *pb.Block4Mvcc
	out chan<- *pb.MvccResponse
}

func createFpgaClient() pb.FpgaClient {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		logger.Fatalf("fail to dial: %v", err)
	}
	//defer conn.Close()
	return pb.NewFpgaClient(conn)
}

func initSendBlock4MvccWorkerWorker() {
	sendBlock4MvccWorker = createFpgaClient()
	sendBlock4MvccTaskPool = make(chan *sendBlock4MvccTask)
}

func initVerifySigWorkers() {
	nWorkers := viper.GetInt("peer.validatorPoolSize")
	if nWorkers == 0 {
		nWorkers = 12 // we do have 12 ECDSA engines on HW
	}
	logger.Infof("peer.validatorPoolSize is: %d", nWorkers)
	verifySigWorkers = make([]pb.FpgaClient, nWorkers)
	for i := 0; i < len(verifySigWorkers); i++ {
		verifySigWorkers[i] = createFpgaClient()
	}
	verifySigTaskPool = make(chan *verifyTask, nWorkers)
	verifySigWorkersSemaphore = semaphore.NewWeighted(int64(nWorkers))
}

func startVerifySigTaskPool() {
	for i := 0; i < len(verifySigWorkers); i++ {
		ii := i
		go func() {
			logger.Infof("startVerifySigTaskPool")
			for {
				params := <-verifySigTaskPool
				// ensure that we don't have too many concurrent verify workers
				verifySigWorkersSemaphore.Acquire(context.Background(), 1)
				go func() {
					defer verifySigWorkersSemaphore.Release(1)
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					response, err := verifySigWorkers[ii].VerifySig4Vscc(ctx, params.in)
					if err != nil {
						logger.Fatalf("%v.VerifySig4Vscc(_) = _, %v: ", verifySigWorkers[i], err)
					}
					logger.Debugf("VerifySig4Vscc succeeded. in: %v, out: %s.", params.in, response.String())
					params.out <- response
				}()
			}
		}()
	}
}

func startSendBlock4MvccTaskPool() {
	go func() {
		logger.Infof("startSendBlock4MvccTaskPool")
		for true {
			params := <-sendBlock4MvccTaskPool
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			response, err := sendBlock4MvccWorker.SendBlock4Mvcc(ctx, params.in)
			if err != nil {
				logger.Fatalf("%v.SendBlock4Mvcc(_) = _, %v: ", sendBlock4MvccWorker, err)
			}
			logger.Debugf("SendBlock4Mvcc succeeded. in: %v, out: %v.", params.in, response)
			params.out <- response
		}
	}()
}

func VerifySig4Vscc(in *pb.VsccEnvelope) *pb.VsccResponse {
	ch := make(chan *pb.VsccResponse)
	verifySigTaskPool <- &verifyTask{in, ch}
	return <-ch
	//return nil
}

func SendBlock4Mvcc(in *pb.Block4Mvcc) *pb.MvccResponse {
	ch := make(chan *pb.MvccResponse)
	sendBlock4MvccTaskPool <- &sendBlock4MvccTask{in, ch}
	return <-ch
	//return nil
}
