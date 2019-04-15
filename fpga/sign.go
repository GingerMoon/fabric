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


	reqLock   *sync.Mutex
	cRequests *list.List

	rcLock *sync.Mutex // result-channelReturn
	cResultChs map[int] chan<-*pb.BatchReply_SignGenReply

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

	w.rcLock = &sync.Mutex{}
	w.cResultChs = make(map[int] chan<-*pb.BatchReply_SignGenReply)

	w.reqLock = &sync.Mutex{}
	w.cRequests = list.New()

	tmp, err := strconv.Atoi(os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	if err != nil {
		w.logger.Fatalf("FPGA_BATCH_GEN_INTERVAL(%s)(ms) is not set correctly!, not the batch_gen_interval is set to default as 50 ms",
			os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	}
	w.interval = time.Duration(tmp)

	w.client = pb.NewBatchRPCClient(conn)
	w.taskCh = make(chan *endorserSignRpcTask)
}

func (w *endorserSignWorker) work() {
	w.logger.Infof("startEndorserSignRpcTaskPool")

	//collect tasks
	go func() {
		reqId := 1 // starts from 1 instead of 0, because the driver cannot print 0.
		for task := range w.taskCh {
			w.reqLock.Lock()
			w.cRequests.PushBack(task.in)
			task.in.ReqId = fmt.Sprintf("%064d", reqId)
			w.reqLock.Unlock()

			w.rcLock.Lock()
			w.cResultChs[reqId] = task.out
			w.rcLock.Unlock()
			reqId++
		}
	}()

	// invoke the rpc every interval Microsecond
	go func() {
		stream, err := w.client.Sign(context.Background())
		if err != nil {
			w.logger.Fatalf("w.client.Sign(context.Background()) failed! err: %v", err)
		}

		var batchId uint64 = 1 // if batch_id is 0, it cannot be printed.
		for true {
			time.Sleep( w.interval * time.Microsecond)

			w.reqLock.Lock()

			size := w.cRequests.Len() // pending cRequests amount
			if size > 0 {
				if size > w.batchSize {
					size = w.batchSize
				}

				var sgReqs []*pb.BatchRequest_SignGenRequest
				for i := 0; i < size; i++ {

					element := w.cRequests.Front()
					w.cRequests.Remove(element)

					req := element.Value.(*pb.BatchRequest_SignGenRequest)
					if req == nil {
						w.logger.Fatalf("why req: = element.Value.(*pb.BatchRequest_SignGenRequest) failed?!!")
					}
					sgReqs = append(sgReqs, req)
				}

				if w.cRequests.Len() > w.batchSize {
					w.logger.Warningf("current w.cRequests.Len is %d, which is bigger than the batch size(%d)", w.cRequests.Len(), w.batchSize)
				}

				request := &pb.BatchRequest{SgRequests:sgReqs, BatchType:0, BatchId: batchId, ReqCount:uint32(len(sgReqs))}
				err = stream.Send(request)
				// rpc failed. print the state information.
				if err != nil {
					w.dump(request)
					w.logger.Fatalf("stream.Send(request) failed. batchId: %d. err: %s", batchId, err)
				}

				response, err := stream.Recv()
				if err != nil {
					w.dump(request)
					w.logger.Fatalf("stream.Send(request) failed. batchId: %d. err: %s", batchId, err)
				}
				w.logger.Debugf("rpc response: %v", *response)

				// gossip
				//w.logger.Debugf("total sign rpc cRequests: %d. gossip: %d.", len(sgReqs), atomic.LoadInt32(&w.gossipCount))
				//atomic.StoreInt32(&w.gossipCount, 0)

				// the req_id can be the same for different batch, and meanwhile, concurrent rpc is not supported by the server.
				//  so it doen't make sense to new a go routine here.
				w.parseResponse(response)
			}
			w.reqLock.Unlock()
			batchId++
		}
	}()
}

func (w *endorserSignWorker) dump(request *pb.BatchRequest) {
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

	w.logger.Errorf("it's only a dump. some state data is not thread safe.")
	w.logger.Errorf("pending w.cRequests.Len is %d", w.cRequests.Len())
	for k, v := range w.cResultChs {
		w.logger.Errorf("w.cResultChs[%v]: %v", k, v)
	}
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