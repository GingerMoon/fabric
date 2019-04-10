package fpga

import (
	"fmt"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ordinary work collects various verify(except the verify in block commit) into a pool and cut them every "interval" seconds.
var (
	ordiWorker   = verifyOrdiWorker{}
)

func init() {
	ordiWorker.start()
}


type verifyOrdiTask struct {
	in  *pb.BatchRequest_SignVerRequest
	out chan<- *pb.BatchReply_SignVerReply
}

type verifyOrdiWorker struct {
	logger			*flogging.FabricLogger
	m               sync.Mutex
	taskCh          chan *verifyOrdiTask
	syncResultChMap map[int] chan<-*pb.BatchReply_SignVerReply // TODO maybe we need to change another container for storing the resp ch.

	rpcRequests  []*pb.BatchRequest_SignVerRequest
	batchSize int
	interval time.Duration // milliseconds

	gossipCount int // todo to be deleted. it's only for investigation purpose.
}

func (w *verifyOrdiWorker) start() {
	w.init()
	w.work()
}

func (w *verifyOrdiWorker) init() {
	w.logger = flogging.MustGetLogger("fpga.ordiVerify")
	w.m = sync.Mutex{}
	w.batchSize = 5000
	w.interval = 100
	w.syncResultChMap = make(map[int] chan<-*pb.BatchReply_SignVerReply)

	w.taskCh = make(chan *verifyOrdiTask, w.batchSize)
}

func (w *verifyOrdiWorker) work() {
	w.logger.Infof("ordinary verify requests collector starts to work.")

	// collect tasks
	go func() {
		for task := range w.taskCh {
			w.m.Lock()

			if strings.Contains(string(debug.Stack()), "gossip") {
				w.gossipCount++
				debug.PrintStack()
			}

			w.rpcRequests = append(w.rpcRequests, task.in)
			reqId := len(w.rpcRequests) - 1
			task.in.ReqId = fmt.Sprintf("%064x", reqId)
			w.syncResultChMap[reqId] = task.out
			w.m.Unlock()
		}
	}()

	// invoke the rpc every interval milliseconds
	go func() {
		var batchId uint64 = 0
		for true {
			time.Sleep( w.interval * time.Millisecond)

			w.m.Lock()
			if len(w.rpcRequests) > 0 {
				in := &pb.BatchRequest{SvRequests:w.rpcRequests}
				out := make(chan *pb.BatchReply)
				task := &verifyRpcTask{in, out}
				vk.pushFront(task)

				// parse rpc response
				for response := range out {
					w.parseResponse(response)
					w.rpcRequests = nil
				}

				w.logger.Warningf("total verify rpc requests: %d. gossip: %d.", len(w.rpcRequests), w.gossipCount)
				w.gossipCount = 0
			}
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

func (w *verifyOrdiWorker) addToTaskPool(task *verifyOrdiTask){
	w.taskCh <- task
}

func EndorserVerify(in *pb.BatchRequest_SignVerRequest) bool {
	ch := make(chan *pb.BatchReply_SignVerReply)
	ordiWorker.addToTaskPool(&verifyOrdiTask{in, ch})
	r := <-ch
	return r.Verified
}