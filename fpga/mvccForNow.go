package fpga

import (
	"context"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"github.com/hyperledger/fabric/protos/utils"
	"time"
)

var (
	sendBlock4MvccBlockRpcWorker pb.BlockRPCClient
	sendBlock4MvccBlockRpcTaskPool chan *sendBlock4MvccBlockRpcTask
)

type sendBlock4MvccBlockRpcTask struct {
	in  *pb.BlockRequest
	out chan<- *pb.BlockReply
}

func initSendBlock4MvccBlockRpcWorkerWorker() {
	sendBlock4MvccBlockRpcWorker = createBlockRpcClient()
	sendBlock4MvccBlockRpcTaskPool = make(chan *sendBlock4MvccBlockRpcTask)
}

func startSendBlock4MvccBlockRpcTaskPool() {
	go func() {
		logger.Infof("startSendBlock4MvccBlockRpcTaskPool")
		for true {
			params := <-sendBlock4MvccBlockRpcTaskPool
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			response, err := sendBlock4MvccBlockRpcWorker.SendBlockData(ctx, params.in)
			if err != nil {
				logger.Fatalf("%v.SendBlock4MvccBlockRpc(_) = _, %v: ", sendBlock4MvccBlockRpcWorker, err)
			}
			logger.Debugf("SendBlock4MvccBlockRpc succeeded. in: %v, out: %v.", params.in, response)
			params.out <- response
		}
	}()
}

func SendBlock4MvccBlockRpc(in *pb.BlockRequest) *pb.BlockReply {
	ch := make(chan *pb.BlockReply)
	sendBlock4MvccBlockRpcTaskPool <- &sendBlock4MvccBlockRpcTask{in, ch}
	return <-ch
	//return nil
}

func GenerateBlock(block *common.Block) *pb.BlockRequest {
	for tIdx, d := range block.Data.Data {
		if d == nil {
			logger.Fatalf("d is nil! index is %v", tIdx)
		}

		if env, err := utils.GetEnvelopeFromBlock(d); err != nil {
			logger.Fatalf("Error getting tx from block: %+v", err)
		} else if env != nil {
			// get the payload from the envelope
			payload, err := utils.GetPayload(env)
			if err != nil {
				logger.Fatalf("GetPayload returns err %s", err)
			}
			logger.Debugf("Header is %s", payload.Header)
			hdr := payload.Header
			if hdr == nil {
				logger.Fatalf("nil payload.header err %s", err)
			}
			chdr, err := utils.UnmarshalChannelHeader(hdr.ChannelHeader)
			if err != nil {
				logger.Fatalf("nil channel header err %s", err)
			}

			shdr, err := utils.GetSignatureHeader(hdr.SignatureHeader)
			if err != nil {
				logger.Fatalf("nil signature header err %s", err)
			}
			if shdr != nil {
				logger.Fatalf("shdr is not nil! err %s", err)
			}

			if common.HeaderType(chdr.Type) == common.HeaderType_ENDORSER_TRANSACTION {
				respPayload, err := utils.GetActionFromEnvelope(d)
				if err != nil {
					logger.Fatalf("GetActionFromEnvelope failed")
				}
				txRWSet := &rwsetutil.TxRwSet{}
				if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
					logger.Fatalf("txRWSet.FromProtoBytes failed")
				}

			} else if common.HeaderType(chdr.Type) == common.HeaderType_CONFIG {
				// [validator.go:432] if err := v.Support.Apply(configEnvelope); err != nil {
				// configuration tx doesn't do vscc.
				logger.Infof("configuration tx doesn't do vscc")
			} else {
				logger.Fatalf("Unknown transaction type [%s] in block number [%d] transaction index [%d]",
					common.HeaderType(chdr.Type), block.Header.Number, tIdx)
			}
		}
	}
	return nil
}