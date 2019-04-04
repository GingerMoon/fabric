package fpga

import (
	"context"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"time"
)

var (
	sendBlock4MvccBlockRpcWorker pb.BlockRPCClient
	sendBlock4MvccBlockRpcTaskPool chan *sendBlock4MvccBlockRpcTask
)

type sendBlock4MvccBlockRpcTask struct {
	in  *pb.BlockRequest
	out chan<- *pb.BlockReply
}

func initSendBlock4MvccBlockRpcWorkerWorker() {
	sendBlock4MvccBlockRpcWorker = pb.NewBlockRPCClient(conn)
	sendBlock4MvccBlockRpcTaskPool = make(chan *sendBlock4MvccBlockRpcTask)
}

func startSendBlock4MvccBlockRpcTaskPool() {
	go func() {
		logger.Infof("startSendBlock4MvccBlockRpcTaskPool")
		for true {
			params := <-sendBlock4MvccBlockRpcTaskPool
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			response, err := sendBlock4MvccBlockRpcWorker.SendBlockData(ctx, params.in)
			if err != nil {
				logger.Fatalf("%v.SendBlock4MvccBlockRpc(_) = _, %v: ", sendBlock4MvccBlockRpcWorker, err)
			}
			//logger.Debugf("SendBlock4MvccBlockRpc succeeded. in: %v, out: %v.", params.in, response)
			params.out <- response
			cancel()
		}
	}()
}

func SendBlock4MvccBlockRpc(in *pb.BlockRequest) *pb.BlockReply {
	//in := utils.GenerateBlock(block)
	ch := make(chan *pb.BlockReply)
	sendBlock4MvccBlockRpcTaskPool <- &sendBlock4MvccBlockRpcTask{in, ch}
	return <-ch
}
