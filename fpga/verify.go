package fpga

import (
	"context"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"sync"
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
	cResultChs sync.Map // map[uint64] chan<-*pb.BatchReply // this member is not accessed by multiple go routines.

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
}

func (w *verifyWorker) work() {
	w.logger.Infof("verifyWorker starts to work.")

	go func() {
		stream, err := w.client.Verify(context.Background())
		if err != nil {
			w.logger.Fatalf("w.client.Verify(context.Background()) failed! err: %v", err)
		}

		var batchId uint64 = 1 // if batch_id is 0, it cannot be printed.
		for task := range w.taskCh {
			w.cResultChs.Store(batchId, task.out)

			// prepare rpc parameter
			task.in.BatchId = batchId
			task.in.BatchType = 1
			task.in.ReqCount = uint32(len(task.in.SvRequests))
			if len(task.in.SvRequests) == 0 {
				w.logger.Fatalf("why len(task.in.SvRequests) is 0?")
			}

			// invoke the rpc
			w.logger.Debugf("rpc request: %v", *task.in)
			err := stream.Send(task.in)
			// rpc failed, print the state information.
			if err != nil {
				w.dump(task.in)
				w.logger.Fatalf("rpc call EndorserVerify failed. batchId: %d. ReqCount: %d. err: %s", batchId, task.in.ReqCount, err)
			}

			response, err := stream.Recv()
			if err != nil {
				w.dump(task.in)
				w.logger.Fatalf("stream.Send(request) failed. batchId: %d. err: %s", batchId, err)
			}
			w.logger.Debugf("rpc response: %v", *response)

			// gossip
			//w.logger.Debugf("total sign rpc cRequests: %d. gossip: %d.", len(task.in.SvRequests), atomic.LoadInt32(&w.gossipCount))
			//atomic.StoreInt32(&w.gossipCount, 0)

			// the req_id can be the same for different batch, and meanwhile, concurrent rpc is not supported by the server.
			//  so it doen't make sense to new a go routine here.
			go w.parseResponse(response)
			batchId++
		}
	}()
}

func (w *verifyWorker) dump(request *pb.BatchRequest) {
	var size uintptr = 0
	for _, req := range request.SvRequests {
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
	w.logger.Errorf("Exiting due to the failed rpc request (the size is %d): %v", size, request)
	//w.logger.Errorf("gossip count: %d", atomic.LoadInt32(&w.gossipCount))

	w.logger.Errorf("it's only a dump. some state data is not thread safe.")
	//for k, v := range w.cResultChs {
	//	w.logger.Errorf("w.cResultChs[%v]: %v", k, v)
	//}

}

func (w *verifyWorker) parseResponse(response *pb.BatchReply) {
	v, ok := w.cResultChs.Load(response.BatchId)
	if !ok {
		//for k, v := range w.cResultChs {
		//	w.logger.Errorf("w.cResultChs[%v]: %v", k, v)
		//}
		w.logger.Fatalf("w.cResultChs[response.BatchId] is nil! k: %v", response.BatchId)
	}

	outCh := v.(chan *pb.BatchReply)
	if outCh == nil {
		w.logger.Fatalf("failed to find w.cResultChs[response.BatchId]! k: %v", response.BatchId)
	}
	outCh <- response
	close(outCh)
	w.cResultChs.Delete(response.BatchId)
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