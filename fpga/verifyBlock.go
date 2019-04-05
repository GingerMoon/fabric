package fpga

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/fpga/utils"
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
	svRequests, err := utils.GenerateBlockVerifyRequests(block)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	response, err := blockVerifyWorker.client.Verify(ctx, &pb.BatchRequest{SvRequests:svRequests, BatchType:1, BatchId: 0, ReqCount:uint32(len(block.Data.Data))})
	if err != nil {
		verifyLogger.Errorf("rpc call CommitBlockVerify failed. err: %v: ", err)
		return err
	}

	verifyResults := response.SvReplies
	for _, result := range verifyResults {
		if !result.Verified {
			bytes, _ := proto.Marshal(block)
			return errors.New(fmt.Sprintf("the block [%s] contains invalid signature!", hex.EncodeToString(bytes)))
		}
	}

	cancel()
	return nil
}