package fpga

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	commonerrors "github.com/hyperledger/fabric/common/errors"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"github.com/pkg/errors"
	"math/big"
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
				logger.Errorf("the batch containing the invalid signature is: %v", *in)
				logger.Errorf("the signature is invalid verified by FPGA. batchId: %d, reqId: %d", reply.BatchId, result.ReqId)

				for _, req := range svRequests {
					if req.ReqId != result.ReqId {
						continue
					}

					r := &big.Int{}
					r.SetBytes(req.SignR)
					s := &big.Int{}
					s.SetBytes(req.SignS)

					x := &big.Int{}
					x.SetBytes(req.Px)
					y := &big.Int{}
					y.SetBytes(req.Py)
					pubkey := ecdsa.PublicKey{Curve:elliptic.P256(), X:x, Y:y}
					succeed := ecdsa.Verify(&pubkey, req.Hash[:], r, s)
					if succeed {
						logger.Fatalf("The signature is invalid [verified by FPGA], but it is valid verified by go library.")
					}
					return nil
				}

				return &commonerrors.VSCCEndorsementPolicyError {
					Err: errors.Errorf("CommitBlockVerify failed. internal index: %d. ", i), // TODO the index(i) is not accurate.
				}
			}
		}
	}
	logger.Infof("CommitBlockVerify finished invoking verify rpc...")
	return nil
}