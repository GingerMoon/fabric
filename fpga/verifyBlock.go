package fpga

import (
	pb "github.com/hyperledger/fabric/protos/fpga"
	"github.com/pkg/errors"
)

func CommitBlockVerify(svRequests []*pb.BatchRequest_SignVerRequest) error {
	logger.Infof("CommitBlockVerify is invoking verify rpc...")
	if len(svRequests) == 0 {
		return errors.Errorf("CommitBlockVerify len(svRequests) is 0")
	}

	in := &pb.BatchRequest{SvRequests:svRequests}
	out := make(chan *pb.BatchReply)
	task := &verifyRpcTask{in, out}
	vk.pushFront(task)
	for reply := range out {
		verifyResults := reply.SvReplies
		for i, result := range verifyResults {
			if !result.Verified {
				return errors.Errorf("CommitBlockVerify (Endorsement or CheckCreator) failed. internal index: %d. ", i)
			}
		}
	}
	logger.Infof("CommitBlockVerify finished invoking verify rpc...")
	return nil
}