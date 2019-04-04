package fpga

import (
	"context"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"sync"
	"time"
)

var (
	verifyLogger = flogging.MustGetLogger("fpga.endorserVerify")
	verifyWorker = endorserVerifyWorker{}
)

func init() {
	verifyWorker.start()
}

type endorserVerifyRpcTask struct {
	in  *pb.VerifyReq
	out chan<- *pb.VerifyResult
}

type endorserVerifyWorker struct {
	client      pb.BlockRPCClient
	taskPool    chan *endorserVerifyRpcTask

	rpcResultMap map[int32] chan<-*pb.VerifyResult

	m sync.Mutex
	rpcRequests  []*pb.VerifyReq

	batchSize int
	interval time.Duration // milliseconds
}

func (e *endorserVerifyWorker) start() {
	e.init()
	e.work()
}

func (e *endorserVerifyWorker) init() {
	e.m = sync.Mutex{}
	e.batchSize = 5000
	e.interval = 100
	e.rpcResultMap = make(map[int32] chan<-*pb.VerifyResult)

	e.client = pb.NewBlockRPCClient(conn)
	e.taskPool = make(chan *endorserVerifyRpcTask, e.batchSize)
}

func (e *endorserVerifyWorker) work() {
	verifyLogger.Infof("startEndorserVerifyRpcTaskPool")

	// collect tasks
	go func() {
		for task := range e.taskPool {
			e.m.Lock()
			e.rpcRequests = append(e.rpcRequests, task.in)
			task.in.ReqId = int32(len(e.rpcRequests)) - 1
			e.m.Unlock()
			e.rpcResultMap[task.in.ReqId] = task.out
		}
	}()

	// invoke the rpc every interval milliseconds
	go func() {
		for true {
			time.Sleep( e.interval * time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			e.m.Lock()
			if len(e.rpcRequests) > 0 {
				response, err := e.client.EndorserVerify(ctx, &pb.EndorserVerifyRequest{VerifyReqs: e.rpcRequests})
				if err != nil {
					verifyLogger.Errorf("rpc call EndorserVerify failed. Will try again later. err: %v: ", e.client, err)
				} else {
					e.parseResponse(response)
					e.rpcRequests = nil
				}
			}
			e.m.Unlock()

			cancel()
		}
	}()
}

func (e *endorserVerifyWorker) parseResponse(response *pb.EndorserVerifyReply) {
	verifyResults := response.VerifyResults
	for _, result := range verifyResults {
		e.rpcResultMap[result.ReqId] <- result
	}
}

func (e *endorserVerifyWorker) addToTaskPool(task *endorserVerifyRpcTask){
	e.taskPool <- task
}

func EndorserVerify(in *pb.VerifyReq) bool {
	ch := make(chan *pb.VerifyResult)
	verifyWorker.addToTaskPool(&endorserVerifyRpcTask{in, ch})
	r := <-ch
	return r.Valid
}