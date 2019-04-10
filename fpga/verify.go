package fpga

import (
	"container/list"
	"context"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"sync"
	"time"
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
		var batchId uint64 = 0
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
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			response, err := w.client.Verify(ctx, task.in)
			if err != nil {
				w.logger.Fatalf("rpc call EndorserVerify failed. batchId: %d. ReqCount: %d. err: %s", batchId, task.in.ReqCount, err)
			}
			cancel()
			w.parseResponse(response)
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
		w.logger.Fatalf("w.syncBatchIdResp[response.BatchId] is nil! k: %v", response.BatchId)
	}
	w.syncBatchIdResp[response.BatchId] <- response
	close(w.syncBatchIdResp[response.BatchId])
	delete(w.syncBatchIdResp, response.BatchId)

}

func (w *verifyWorker) pushFront(task *verifyRpcTask) {
	w.m.Lock()
	w.logger.Debugf("enter pushFront")
	defer w.m.Unlock()
	defer w.logger.Debugf("exit pushFront")
	w.syncTaskPool.PushFront(task)
	w.c.Signal()
}

func (w *verifyWorker) pushBack(task *verifyRpcTask) {
	w.m.Lock()
	w.logger.Debugf("enter pushBack")
	defer w.m.Unlock()
	defer w.logger.Debugf("exit pushBack")
	w.syncTaskPool.PushBack(task)
	w.c.Signal()
}