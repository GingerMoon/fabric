package fpga

import (
	"context"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"time"
	"unsafe"
)

var (
	vk = verifyWorker{}
)

func init() {
	vk.start()
}

type verifyRpcTask struct {
	in  *pb.BatchRequest
	out chan *pb.BatchReply
}

type verifyWorker struct {
	logger			*flogging.FabricLogger
	client      	pb.BatchRPCClient

	taskCh chan *verifyRpcTask
	cResultChs map[uint64] chan<-*pb.BatchReply

	//gossipCount int32 // todo to be deleted. it's only for investigation purpose.
}

func (w *verifyWorker) start() {
	w.init()
	w.work()
}

func (w *verifyWorker) init() {
	w.logger = flogging.MustGetLogger("fpga.verify")
	w.client = pb.NewBatchRPCClient(conn)
	w.taskCh = make(chan *verifyRpcTask)
	w.cResultChs = make(map[uint64] chan<-*pb.BatchReply)
}

func (w *verifyWorker) work() {
	w.logger.Infof("verifyWorker starts to work.")

	go func() {
		var batchId uint64 = 1 // if batch_id is 0, it cannot be printed.
		for task := range w.taskCh {
			w.cResultChs[batchId] = task.out

			// prepare rpc parameter
			task.in.BatchId = batchId
			task.in.BatchType = 1
			task.in.ReqCount = uint32(len(task.in.SvRequests))
			if len(task.in.SvRequests) == 0 {
				w.logger.Fatalf("why len(task.in.SvRequests) is 0?")
			}

			// invoke the rpc
			w.logger.Debugf("rpc request: %v", *task.in)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			response, err := w.client.Verify(ctx, task.in)

			// rpc failed, print the state information.
			if err != nil {
				w.logger.Errorf("rpc call EndorserVerify failed. batchId: %d. ReqCount: %d. err: %s", batchId, task.in.ReqCount, err)

				var size uintptr = 0
				for _, req := range task.in.SvRequests {
					size += unsafe.Sizeof(req.SignR)
					size += unsafe.Sizeof(req.SignS)
					size += unsafe.Sizeof(req.Px)
					size += unsafe.Sizeof(req.Py)
					size += unsafe.Sizeof(req.Hash)
					size += unsafe.Sizeof(req.ReqId)
					size += unsafe.Sizeof(req.XXX_NoUnkeyedLiteral)
					size += unsafe.Sizeof(req.XXX_sizecache)
					size += unsafe.Sizeof(req.XXX_unrecognized)
				}
				w.logger.Errorf("Exiting due to the failed rpc request (the size is %d): %v", size, task.in)
				//w.logger.Errorf("gossip count: %d", atomic.LoadInt32(&w.gossipCount))

				for k, v := range w.cResultChs {
					w.logger.Errorf("w.cResultChs[%v]: %v", k, v)
				}

				w.logger.Fatalf("rpc call EndorserVerify failed. batchId: %d. ReqCount: %d. err: %s", batchId, task.in.ReqCount, err)
			}
			w.logger.Debugf("rpc response: %v", *response)

			// gossip
			//w.logger.Debugf("total sign rpc cRequests: %d. gossip: %d.", len(task.in.SvRequests), atomic.LoadInt32(&w.gossipCount))
			//atomic.StoreInt32(&w.gossipCount, 0)

			cancel()
			// the req_id can be the same for different batch, and meanwhile, concurrent rpc is not supported by the server.
			//  so it doen't make sense to new a go routine here.
			w.parseResponse(response)
			batchId++
		}
	}()
}

func (w *verifyWorker) parseResponse(response *pb.BatchReply) {
	if w.cResultChs[response.BatchId] == nil {
		for k, v := range w.cResultChs {
			w.logger.Errorf("w.cResultChs[%v]: %v", k, v)
		}
		w.logger.Fatalf("w.cResultChs[response.BatchId] is nil! k: %v", response.BatchId)
	}
	w.cResultChs[response.BatchId] <- response
	close(w.cResultChs[response.BatchId])
	delete(w.cResultChs, response.BatchId)
}

func (w *verifyWorker) pushFront(task *verifyRpcTask) {
	//gossip
	//if strings.Contains(string(debug.Stack()), "gossip") {
	//	atomic.AddInt32(&w.gossipCount, 1)
	//	debug.PrintStack()
	//}
	w.taskCh <- task
}

func (w *verifyWorker) pushBack(task *verifyRpcTask) {
	//gossip
	//if strings.Contains(string(debug.Stack()), "gossip") {
	//	atomic.AddInt32(&w.gossipCount, 1)
	//	debug.PrintStack()
	//}
	w.taskCh <- task
}