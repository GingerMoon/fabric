package fpga

import (
	"container/list"
	"fmt"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ordinary work collects various verify(except the verify in block commit) into a pool and cut them every "interval" Microsecond.
var (
	ordiWorker   = verifyOrdiWorker{}
)

func init() {
	ordiWorker.start()
}


type verifyOrdiTask struct {
	in  *pb.BatchRequest_SignVerRequest
	out chan *pb.BatchReply_SignVerReply
}

type verifyOrdiWorker struct {
	logger			*flogging.FabricLogger
	taskCh          chan *verifyOrdiTask

	m               sync.Mutex
	syncResultChMap map[int] chan<-*pb.BatchReply_SignVerReply // TODO maybe we need to change another container for storing the resp ch.
	syncSvReqList *list.List

	// rpc call EndorserVerify failed. batchId: 1763. ReqCount: 17837. err: rpc error: code = ResourceExhausted desc = grpc: received message larger than max (4262852 vs. 4194304)
	// log: Exiting due to the failed rpc request: batch_id:1763 batch_type:1 req_count:17837
	interval time.Duration // milliseconds
	batchSize     int

	gossipCount int // todo to be deleted. it's only for investigation purpose.
}

func (w *verifyOrdiWorker) start() {
	w.init()
	w.work()
}

func (w *verifyOrdiWorker) init() {
	w.logger = flogging.MustGetLogger("fpga.ordiVerify")
	w.m = sync.Mutex{}
	w.batchSize = 10000
	w.syncSvReqList = list.New()

	tmp, err := strconv.Atoi(os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	if err != nil {
		w.logger.Fatalf("FPGA_BATCH_GEN_INTERVAL(%s)(ms) is not set correctly!, not the batch_gen_interval is set to default as 50 ms",
			os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	}
	w.interval = time.Duration(tmp)

	w.syncResultChMap = make(map[int] chan<-*pb.BatchReply_SignVerReply)

	w.taskCh = make(chan *verifyOrdiTask)
}

func (w *verifyOrdiWorker) work() {
	w.logger.Infof("ordinary verify requests collector starts to work.")

	// collect tasks
	go func() {
		for task := range w.taskCh {
			w.m.Lock()
			w.logger.Debugf("enter lock w.syncSvReqList = append(w.syncSvReqList, task.in)")
			w.syncSvReqList.PushBack(task.in)
			reqId := w.syncSvReqList.Len() - 1
			task.in.ReqId = fmt.Sprintf("%064d", reqId)
			w.syncResultChMap[reqId] = task.out
			w.logger.Debugf("exit lock w.syncSvReqList = append(w.syncSvReqList, task.in)")
			w.m.Unlock()
		}
	}()

	// invoke the rpc every interval Microsecond
	go func() {
		var batchId uint64 = 0
		for true {
			time.Sleep( w.interval * time.Microsecond)

			w.m.Lock()
			w.logger.Debugf("enter lock for w.syncSvReqList = nil")
			if w.syncSvReqList.Len() > 0 {
				var svReqs []*pb.BatchRequest_SignVerRequest
				for i := 0; i < w.batchSize; i++ {
					element := w.syncSvReqList.Front()
					w.syncSvReqList.Remove(element)
					req := element.Value.(*pb.BatchRequest_SignVerRequest)
					if req == nil {
						w.logger.Fatalf("why req: = element.Value.(*pb.BatchRequest_SignVerRequest) failed?!!")
					}
					svReqs = append(svReqs, req)
				}
				if w.syncSvReqList.Len() > w.batchSize {
					w.logger.Warningf("current w.syncSvReqList.Len is %d, which is bigger than the batch size(%d)", w.syncSvReqList.Len(), w.batchSize)
				}

				in := &pb.BatchRequest{SvRequests:svReqs}
				out := make(chan *pb.BatchReply)
				task := &verifyRpcTask{in, out}
				vk.pushBack(task)

				// parse rpc response
				for response := range out {
					w.logger.Debugf("total verify rpc requests: %d. gossip: %d.", w.syncSvReqList.Len(), w.gossipCount)
					w.parseResponse(response)
					w.syncSvReqList = nil
					w.gossipCount = 0
				}
			}
			w.logger.Debugf("exit lock for w.syncSvReqList = nil")
			w.m.Unlock()
			batchId++
		}
	}()
}

// this method need to be locked where it is invoked.
func (w *verifyOrdiWorker) parseResponse(response *pb.BatchReply) {
	verifyResults := response.SvReplies
	for _, result := range verifyResults {
		reqId, err := strconv.Atoi(result.ReqId)
		if err != nil || w.syncResultChMap[reqId] == nil {
			w.logger.Fatalf("[verifyOrdiWorker] the request id(%s) in the rpc reply is not stored before.", result)
		}

		w.syncResultChMap[reqId] <- result
	}
}

func (w *verifyOrdiWorker) putToTaskCh(task *verifyOrdiTask){
	w.taskCh <- task
}

func EndorserVerify(in *pb.BatchRequest_SignVerRequest) bool {
	if strings.Contains(string(debug.Stack()), "gossip") {
		ordiWorker.gossipCount++
	}

	logger.Debugf("EndorserVerify is invoking verify rpc...")
	ch := make(chan *pb.BatchReply_SignVerReply)
	ordiWorker.putToTaskCh(&verifyOrdiTask{in, ch})
	r := <-ch
	logger.Debugf("EndorserVerify finished invoking verify rpc. result: %v", r)
	return r.Verified
}