package fpga

import (
	"container/list"
	"context"
	"fmt"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"os"
	"strconv"
	"sync"
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

	rcLock *sync.RWMutex // result-channelReturn
	cResultChs map[int] chan<-*pb.BatchReply_SignGenReply

	reqLock   *sync.RWMutex
	cRequests *list.List

	batchSize int
	interval time.Duration // milliseconds

	//gossipCount int32 // todo to be deleted. it's only for investigation purpose.
}

func (w *endorserSignWorker) start() {
	w.init()
	w.work()
}

func (w *endorserSignWorker) init() {
	w.logger = flogging.MustGetLogger("fpga.sign")
	w.batchSize = 10000

	w.rcLock = &sync.RWMutex{}
	w.cResultChs = make(map[int] chan<-*pb.BatchReply_SignGenReply)

	w.reqLock = &sync.RWMutex{}
	w.cRequests = list.New()

	tmp, err := strconv.Atoi(os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	if err != nil {
		w.logger.Fatalf("FPGA_BATCH_GEN_INTERVAL(%s)(ms) is not set correctly!, not the batch_gen_interval is set to default as 50 ms",
			os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	}
	w.interval = time.Duration(tmp)

	w.client = pb.NewBatchRPCClient(conn)
	w.taskCh = make(chan *endorserSignRpcTask, w.batchSize)
}

func (w *endorserSignWorker) work() {
	w.logger.Infof("startEndorserSignRpcTaskPool")

	//collect tasks
	go func() {
		for task := range w.taskCh {
			w.reqLock.Lock()
			w.cRequests.PushBack(task.in)
			reqId := w.cRequests.Len() - 1
			w.reqLock.Unlock()
			task.in.ReqId = fmt.Sprintf("%064d", reqId)

			w.rcLock.Lock()
			w.cResultChs[reqId] = task.out
			w.rcLock.Unlock()
		}
	}()

	// invoke the rpc every interval Microsecond
	go func() {
		var batchId uint64 = 1 // if batch_id is 0, it cannot be printed.
		for true {
			time.Sleep( w.interval * time.Microsecond)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			w.reqLock.RLock()
			size := w.cRequests.Len() // pending cRequests amount
			w.reqLock.RUnlock()

			if size > 0 {
				if size > w.batchSize {
					size = w.batchSize
				}

				var sgReqs []*pb.BatchRequest_SignGenRequest
				for i := 0; i < size; i++ {

					w.reqLock.Lock()
					element := w.cRequests.Front()
					w.cRequests.Remove(element)
					w.reqLock.Unlock()

					req := element.Value.(*pb.BatchRequest_SignGenRequest)
					if req == nil {
						w.logger.Fatalf("why req: = element.Value.(*pb.BatchRequest_SignGenRequest) failed?!!")
					}
					sgReqs = append(sgReqs, req)
				}

				w.reqLock.RLock()
				if w.cRequests.Len() > w.batchSize {
					w.logger.Warningf("current w.cRequests.Len is %d, which is bigger than the batch size(%d)", w.cRequests.Len(), w.batchSize)
				}
				w.reqLock.RUnlock()

				request := &pb.BatchRequest{SgRequests:sgReqs, BatchType:0, BatchId: batchId, ReqCount:uint32(len(sgReqs))}
				w.logger.Debugf("rpc request: %v", *request)
				response, err := w.client.Sign(ctx, request)

				// rpc failed. print the state information.
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
					w.logger.Errorf("Exiting due to the failed rpc request (the size is %d): %v", size, request)
					w.logger.Errorf("batch size: %d. interval: %d(Microseconds)", w.batchSize, w.interval)
					//w.logger.Errorf("gossip count: %d", atomic.LoadInt32(&w.gossipCount))

					w.reqLock.RLock()
					w.logger.Errorf("pending w.cRequests.Len is %d", w.cRequests.Len())
					for k, v := range w.cResultChs {
						w.logger.Errorf("w.cResultChs[%v]: %v", k, v)
					}
					w.reqLock.RUnlock()

					w.logger.Fatalf("rpc call EndorserSign failed. Will try again later. batchId: %d. err: %s", batchId, err)
				} else {
					w.logger.Debugf("rpc response: %v", *response)

					// gossip
					//w.logger.Debugf("total sign rpc cRequests: %d. gossip: %d.", len(sgReqs), atomic.LoadInt32(&w.gossipCount))
					//atomic.StoreInt32(&w.gossipCount, 0)

					// the req_id can be the same for different batch, and meanwhile, concurrent rpc is not supported by the server.
					//  so it doen't make sense to new a go routine here.
					w.parseResponse(response)
				}
			}

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

		w.rcLock.Lock()
		if err != nil || w.cResultChs[reqId] == nil {
			for k, v := range w.cResultChs {
				w.logger.Errorf("w.cResultChs[%v]: %v", k, v)
			}
			w.logger.Fatalf("the request id(%s)-(%v) in the rpc reply is not stored before.", sig.ReqId, reqId)
		}

		w.cResultChs[reqId] <- sig
		close(w.cResultChs[reqId])
		delete(w.cResultChs, reqId)
		w.rcLock.Unlock()
	}
}

func EndorserSign(in *pb.BatchRequest_SignGenRequest) *pb.BatchReply_SignGenReply {
	//gossip
	//if strings.Contains(string(debug.Stack()), "gossip") {
	//	atomic.AddInt32(&signWorker.gossipCount, 1)
	//	debug.PrintStack()
	//}

	ch := make(chan *pb.BatchReply_SignGenReply)
	signWorker.taskCh <- &endorserSignRpcTask{in, ch}
	r := <-ch
	return r
}