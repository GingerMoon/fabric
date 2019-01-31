package fpga

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"github.com/golang/protobuf/proto"
	bccsputl "github.com/hyperledger/fabric/bccsp/utils"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/fpga"
	m "github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"time"
)

const (
	channelConfigKey = "resourcesconfigtx.CHANNEL_CONFIG_KEY"
	peerNamespace    = ""
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
			//logger.Debugf("SendBlock4MvccBlockRpc succeeded. in: %v, out: %v.", params.in, response)
			params.out <- response
		}
	}()
}

func SendBlock4MvccBlockRpc(block *common.Block) *pb.BlockReply {
	in := generateBlock(block)
	ch := make(chan *pb.BlockReply)
	sendBlock4MvccBlockRpcTaskPool <- &sendBlock4MvccBlockRpcTask{in, ch}
	return <-ch
}

func generateBlock(block *common.Block) *pb.BlockRequest {
	b := &pb.BlockRequest{BlockId:block.Header.Number, ColdStart:false, Crc:0}
	b.TxCount = uint32(len(block.Data.Data))
	b.Tx = make([]*pb.BlockRequest_Transaction, b.TxCount)

	for tIdx, d := range block.Data.Data { // block.Data.Data is Transaction array
		tx := &pb.BlockRequest_Transaction{}
		tx.IndexInBlock = uint32(tIdx)

		// get envelop
		if d == nil { // d is one Transaction
			logger.Fatalf("d is nil! index is %v", tIdx)
		}
		env, err := utils.GetEnvelopeFromBlock(d)
		if err != nil {
			logger.Fatalf("Error getting tx from block: %+v", err)
		}
		if env == nil {
			logger.Warnf("Please have a look at why evn is nil!!!!!!!")
			continue
		}

		// get the payload of the envelop
		payload, err := utils.GetPayload(env) // payload is TransactionAction?
		if err != nil {
			logger.Fatalf("GetPayload returns err %s", err)
		}
		logger.Debugf("Header is %s", payload.Header)
		// get the header of the payload
		hdr := payload.Header
		if hdr == nil {
			logger.Fatalf("nil payload.header err %s", err)
		}
		// get the channel header
		chdr, err := utils.UnmarshalChannelHeader(hdr.ChannelHeader)
		if err != nil {
			logger.Fatalf("nil channel header err %s", err)
		}

		shdr, err := utils.GetSignatureHeader(hdr.SignatureHeader)
		if err != nil {
			logger.Fatalf("nil signature header err %s", err)
		}

		err = populateTx4creatorCheck(tx, shdr.Creator, env.Signature, env.Payload, chdr.ChannelId)
		if err != nil {
			logger.Errorf("check creator returns error: %s. Block id: %v", err.Error(), b.BlockId)
		}

		// HeaderType_CONFIG has only mvcc, doesn't have vscc
		if common.HeaderType(chdr.Type) == common.HeaderType_ENDORSER_TRANSACTION {
			tx.TxId = chdr.TxId
			// err, cde := v.Vscc.VSCCValidateTx(tIdx, payload, d, block)

			// for mvcc rwset
			respPayload, err := utils.GetActionFromEnvelope(d)
			if err != nil {
				logger.Fatalf("GetActionFromEnvelope failed")
			}
			txRWSet := &rwsetutil.TxRwSet{}
			if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
				logger.Fatalf("txRWSet.FromProtoBytes failed")
			}
			populateEndorsedTx4mvcc(tx, txRWSet)
			populateTx4vscc(tx, payload.Data)
		} else if common.HeaderType(chdr.Type) == common.HeaderType_CONFIG {
			// [validator.go:432] if err := v.Support.Apply(configEnvelope); err != nil {
			// configuration tx doesn't do vscc.
			logger.Infof("configuration tx doesn't do vscc")
			populateConfigTx4mvcc(tx, env)
		} else {
			logger.Fatalf("Unknown transaction type [%s] in block number [%d] transaction index [%d]",
				common.HeaderType(chdr.Type), block.Header.Number, tIdx)
		}
		b.Tx[tIdx] = tx // b.Tx = append(b.Tx, tx)
	}
	return b
}

//for now fpga only supports bccsp.SHA256. bccsp.SHA3_256 is not supported.
func populateTx4creatorCheck(tx *pb.BlockRequest_Transaction, creatorBytes []byte, sigBytes []byte, msg []byte, ChainID string) error {
	// check for nil argument
	if creatorBytes == nil || sigBytes == nil || msg == nil {
		return errors.New("nil arguments")
	}

	sig := &pb.BlockRequest_Transaction_SignatureStruct{}
	sig.E = util.ComputeSHA256(msg)

	r, s, err := bccsputl.UnmarshalECDSASignature(sigBytes)
	if err != nil {
		logger.Fatalf("utils.UnmarshalECDSASignature failed. signature is: %v, error message: %v.", base64.StdEncoding.EncodeToString(sigBytes), err.Error())
	}
	sig.SignR = r.Bytes()
	sig.SignW = s.Bytes()


	mspObj := mspmgmt.GetIdentityDeserializer(ChainID)
	if mspObj == nil {
		return errors.Errorf("could not get msp for channel [%s]", ChainID)
	}

	// get the identity of the creator
	creator, err := mspObj.DeserializeIdentity(creatorBytes)
	if err != nil {
		return errors.WithMessage(err, "MSP error")
	}

	// the check has already been done in checkSignatureFromCreatorFpga
	// ensure that creator is a valid certificate
	//err = creator.Validate()
	//if err != nil {
	//	return errors.WithMessage(err, "creator certificate is not valid")
	//}

	pubkey, err := creator.GetPublicKey()
	if err != nil {
		logger.Fatalf("Expected *ecdsa.PublicKey.")
	}
	sig.PkX = pubkey.X.Bytes()
	sig.PkY = pubkey.Y.Bytes()

	tx.Signatures = append(tx.Signatures, sig)

	return nil
}

func populateTx4vscc(tx *pb.BlockRequest_Transaction, txBytes []byte) {
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
		logger.Fatalf("GetTransaction failed, err %s", err)
	}
	cap, err := utils.GetChaincodeActionPayload(transaction.Actions[0].Payload)
	if err != nil {
		logger.Fatalf("GetChaincodeActionPayload failed, err %s", err)
	}

	endorsements := cap.Action.Endorsements
	prp := cap.Action.ProposalResponsePayload
	//pRespPayload, err := utils.GetProposalResponsePayload(cap.Action.ProposalResponsePayload)
	//respPayload, err := utils.GetChaincodeAction(pRespPayload.Extension)
	//rwset := respPayload.Results

	for _, endorsement := range endorsements {
		sig := &pb.BlockRequest_Transaction_SignatureStruct{}

		data := make([]byte, len(prp)+len(endorsement.Endorser))
		copy(data, prp)
		copy(data[len(prp):], endorsement.Endorser)
		sig.E = util.ComputeSHA256(data)

		//signatureSet = append(signatureSet, &common.SignedData{
		//	// set the data that is signed; concatenation of proposal response bytes and endorser ID
		//	Data: data,
		//	// set the identity that signs the message: it's the endorser
		//	Identity: endorsement.Endorser,
		//	// set the signature
		//	Signature: endorsement.Signature})

		sId := &m.SerializedIdentity{}
		err := proto.Unmarshal(endorsement.Endorser, sId) // func (msp *bccspmsp) DeserializeIdentity
		if err != nil {
			logger.Fatalf("could not deserialize a SerializedIdentity")
		}
		bl, _ := pem.Decode(sId.IdBytes)
		if bl == nil {
			logger.Fatalf("could not decode the PEM structure")
		}
		cert, err := x509.ParseCertificate(bl.Bytes)
		if err != nil {
			logger.Fatalf("parseCertificate failed")
		}
		pubkey, ok := cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			logger.Fatalf("Invalid raw material. Expected *ecdsa.PublicKey.")
		}
		sig.PkX = pubkey.X.Bytes()
		sig.PkY = pubkey.Y.Bytes()

		r, s, err := bccsputl.UnmarshalECDSASignature(endorsement.Signature)
		if err != nil {
			logger.Fatalf("utils.UnmarshalECDSASignature failed. signature is: %v, error message: %v.", base64.StdEncoding.EncodeToString(endorsement.Signature), err.Error())
		}
		sig.SignR = r.Bytes()
		sig.SignW = s.Bytes()
		//sig.SignW = elliptic.P256().Inverse(s).Bytes()
		//if os.Getenv("FPGA_MOCK") == "1" {
		//	// TBD: Right now HW doesn't support inverse(), so we have to pass down w (a.k.a inversion of s) instead of s.
		//	sig.SignW = s.Bytes()
		//}
		tx.Signatures = append(tx.Signatures, sig)
	}
}

func populateEndorsedTx4mvcc(tx *pb.BlockRequest_Transaction, txRWSet *rwsetutil.TxRwSet) {
	if tx == nil {
		logger.Fatalf("tx is nil!")
	}

	// the key of fpgapb.TxRS and fpgapb.TxWS need to be constructed by "namespace key".
	// [stateleveldb] ApplyUpdates -> DEBU 160 Channel [mychannel]:
	// Applying key(string)=[mycc 2] key(bytes)=[[]byte{0x6d, 0x79, 0x63, 0x63, 0x0, 0x32}]
	for _, rwset := range txRWSet.NsRwSets {
		logger.Debugf("namespace is %v.", rwset.NameSpace)
		// read set
		tx.RdCount += uint32(len(rwset.KvRwSet.Reads))
		for _, e := range rwset.KvRwSet.Reads {
			rs := &pb.BlockRequest_Transaction_ReadStruct{}
			rs.Key = rwset.NameSpace + " " + e.Key
			if e.Version != nil { // for channel creation mvcc block, version is nil
				versionBytes := make([]byte, 40)
				binary.BigEndian.PutUint64(versionBytes[:8], e.Version.BlockNum)
				binary.BigEndian.PutUint64(versionBytes[9:], e.Version.TxNum)
				rs.Version = versionBytes
			}
			tx.Reads = append(tx.Reads, rs)
			logger.Debugf("txs(%v) rs: %+v appended to read set.", tx.TxId, rs)
		}

		// wirte set
		tx.WtCount += uint32(len(rwset.KvRwSet.Writes))
		for _, e := range rwset.KvRwSet.Writes {
			ws := &pb.BlockRequest_Transaction_WriteStruct{}
			ws.Key = rwset.NameSpace + " " + e.Key
			ws.Value = e.Value
			ws.IsDel = e.IsDelete
			tx.Writes = append(tx.Writes, ws)
			logger.Debugf("txs(%v) ws: {Key: %v, Value(base64 encoded hash): %v, IsDel: %v, } appended to write set.",
				tx.TxId, ws.Key, base64.StdEncoding.EncodeToString(util.ComputeSHA256(ws.Value)), ws.IsDel)
		}
	}
}

func populateConfigTx4mvcc(tx *pb.BlockRequest_Transaction, env *common.Envelope) {
	configEnvelope := &common.ConfigEnvelope{}
	if _, err := utils.UnmarshalEnvelopeOfType(env, common.HeaderType_CONFIG, configEnvelope); err != nil {
		logger.Fatalf("utils.UnmarshalEnvelopeOfType(env, common.HeaderType_CONFIG, configEnvelope) failed")
	}
	channelConfig := configEnvelope.Config
	serializedConfig, err := proto.Marshal(channelConfig)
	if err != nil {
		logger.Infof(err.Error())
	}
	// simulator.SetState(peerNamespace, channelConfigKey, serializedConfig)
	// nsPubRwBuilder := b.getOrCreateNsPubRwBuilder(ns)
	//	nsPubRwBuilder.writeMap[key] = newKVWrite(key, value)

	tx.WtCount = 1
	ws := &pb.BlockRequest_Transaction_WriteStruct{}
	ws.IsDel = false
	ws.Key = channelConfigKey
	ws.Value = serializedConfig
	tx.Writes = append(tx.Writes, ws)
}