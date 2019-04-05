package utils

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"github.com/golang/protobuf/proto"
	bccsputils "github.com/hyperledger/fabric/bccsp/utils"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/util"
	fpgaident "github.com/hyperledger/fabric/fpga/identities"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/fpga"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"log"
	"strconv"
)

var logger = flogging.MustGetLogger("fpga.utilities")

func GenerateBlockVerifyRequests(block *common.Block) (requests []*pb.BatchRequest_SignVerRequest, err error) {

	for tIdx, d := range block.Data.Data { // block.Data.Data is Transaction array
		// get envelop
		if d == nil { // d is one Transaction
			err = errors.Errorf("d is nil! index is %v", tIdx)
			return nil, err
		}
		env, err := utils.GetEnvelopeFromBlock(d)
		if err != nil {
			err = errors.Errorf("Error getting tx from block: err: %+v", err)
			return nil, err
		}
		if env == nil {
			err = errors.Errorf("The envelop is nil. err: %+v", err)
			return nil, err
		}

		// get the payload of the envelop
		payload, err := utils.GetPayload(env) // payload is TransactionAction?
		if err != nil {
			err = errors.Errorf("GetPayload returns err: %+v", err)
			return nil, err
		}
		logger.Debugf("Header is %s", payload.Header)

		// get the header of the payload
		hdr := payload.Header
		if hdr == nil {
			err = errors.Errorf("GetPayload header is nil. err: %+v", err)
			return nil, err
		}
		// get the channel header
		chdr, err := utils.UnmarshalChannelHeader(hdr.ChannelHeader)
		if err != nil {
			err = errors.Errorf("channel header is nil. err: %+v", err)
			return nil, err
		}

		shdr, err := utils.GetSignatureHeader(hdr.SignatureHeader)
		if err != nil {
			err = errors.Errorf("signature header is nil. err: %+v", err)
			return nil, err
		}

		request, err := populateReq4creatorCheck(shdr.Creator, env.Signature, env.Payload, chdr.ChannelId)
		requests = append(requests, request)
		if err != nil {
			err = errors.Errorf("populateTx4creatorCheck err: %+v", err)
			return nil, err
		}
		// HeaderType_CONFIG has only mvcc, doesn't have vscc
		if common.HeaderType(chdr.Type) == common.HeaderType_ENDORSER_TRANSACTION {
			endorse_requests, err := populateReq4vscc(payload.Data)
			if err != nil {
				return nil, err
			}
			for _, e := range endorse_requests {
				requests = append(requests, e)
			}
		} else if common.HeaderType(chdr.Type) == common.HeaderType_CONFIG {
			// [validator.go:432] if err := v.Support.Apply(configEnvelope); err != nil {
			// configuration tx doesn't do vscc.
			logger.Infof("configuration tx doesn't do vscc")
		} else {
			log.Fatalf("Unknown transaction type [%s] in block number [%d] transaction index [%d]",
				common.HeaderType(chdr.Type), block.Header.Number, tIdx)
		}
	}
	return requests, nil
}

//for now fpga only supports bccsp.SHA256. bccsp.SHA3_256 is not supported.
func populateReq4creatorCheck(creatorBytes []byte, sigBytes []byte, msg []byte, ChainID string) (*pb.BatchRequest_SignVerRequest, error) {
	// check for nil argument
	if creatorBytes == nil || sigBytes == nil || msg == nil {
		return nil, errors.New("populateTx4creatorCheck nil arguments")
	}
	request := &pb.BatchRequest_SignVerRequest{}
	request.ReqId = "0"

	request.Hash = util.ComputeSHA256(msg)

	r, s, err := bccsputils .UnmarshalECDSASignature(sigBytes)
	if err != nil {
		return nil, errors.Errorf("utils.UnmarshalECDSASignature failed. signature is: %v, error message: %v.", base64.StdEncoding.EncodeToString(sigBytes), err.Error())
	}
	request.SignR = r.Bytes()
	request.SignS = s.Bytes()


	mspObj := mspmgmt.GetIdentityDeserializer(ChainID)
	if mspObj == nil {
		return nil, errors.Errorf("could not get msp for channel [%s]", ChainID)
	}

	// get the identity of the creator
	creator, err := mspObj.DeserializeIdentity(creatorBytes)
	if err != nil {
		return nil, errors.WithMessage(err, "MSP error")
	}

	// the check has already been done in checkSignatureFromCreatorFpga
	// ensure that creator is a valid certificate
	//err = creator.Validate()
	//if err != nil {
	//	return errors.WithMessage(err, "creator certificate is not valid")
	//}

	id := fpgaident.GenerateFpgaIdentity(creator)
	pubkey, err := id.GetPublicKey()
	if err != nil {
		return nil, err
	}
	request.Px = pubkey.X.Bytes()
	request.Py = pubkey.Y.Bytes()

	return request, nil
}

func populateReq4vscc(txBytes []byte) (requests []*pb.BatchRequest_SignVerRequest, err error) {
	//var wrNamespace []string
	//// !!! for now we don't consider this scenario.
	//// alwaysEnforceOriginalNamespace := v.support.Capabilities().V1_2Validation()
	////if alwaysEnforceOriginalNamespace {
	////	wrNamespace = append(wrNamespace, ccID)
	////}
	//ccID := respPayload.ChaincodeId.Name
	//wrNamespace = append(wrNamespace, ccID)
	//for _, ns := range txRWSet.NsRwSets {
	//	if ns.NameSpace != ccID {
	//		wrNamespace = append(wrNamespace, ns.NameSpace)
	//	}
	//}
	//TODO for now we don't support policy, we process according to the default policy "all agreed"
	//ctx := &Context{
	//	Seq:       seq,
	//	Envelope:  envBytes,
	//	Block:     block,
	//	TxID:      chdr.TxId,
	//	Channel:   chdr.ChannelId,
	//	Namespace: ns,
	//	Policy:    policy,
	//	VSCCName:  vscc.ChaincodeName,
	//}
	// err = plugin.Validate(ctx.Block, ctx.Namespace, ctx.Seq,h 0, SerializedPolicy(ctx.Policy))

	transaction, err := utils.GetTransaction(txBytes)
	if err != nil {
		return nil, errors.Errorf("GetTransaction failed, err %s", err)
	}

	if len(transaction.Actions) != 1 {
		return nil,  errors.Errorf("only one action per transaction is supported, tx contains %d", len(transaction.Actions))
	}

	cap, err := utils.GetChaincodeActionPayload(transaction.Actions[0].Payload)
	if err != nil {
		return nil,  errors.Errorf("GetChaincodeActionPayload failed, err %s", err)
	}

	endorsements := cap.Action.Endorsements
	prp := cap.Action.ProposalResponsePayload
	//pRespPayload, err := utils.GetProposalResponsePayload(cap.Action.ProposalResponsePayload)
	//respPayload, err := utils.GetChaincodeAction(pRespPayload.Extension)
	//rwset := respPayload.Results

	for i, endorsement := range endorsements {
		request := &pb.BatchRequest_SignVerRequest{}
		request.ReqId = strconv.Itoa(i)

		data := make([]byte, len(prp)+len(endorsement.Endorser))
		copy(data, prp)
		copy(data[len(prp):], endorsement.Endorser)
		request.Hash = util.ComputeSHA256(data)

		sId := &msp.SerializedIdentity{}
		err := proto.Unmarshal(endorsement.Endorser, sId) // func (msp *bccspmsp) DeserializeIdentity
		if err != nil {
			return nil, errors.Errorf("could not deserialize a SerializedIdentity")
		}
		bl, _ := pem.Decode(sId.IdBytes)
		if bl == nil {
			errors.Errorf("could not decode the PEM structure")
		}
		cert, err := x509.ParseCertificate(bl.Bytes)
		if err != nil {
			logger.Fatalf("parseCertificate failed")
		}
		pubkey, ok := cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			logger.Fatalf("Invalid raw material. Expected *ecdsa.PublicKey.")
		}
		request.Px = pubkey.X.Bytes()
		request.Py = pubkey.Y.Bytes()

		r, s, err := bccsputils.UnmarshalECDSASignature(endorsement.Signature)
		if err != nil {
			logger.Fatalf("utils.UnmarshalECDSASignature failed. signature is: %v, error message: %v.", base64.StdEncoding.EncodeToString(endorsement.Signature), err.Error())
		}
		request.SignR = r.Bytes()
		request.SignS = s.Bytes()
		requests = append(requests, request)
	}
	return requests, nil
}