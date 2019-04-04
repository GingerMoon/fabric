package fpga

import (
	"context"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"sync"
	"time"
)

var (
	signLogger = flogging.MustGetLogger("fpga.endorserSign")
	signWorker = endorserSignWorker{}
)

func init() {
	signWorker.start()
}

type endorserSignRpcTask struct {
	in  *pb.SignReq
	out chan<- *pb.SignResult
}

type endorserSignWorker struct {
	client      pb.BlockRPCClient
	taskPool    chan *endorserSignRpcTask

	rpcResultMap map[int32] chan<-*pb.SignResult

	m *sync.Mutex
	rpcRequests  []*pb.SignReq

	batchSize int
	interval time.Duration // milliseconds
}

func (e *endorserSignWorker) start() {
	e.init()
	e.work()
}

func (e *endorserSignWorker) init() {
	e.m = &sync.Mutex{}
	e.batchSize = 5000
	e.interval = 100
	e.rpcResultMap = make(map[int32] chan<-*pb.SignResult)

	e.client = pb.NewBlockRPCClient(conn)
	e.taskPool = make(chan *endorserSignRpcTask, e.batchSize)
}

func (e *endorserSignWorker) work() {
	signLogger.Infof("startEndorserSignRpcTaskPool")

	//collect tasks
	go func() {
		for task := range e.taskPool {
			e.m.Lock()
			e.rpcRequests = append(e.rpcRequests, task.in)
			task.in.ReqId = int32(len(e.rpcRequests)) - 1
			e.m.Unlock()
			e.rpcResultMap[task.in.ReqId] = task.out
		}
	}()

	// invoke the rpc every interval milliseconds
	go func() {
		for true {
			time.Sleep( e.interval * time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			e.m.Lock()
			if len(e.rpcRequests) > 0 {
				response, err := e.client.EndorserSign(ctx, &pb.EndorserSignRequest{SignReqs: e.rpcRequests})
				if err != nil {
					signLogger.Errorf("rpc call EndorserSign failed. Will try again later. err: %v: ", e.client, err)
				} else {
					e.parseResponse(response)
					e.rpcRequests = nil
				}
			}
			e.m.Unlock()

			cancel()
		}
	}()
}

func (e *endorserSignWorker) parseResponse(response *pb.EndorserSignReply) {
	signatures := response.SignResults
	for _, sig := range signatures {
		e.rpcResultMap[sig.ReqId] <- sig
	}
}

func EndorserSign(in *pb.SignReq) *pb.SignResult {
	ch := make(chan *pb.SignResult)
	signWorker.taskPool <- &endorserSignRpcTask{in, ch}
	return <-ch
}