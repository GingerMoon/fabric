package fpga

import (
	"context"
	"fmt"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"strconv"
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
	in  *pb.BatchRequest_SignGenRequest
	out chan<- *pb.BatchReply_SignGenReply
}

type endorserSignWorker struct {
	client      pb.BatchRPCClient
	taskPool    chan *endorserSignRpcTask

	rpcResultMap map[int] chan<-*pb.BatchReply_SignGenReply

	m *sync.Mutex
	rpcRequests  []*pb.BatchRequest_SignGenRequest

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
	e.rpcResultMap = make(map[int] chan<-*pb.BatchReply_SignGenReply)

	e.client = pb.NewBatchRPCClient(connSign)
	e.taskPool = make(chan *endorserSignRpcTask, e.batchSize)
}

func (e *endorserSignWorker) work() {
	signLogger.Infof("startEndorserSignRpcTaskPool")

	//collect tasks
	go func() {
		for task := range e.taskPool {
			e.m.Lock()
			e.rpcRequests = append(e.rpcRequests, task.in)
			reqId := len(e.rpcRequests) - 1
			task.in.ReqId = fmt.Sprintf("%064x", reqId)
			e.m.Unlock()
			e.rpcResultMap[reqId] = task.out
		}
	}()

	// invoke the rpc every interval milliseconds
	go func() {
		var batchId uint64 = 0
		for true {
			time.Sleep( e.interval * time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			e.m.Lock()
			if len(e.rpcRequests) > 0 {
				response, err := e.client.Sign(ctx, &pb.BatchRequest{SgRequests:e.rpcRequests, BatchType:0, BatchId: batchId, ReqCount:uint32(len(e.rpcRequests))})
				if err != nil {
					signLogger.Fatalf("rpc call EndorserSign failed. Will try again later. batchId: %d. err: %s", batchId, err)
				} else {
					e.parseResponse(response)
					e.rpcRequests = nil
				}
			}
			e.m.Unlock()

			cancel()
			batchId++
		}
	}()
}

func (e *endorserSignWorker) parseResponse(response *pb.BatchReply) {
	signatures := response.SgReplies
	for _, sig := range signatures {
		reqId, err := strconv.Atoi(sig.ReqId)
		if err != nil || e.rpcResultMap[reqId] == nil {
			logger.Fatalf("[endorserSignWorker] the request id(%s) in the rpc reply is not stored before.", sig.ReqId)
		}

		e.rpcResultMap[reqId] <- sig
	}
}

func EndorserSign(in *pb.BatchRequest_SignGenRequest) *pb.BatchReply_SignGenReply {
	ch := make(chan *pb.BatchReply_SignGenReply)
	signWorker.taskPool <- &endorserSignRpcTask{in, ch}
	return <-ch
}