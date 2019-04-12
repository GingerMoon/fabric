package fpga

import (
	"context"
	"fmt"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	signWorker = endorserSignWorker{}
)

func init() {
	signWorker.start()
}

type endorserSignRpcTask struct {
	in  *pb.BatchRequest_SignGenRequest
	out chan *pb.BatchReply_SignGenReply
}

type endorserSignWorker struct {
	logger *flogging.FabricLogger
	client pb.BatchRPCClient
	taskCh chan *endorserSignRpcTask

	rpcResultMap map[int] chan<-*pb.BatchReply_SignGenReply

	m *sync.Mutex
	rpcRequests  []*pb.BatchRequest_SignGenRequest

	batchSize int
	interval time.Duration // milliseconds

	gossipCount int // todo to be deleted. it's only for investigation purpose.
}

func (w *endorserSignWorker) start() {
	w.init()
	w.work()
}

func (w *endorserSignWorker) init() {
	w.logger = flogging.MustGetLogger("fpga.sign")
	w.m = &sync.Mutex{}
	w.batchSize = 5000
	w.interval = 100
	w.rpcResultMap = make(map[int] chan<-*pb.BatchReply_SignGenReply)

	w.client = pb.NewBatchRPCClient(conn)
	w.taskCh = make(chan *endorserSignRpcTask, w.batchSize)
}

func (w *endorserSignWorker) work() {
	w.logger.Infof("startEndorserSignRpcTaskPool")

	//collect tasks
	go func() {
		for task := range w.taskCh {
			w.m.Lock()
			w.rpcRequests = append(w.rpcRequests, task.in)
			reqId := len(w.rpcRequests) - 1
			task.in.ReqId = fmt.Sprintf("%064d", reqId)
			w.m.Unlock()
			w.rpcResultMap[reqId] = task.out
		}
	}()

	// invoke the rpc every interval milliseconds
	go func() {
		var batchId uint64 = 1 // if batch_id is 0, it cannot be printed.
		for true {
			time.Sleep( w.interval * time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			w.m.Lock()
			if len(w.rpcRequests) > 0 {
				request := &pb.BatchRequest{SgRequests:w.rpcRequests, BatchType:0, BatchId: batchId, ReqCount:uint32(len(w.rpcRequests))}
				w.logger.Debugf("rpc request: %v", *request)
				response, err := w.client.Sign(ctx, request)
				if err != nil {
					w.logger.Fatalf("rpc call EndorserSign failed. Will try again later. batchId: %d. err: %s", batchId, err)
				} else {
					w.logger.Debugf("rpc response: %v", *response)
					w.logger.Debugf("total sign rpc requests: %d. gossip: %d.", len(w.rpcRequests), w.gossipCount)
					w.gossipCount = 0

					w.parseResponse(response) // TODO this need to be changed to: go e.parseResponse(response)
					w.rpcRequests = nil
				}
			}
			w.m.Unlock()

			cancel()
			batchId++
		}
	}()
}

// this method need to be locked where it is invoked.
func (w *endorserSignWorker) parseResponse(response *pb.BatchReply) {
	signatures := response.SgReplies
	for _, sig := range signatures {
		reqId, err := strconv.Atoi(sig.ReqId)
		if err != nil || w.rpcResultMap[reqId] == nil {
			for k, v := range w.rpcResultMap {
				w.logger.Errorf("w.rpcResultMap[%v]: %v", k, v)
			}
			w.logger.Fatalf("[endorserSignWorker] the request id(%s) in the rpc reply is not stored before.", sig.ReqId)
		}

		w.rpcResultMap[reqId] <- sig
		close(w.rpcResultMap[reqId])
		delete(w.rpcResultMap, reqId)

	}
}

func EndorserSign(in *pb.BatchRequest_SignGenRequest) *pb.BatchReply_SignGenReply {
	if strings.Contains(string(debug.Stack()), "gossip") {
		signWorker.gossipCount++
		debug.PrintStack()
	}

	logger.Infof("EndorserSign is invoking sign rpc...")
	ch := make(chan *pb.BatchReply_SignGenReply)
	signWorker.taskCh <- &endorserSignRpcTask{in, ch}
	r := <-ch
	logger.Infof("EndorserSign finished invoking sign rpc. r: %v")
	return r
}