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
	"strconv"
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

	cResultChs  sync.Map //map[int] chan<-*pb.BatchReply_SignVerReply // TODO maybe we need to change another container for storing the resp ch.

	reqLock *sync.Mutex
	cRequests  *list.List

	// rpc call EndorserVerify failed. batchId: 1763. ReqCount: 17837. err: rpc error: code = ResourceExhausted desc = grpc: received message larger than max (4262852 vs. 4194304)
	// log: Exiting due to the failed rpc request: batch_id:1763 batch_type:1 req_count:17837
	interval time.Duration
	batchSize     int

	//gossipCount int32 // todo to be deleted. it's only for investigation purpose.
}

func (w *verifyOrdiWorker) start() {
	w.init()
	w.work()
}

func (w *verifyOrdiWorker) init() {
	w.logger = flogging.MustGetLogger("fpga.ordiVerify")
	w.batchSize = 10000

	tmp, err := strconv.Atoi(os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	if err != nil {
		w.logger.Fatalf("FPGA_BATCH_GEN_INTERVAL(%s)(ms) is not set correctly!, not the batch_gen_interval is set to default as 50 ms",
			os.Getenv("FPGA_BATCH_GEN_INTERVAL"))
	}
	w.interval = time.Duration(tmp)

	w.taskCh = make(chan *verifyOrdiTask, w.batchSize)

	w.reqLock = &sync.Mutex{}
	w.cRequests = list.New()

}

func (w *verifyOrdiWorker) work() {
	w.logger.Infof("ordinary verify cRequests collector starts to work.")

	// collect tasks
	go func() {
		reqId := 1
		for task := range w.taskCh {
			w.reqLock.Lock()
			w.cRequests.PushBack(task.in)
			task.in.ReqId = fmt.Sprintf("%064d", reqId)
			w.cResultChs.Store(reqId, task.out)
			w.reqLock.Unlock()

			reqId++
		}
	}()

	// invoke the rpc every interval Microsecond and less than batch size.
	go func() {
		var batchId uint64 = 0
		for true {
			time.Sleep( w.interval * time.Microsecond)

			w.reqLock.Lock()
			size := w.cRequests.Len() // pending cRequests amount
			if size > 0 {
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
					go w.parseResponse(response)
					//w.logger.Debugf("total verify rpc cRequests: %d. gossip: %d.", w.cRequests.Len(), atomic.LoadInt32(&w.gossipCount))
					//atomic.StoreInt32(&w.gossipCount, 0)
				}
			}
			w.reqLock.Unlock()
			batchId++
		}
	}()
}

// this method need to be locked where it is invoked.
func (w *verifyOrdiWorker) parseResponse(response *pb.BatchReply) {
	verifyResults := response.SvReplies
	for _, result := range verifyResults {
		reqId, err := strconv.Atoi(result.ReqId)

		v, ok := w.cResultChs.Load(reqId)
		if err != nil || !ok {
			w.logger.Fatalf("the request id(%s) in the rpc reply is not stored before.", result)
		}

		outCh := v.(chan *pb.BatchReply_SignVerReply)
		if outCh == nil {
			w.logger.Fatalf("failed to find the request id(%s)'s output channel. ", result)
		}
		outCh <- result
		close(outCh)
		w.cResultChs.Delete(reqId)
	}
}

func (w *verifyOrdiWorker) putToTaskCh(task *verifyOrdiTask){
	w.taskCh <- task
}

func EndorserVerify(in *pb.BatchRequest_SignVerRequest) bool {
	//if strings.Contains(string(debug.Stack()), "gossip") {
	//	atomic.AddInt32(&ordiWorker.gossipCount, 1)
	//}

	ch := make(chan *pb.BatchReply_SignVerReply)
	ordiWorker.putToTaskCh(&verifyOrdiTask{in, ch})
	r := <-ch

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