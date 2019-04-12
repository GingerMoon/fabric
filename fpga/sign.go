package fpga

import (
	"context"
	"fmt"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
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

	gossipCount int32 // todo to be deleted. it's only for investigation purpose.
}

func (w *endorserSignWorker) start() {
	w.init()
	w.work()
}

func (w *endorserSignWorker) init() {
	w.logger = flogging.MustGetLogger("fpga.sign")
	w.m = &sync.Mutex{}
	w.batchSize = 1500
	tmp, err := strconv.Atoi(os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	if err != nil {
		w.logger.Fatalf("FPGA_BATCH_GEN_INTERVAL(%s)(ms) is not set correctly!, not the batch_gen_interval is set to default as 50 ms",
			os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	}
	w.interval = time.Duration(tmp)

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

	// invoke the rpc every interval Microsecond
	go func() {
		var batchId uint64 = 1 // if batch_id is 0, it cannot be printed.
		for true {
			time.Sleep( w.interval * time.Microsecond)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			// rpc
			w.m.Lock()
			if len(w.rpcRequests) > 0 {
				request := &pb.BatchRequest{SgRequests:w.rpcRequests, BatchType:0, BatchId: batchId, ReqCount:uint32(len(w.rpcRequests))}
				w.logger.Debugf("rpc request: %v", *request)
				response, err := w.client.Sign(ctx, request)
				if err != nil {
					var size uintptr = 0
					for _, req := range request.SgRequests {
						size += unsafe.Sizeof(req.D)
						size += unsafe.Sizeof(req.Hash)
						size += unsafe.Sizeof(req.ReqId)
						size += unsafe.Sizeof(req.XXX_NoUnkeyedLiteral)
						size += unsafe.Sizeof(req.XXX_sizecache)
						size += unsafe.Sizeof(req.XXX_unrecognized)
					}
					w.logger.Errorf("Exiting due to the failed rpc request(the size is %d): %v", size, request)
					w.logger.Errorf("batch size: %d. interval: %d(Microseconds)", w.batchSize, w.interval)
					w.logger.Errorf("gossip count: %d", atomic.LoadInt32(&w.gossipCount))

					// Attention!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
					// attention! the results of rpcRequests and rpcResultMap might be not correct.
					// because they might be modified in another go routine.
					// we don't use lock to avoid the possible deadlock which is an unnecessary risk.
					for _, req := range w.rpcRequests {
						w.logger.Errorf("pending request: %v", req)
					}
					for k, v := range w.rpcResultMap {
						w.logger.Errorf("w.rpcResultMap[%v]: %v", k, v)
					}

					w.logger.Fatalf("rpc call EndorserSign failed. Will try again later. batchId: %d. err: %s", batchId, err)
				} else {
					w.logger.Debugf("rpc response: %v", *response)

					// gossip
					w.logger.Debugf("total sign rpc requests: %d. gossip: %d.", len(w.rpcRequests), atomic.LoadInt32(&w.gossipCount))
					atomic.StoreInt32(&w.gossipCount, 0)

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
	//gossip
	if strings.Contains(string(debug.Stack()), "gossip") {
		atomic.AddInt32(&signWorker.gossipCount, 1)
		debug.PrintStack()
	}

	signWorker.logger.Debugf("EndorserSign is invoking sign rpc...")
	ch := make(chan *pb.BatchReply_SignGenReply)
	signWorker.taskCh <- &endorserSignRpcTask{in, ch}
	r := <-ch
	signWorker.logger.Debugf("EndorserSign finished invoking sign rpc. result: %v", r)
	return r
}