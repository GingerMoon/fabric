package fpga

import (
	"container/list"
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"math/big"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

	rcLock     sync.Mutex
	cResultChs map[int] chan<-*pb.BatchReply_SignVerReply // TODO maybe we need to change another container for storing the resp ch.
	cRequests  *list.List

	// rpc call EndorserVerify failed. batchId: 1763. ReqCount: 17837. err: rpc error: code = ResourceExhausted desc = grpc: received message larger than max (4262852 vs. 4194304)
	// log: Exiting due to the failed rpc request: batch_id:1763 batch_type:1 req_count:17837
	interval time.Duration // milliseconds
	batchSize     int

	gossipCount int32 // todo to be deleted. it's only for investigation purpose.
}

func (w *verifyOrdiWorker) start() {
	w.init()
	w.work()
}

func (w *verifyOrdiWorker) init() {
	w.logger = flogging.MustGetLogger("fpga.ordiVerify")
	w.batchSize = 10000
	w.cRequests = list.New()

	tmp, err := strconv.Atoi(os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	if err != nil {
		w.logger.Fatalf("FPGA_BATCH_GEN_INTERVAL(%s)(ms) is not set correctly!, not the batch_gen_interval is set to default as 50 ms",
			os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	}
	w.interval = time.Duration(tmp)

	w.rcLock = sync.Mutex{}
	w.cResultChs = make(map[int] chan<-*pb.BatchReply_SignVerReply)

	w.taskCh = make(chan *verifyOrdiTask)
}

func (w *verifyOrdiWorker) work() {
	w.logger.Infof("ordinary verify requests collector starts to work.")

	// collect tasks
	go func() {
		for task := range w.taskCh {
			w.rcLock.Lock()
			w.logger.Debugf("enter lock w.cRequests = append(w.cRequests, task.in)")
			w.cRequests.PushBack(task.in)
			reqId := w.cRequests.Len() - 1
			task.in.ReqId = fmt.Sprintf("%064d", reqId)
			w.cResultChs[reqId] = task.out
			w.logger.Debugf("exit lock w.cRequests = append(w.cRequests, task.in)")
			w.rcLock.Unlock()
		}
	}()

	// invoke the rpc every interval Microsecond
	go func() {
		var batchId uint64 = 0
		for true {
			time.Sleep( w.interval * time.Microsecond)

			w.rcLock.Lock()
			w.logger.Debugf("enter lock for w.cRequests = nil")
			if w.cRequests.Len() > 0 {
				size := w.cRequests.Len()
				if size > w.batchSize {
					size = w.batchSize
				}
				var svReqs []*pb.BatchRequest_SignVerRequest
				for i := 0; i < size; i++ {
					element := w.cRequests.Front()
					w.cRequests.Remove(element)
					req := element.Value.(*pb.BatchRequest_SignVerRequest)
					if req == nil {
						w.logger.Fatalf("why req: = element.Value.(*pb.BatchRequest_SignVerRequest) failed?!!")
					}
					svReqs = append(svReqs, req)
				}
				if w.cRequests.Len() > w.batchSize {
					w.logger.Warningf("current w.cRequests.Len is %d, which is bigger than the batch size(%d)", w.cRequests.Len(), w.batchSize)
				}

				in := &pb.BatchRequest{SvRequests:svReqs}
				out := make(chan *pb.BatchReply)
				task := &verifyRpcTask{in, out}
				vk.pushBack(task)

				// parse rpc response
				for response := range out {
					w.logger.Debugf("total verify rpc requests: %d. gossip: %d.", w.cRequests.Len(), atomic.LoadInt32(&w.gossipCount))
					w.parseResponse(response)
					atomic.StoreInt32(&w.gossipCount, 0)
				}
			}
			w.logger.Debugf("exit lock for w.cRequests = nil")
			w.rcLock.Unlock()
			batchId++
		}
	}()
}

// this method need to be locked where it is invoked.
func (w *verifyOrdiWorker) parseResponse(response *pb.BatchReply) {
	verifyResults := response.SvReplies
	for _, result := range verifyResults {
		reqId, err := strconv.Atoi(result.ReqId)
		if err != nil || w.cResultChs[reqId] == nil {
			w.logger.Fatalf("[verifyOrdiWorker] the request id(%s) in the rpc reply is not stored before.", result)
		}

		w.cResultChs[reqId] <- result
	}
}

func (w *verifyOrdiWorker) putToTaskCh(task *verifyOrdiTask){
	w.taskCh <- task
}

func EndorserVerify(in *pb.BatchRequest_SignVerRequest) bool {
	if strings.Contains(string(debug.Stack()), "gossip") {
		atomic.AddInt32(&ordiWorker.gossipCount, 1)
	}

	logger.Debugf("EndorserVerify is invoking verify rpc...")
	ch := make(chan *pb.BatchReply_SignVerReply)
	ordiWorker.putToTaskCh(&verifyOrdiTask{in, ch})
	r := <-ch
	logger.Debugf("EndorserVerify finished invoking verify rpc. result: %v", r)
	if !r.Verified {
		r := &big.Int{}
		r.SetBytes(in.SignR)
		s := &big.Int{}
		s.SetBytes(in.SignS)

		x := &big.Int{}
		x.SetBytes(in.Px)
		y := &big.Int{}
		y.SetBytes(in.Py)
		pubkey := ecdsa.PublicKey{Curve:elliptic.P256(), X:x, Y:y}
		succeed := ecdsa.Verify(&pubkey, in.Hash[:], r, s)
		if succeed {
			panic("The signature is invalid [verified by FPGA], but it is valid by go library.")
		}
	}
	return r.Verified
}