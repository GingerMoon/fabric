package fpga

import (
	"context"
	"github.com/pkg/errors"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"time"
)

var (
	verifyBlocLogger = flogging.MustGetLogger("fpga.committerVerify")
	blockVerifyWorker = verifyBlockWorker{}
)

func init() {
	endorserVerifyWorker.client = pb.NewBatchRPCClient(conn)
}

type verifyBlockWorker struct {
	client      pb.BatchRPCClient
}

func CommitBlockVerify(svRequests []*pb.BatchRequest_SignVerRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	response, err := blockVerifyWorker.client.Verify(ctx, &pb.BatchRequest{SvRequests:svRequests, BatchType:1, BatchId: 0, ReqCount:uint32(len(svRequests))})
	if err != nil {
		verifyLogger.Errorf("rpc call CommitBlockVerify failed. err: %v: ", err)
		return err
	}

	verifyResults := response.SvReplies
	for i, result := range verifyResults {
		if !result.Verified {
			return errors.Errorf("CommitBlockVerify (Endorsement or CheckCreator) failed. internal index: %d. ", i)
		}
	}

	cancel()
	return nil
}