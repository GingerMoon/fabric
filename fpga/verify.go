package fpga

import (
	"container/list"
	"context"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"sync"
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

	rcLock     *sync.Mutex
	cResultChs map[uint64] chan<-*pb.BatchReply

	cdTasksLock *sync.Cond
	cTasks      *list.List

	//gossipCount int32 // todo to be deleted. it's only for investigation purpose.
}

func (w *verifyWorker) start() {
	w.init()
	w.work()
}

func (w *verifyWorker) init() {
	w.logger = flogging.MustGetLogger("fpga.verify")

	w.client = pb.NewBatchRPCClient(conn)

	w.rcLock = &sync.Mutex{}
	w.cResultChs = make(map[uint64] chan<-*pb.BatchReply)

	w.cdTasksLock = sync.NewCond(&sync.Mutex{})
	w.cTasks = list.New()

}

func (w *verifyWorker) work() {
	w.logger.Infof("verifyWorker starts to work.")

	go func() {
		var batchId uint64 = 1 // if batch_id is 0, it cannot be printed.
		for true {
			// get task from pool and store [batchId, channel] in cResultChs
			var task *verifyRpcTask
			w.cdTasksLock.L.Lock()
			w.logger.Debugf("enter element := w.cTasks.Front()")
			for w.cTasks.Len() == 0 {
				w.cdTasksLock.Wait()
			}
			w.logger.Debugf("awaik at w.cTasks.Len() == 0")
			element := w.cTasks.Front()
			w.cTasks.Remove(element)
			task = element.Value.(*verifyRpcTask)
			if task == nil {
				w.logger.Fatalf("w.taskCh.Front().Value.(*pb.verifyRpcTask) is expected!")
			}
			w.logger.Debugf("exit element := w.cTasks.Front()")
			w.cdTasksLock.L.Unlock()

			w.rcLock.Lock()
			w.logger.Debugf("enter w.cResultChs[batchId] = task.out")
			w.cResultChs[batchId] = task.out
			w.rcLock.Unlock()
			w.logger.Debugf("exit w.cResultChs[batchId] = task.out")

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
			if err != nil {
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

				// Attention!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
				// attention! the results of cTasks and cResultChs might be not correct.
				// because they might be modified in another go routine.
				// we don't use lock to avoid the possible deadlock which is an unnecessary risk.
				for i := 0; i < w.cTasks.Len(); i++ {
					element := w.cTasks.Front()
					w.cTasks.Remove(element)
					task = element.Value.(*verifyRpcTask)
					w.logger.Errorf("pending request: %v",  task)
				}
				for k, v := range w.cResultChs {
					w.logger.Errorf("w.cResultChs[%v]: %v", k, v)
				}

				w.logger.Fatalf("rpc call EndorserVerify failed. batchId: %d. ReqCount: %d. err: %s", batchId, task.in.ReqCount, err)
			}
			w.logger.Debugf("rpc response: %v", *response)

			// gossip
			//w.logger.Debugf("total sign rpc requests: %d. gossip: %d.", len(task.in.SvRequests), atomic.LoadInt32(&w.gossipCount))
			//atomic.StoreInt32(&w.gossipCount, 0)

			cancel()
			w.parseResponse(response) // TODO this need to be changed to: go e.parseResponse(response)
			batchId++
		}
	}()
}

func (w *verifyWorker) parseResponse(response *pb.BatchReply) {
	w.rcLock.Lock()
	defer w.rcLock.Unlock()
	defer w.logger.Debugf("exit lock parseResponse")
	w.logger.Debugf("enter lock parseResponse")
	
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

	w.cdTasksLock.L.Lock()
	w.cTasks.PushFront(task)
	w.cdTasksLock.Signal()
	w.cdTasksLock.L.Unlock()

}

func (w *verifyWorker) pushBack(task *verifyRpcTask) {
	//gossip
	//if strings.Contains(string(debug.Stack()), "gossip") {
	//	atomic.AddInt32(&w.gossipCount, 1)
	//	debug.PrintStack()
	//}

	w.cdTasksLock.L.Lock()
	w.cTasks.PushBack(task)
	w.cdTasksLock.Signal()
	w.cdTasksLock.L.Unlock()

}