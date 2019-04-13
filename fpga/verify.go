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

	m               *sync.Mutex
	syncBatchIdResp map[uint64] chan<-*pb.BatchReply

	c               *sync.Cond
	syncTaskPool    *list.List

	//gossipCount int32 // todo to be deleted. it's only for investigation purpose.
}

func (w *verifyWorker) start() {
	w.init()
	w.work()
}

func (w *verifyWorker) init() {
	w.logger = flogging.MustGetLogger("fpga.verify")

	w.client = pb.NewBatchRPCClient(conn)

	w.m = &sync.Mutex{}
	w.syncBatchIdResp = make(map[uint64] chan<-*pb.BatchReply)

	w.c = sync.NewCond(&sync.Mutex{})
	w.syncTaskPool = list.New()

}

func (w *verifyWorker) work() {
	w.logger.Infof("verifyWorker starts to work.")

	go func() {
		var batchId uint64 = 1 // if batch_id is 0, it cannot be printed.
		for true {
			// get task from pool and store [batchId, channel] in syncBatchIdResp
			var task *verifyRpcTask
			w.c.L.Lock()
			w.logger.Debugf("enter element := w.syncTaskPool.Front()")
			for w.syncTaskPool.Len() == 0 {
				w.c.Wait()
			}
			w.logger.Debugf("awaik at w.syncTaskPool.Len() == 0")
			element := w.syncTaskPool.Front()
			w.syncTaskPool.Remove(element)
			task = element.Value.(*verifyRpcTask)
			if task == nil {
				w.logger.Fatalf("w.taskCh.Front().Value.(*pb.verifyRpcTask) is expected!")
			}
			w.logger.Debugf("exit element := w.syncTaskPool.Front()")
			w.c.L.Unlock()

			w.m.Lock()
			w.logger.Debugf("enter w.syncBatchIdResp[batchId] = task.out")
			w.syncBatchIdResp[batchId] = task.out
			w.m.Unlock()
			w.logger.Debugf("exit w.syncBatchIdResp[batchId] = task.out")

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
				// attention! the results of syncTaskPool and syncBatchIdResp might be not correct.
				// because they might be modified in another go routine.
				// we don't use lock to avoid the possible deadlock which is an unnecessary risk.
				for i := 0; i < w.syncTaskPool.Len(); i++ {
					element := w.syncTaskPool.Front()
					w.syncTaskPool.Remove(element)
					task = element.Value.(*verifyRpcTask)
					w.logger.Errorf("pending request: %v",  task)
				}
				for k, v := range w.syncBatchIdResp {
					w.logger.Errorf("w.rpcResultMap[%v]: %v", k, v)
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
	w.m.Lock()
	defer w.m.Unlock()
	defer w.logger.Debugf("exit lock parseResponse")
	w.logger.Debugf("enter lock parseResponse")
	
	if w.syncBatchIdResp[response.BatchId] == nil {
		for k, v := range w.syncBatchIdResp {
			w.logger.Errorf("w.syncBatchIdResp[%v]: %v", k, v)
		}
		w.logger.Fatalf("w.syncBatchIdResp[response.BatchId] is nil! k: %v", response.BatchId)
	}
	w.syncBatchIdResp[response.BatchId] <- response
	close(w.syncBatchIdResp[response.BatchId])
	delete(w.syncBatchIdResp, response.BatchId)

}

func (w *verifyWorker) pushFront(task *verifyRpcTask) {
	//gossip
	//if strings.Contains(string(debug.Stack()), "gossip") {
	//	atomic.AddInt32(&w.gossipCount, 1)
	//	debug.PrintStack()
	//}

	w.m.Lock()
	w.syncTaskPool.PushFront(task)
	w.m.Unlock()

	w.c.Signal()
}

func (w *verifyWorker) pushBack(task *verifyRpcTask) {
	//gossip
	//if strings.Contains(string(debug.Stack()), "gossip") {
	//	atomic.AddInt32(&w.gossipCount, 1)
	//	debug.PrintStack()
	//}

	w.m.Lock()
	w.syncTaskPool.PushBack(task)
	w.m.Unlock()

	w.c.Signal()
}