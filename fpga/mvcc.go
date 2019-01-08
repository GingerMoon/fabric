package fpga

import (
	"context"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"time"
)

var (
	sendBlock4MvccWorker pb.FpgaClient
	sendBlock4MvccTaskPool chan *sendBlock4MvccTask
)

type sendBlock4MvccTask struct {
	in  *pb.Block4Mvcc
	out chan<- *pb.MvccResponse
}

func initSendBlock4MvccWorkerWorker() {
	sendBlock4MvccWorker = createFpgaClient()
	sendBlock4MvccTaskPool = make(chan *sendBlock4MvccTask)
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

func SendBlock4Mvcc(in *pb.Block4Mvcc) *pb.MvccResponse {
	ch := make(chan *pb.MvccResponse)
	sendBlock4MvccTaskPool <- &sendBlock4MvccTask{in, ch}
	return <-ch
	//return nil
}
