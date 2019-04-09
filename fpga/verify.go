package fpga

import (
	"context"
	"fmt"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"strconv"
	"sync"
	"time"
)

var (
	verifyLogger = flogging.MustGetLogger("fpga.endorserVerify")
	endorserVerifyWorker = verifyWorker{}
)

func init() {
	endorserVerifyWorker.start()
}


type verifyRpcTask struct {
	in  *pb.BatchRequest_SignVerRequest
	out chan<- *pb.BatchReply_SignVerReply
}

type verifyWorker struct {
	client      pb.BatchRPCClient
	taskPool    chan *verifyRpcTask

	rpcResultMap map[string] chan<-*pb.BatchReply_SignVerReply

	m sync.Mutex
	rpcRequests  []*pb.BatchRequest_SignVerRequest

	batchSize int
	interval time.Duration // milliseconds
}

func (e *verifyWorker) start() {
	e.init()
	e.work()
}

func (e *verifyWorker) init() {
	e.m = sync.Mutex{}
	e.batchSize = 5000
	e.interval = 100
	e.rpcResultMap = make(map[string] chan<-*pb.BatchReply_SignVerReply)

	e.client = pb.NewBatchRPCClient(conn)
	e.taskPool = make(chan *verifyRpcTask, e.batchSize)
}

func (e *verifyWorker) work() {
	verifyLogger.Infof("startEndorserVerifyRpcTaskPool")

	// collect tasks
	go func() {
		for task := range e.taskPool {
			e.m.Lock()
			e.rpcRequests = append(e.rpcRequests, task.in)
			reqIdTmp := strconv.Itoa(len(e.rpcRequests) - 1)
			task.in.ReqId = fmt.Sprintf("%32s", reqIdTmp)
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
				response, err := e.client.Verify(ctx, &pb.BatchRequest{SvRequests:e.rpcRequests, BatchType:1, BatchId: 0, ReqCount:uint32(len(e.rpcRequests))})
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

func (e *verifyWorker) parseResponse(response *pb.BatchReply) {
	verifyResults := response.SvReplies
	for _, result := range verifyResults {
		e.rpcResultMap[result.ReqId] <- result
	}
}

func (e *verifyWorker) addToTaskPool(task *verifyRpcTask){
	e.taskPool <- task
}

func EndorserVerify(in *pb.BatchRequest_SignVerRequest) bool {
	ch := make(chan *pb.BatchReply_SignVerReply)
	endorserVerifyWorker.addToTaskPool(&verifyRpcTask{in, ch})
	r := <-ch
	return r.Verified
}

//func CommitterVerify(in *pb.BatchRequest_SignVerRequest) bool {
//	logger.Fatalf("CommitterVerify should not be used because the verification in commit block should be handled in the same batch.")
//	ch := make(chan *pb.BatchReply_SignVerReply)
//	committerVerifyWorker.addToTaskPool(&verifyRpcTask{in, ch})
//	r := <-ch
//	return r.Verified
//}