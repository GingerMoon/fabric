package fpga

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/fpga"
	"google.golang.org/grpc"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
)

var block4vscc fpga.BlockRequest

var pubkeysStatistic = make(map[string]int)
var dbKeyVersion = make(map[string][]byte)

func init() {
	if os.Getenv("FPGA_MOCK") == "1" {
		go start()

		txs := make([]*fpga.BlockRequest_Transaction, 1, 1)
		block4vscc.Tx = txs
		block4vscc.Tx[0] = &fpga.BlockRequest_Transaction{}
	}
}

type fpgaServer struct {
}

func (s *fpgaServer) VerifySig4Vscc(cxt context.Context, env *fpga.VsccEnvelope) (*fpga.VsccResponse, error) {
	logger.Debugf("mock fpga  receive a VerifySig4Vscc request")

	intR := big.Int{}
	intR.SetBytes(env.SignR)
	intS := big.Int{}
	intS.SetBytes(env.SignS)

	x := big.Int{}
	x.SetBytes(env.PkX)
	y := big.Int{}
	y.SetBytes(env.PkY)
	pubkey := ecdsa.PublicKey{elliptic.P256(), &x, &y}

	valid := ecdsa.Verify(&pubkey, env.E, &intR, &intS)
	if !valid {
		logger.Warnf("grpc server verify result: false")
	}

	pubkeybytes, _ := x509.MarshalPKIXPublicKey(&pubkey)
	pubkeysStatistic[string(pem.EncodeToMemory(&pem.Block{Type: "EC PUBLIC KEY", Bytes: pubkeybytes}))]++
	logger.Infof("Currently the amount of public keys is: %v", len(pubkeysStatistic))


	// dump vscc for hw
	block4vscc.Tx[0].SgCount++
	sig := &fpga.BlockRequest_Transaction_SignatureStruct{}
	sig.E = env.E
	sig.SignR = env.SignR
	sig.SignW = env.SignS
	sig.PkX = env.PkX
	sig.PkY = env.PkY
	block4vscc.Tx[0].Signatures = append(block4vscc.Tx[0].Signatures, sig)

	return &fpga.VsccResponse{Result: valid}, nil
}

func vsccdump2file(bid uint64) {
	filename := "vscc" + strconv.Itoa(int(bid)) + ".dump"
	fo, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	data, err := proto.Marshal(&block4vscc)
	if err != nil {
		log.Fatalln("Marshal data error:", err)
	}
	fo.Write(data)

	// "clear" the vscc block
	block4vscc.Tx[0] = &fpga.BlockRequest_Transaction{}
}

func (s *fpgaServer) SendBlock4Mvcc(cxt context.Context, env *fpga.Block4Mvcc) (*fpga.MvccResponse, error) {
	vsccdump2file(env.Num)

	logger.Debugf("mock fpga  receive a SendBlock4Mvcc request")

	//mvcc dump for hw
	block := &fpga.BlockRequest{}
	block.BlockId = env.Num
	block.Crc = 0
	block.ColdStart = true
	block.TxCount = uint32(len(env.Txs))
	for _, e := range env.Txs {
		tx := &fpga.BlockRequest_Transaction{}
		tx.TxId = e.Id
		tx.WtCount = e.WtCount
		tx.RdCount = e.RdCount

		for _, ee := range e.Rs {
			read := &fpga.BlockRequest_Transaction_ReadStruct{}
			read.Key = ee.Key
			if ee.Version != nil { // for channel creation mvcc block, version is nil
				versionBytes := make([]byte, 40)
				binary.BigEndian.PutUint64(versionBytes[:8], ee.Version.BlockNum)
				binary.BigEndian.PutUint64(versionBytes[9:], ee.Version.TxNum)
				read.Version = versionBytes
			}
			tx.Reads = append(tx.Reads, read)
		}

		for _, ee := range e.Ws {
			write := &fpga.BlockRequest_Transaction_WriteStruct{}
			write.Key = ee.Key
			write.Value = ee.Value
			write.IsDel = ee.IsDel
			tx.Writes = append(tx.Writes, write)
		}
		block.Tx = append(block.Tx, tx)
	}

	filename := "mvcc" + strconv.Itoa(int(env.Num)) + ".dump"
	fo, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	data, err := proto.Marshal(block)
	if err != nil {
		log.Fatalln("Marshal data error:", err)
	}
	fo.Write(data)

	return &fpga.MvccResponse{Result: true}, nil
}

func (s *fpgaServer) SendBlockData(cxt context.Context, block *fpga.BlockRequest) (*fpga.BlockReply, error) {
	logger.Debugf("mock fpga received a SendBlockData request: ")
	logger.Debugf("Block Id is [%d]. The amount of txs is [%d]", block.BlockId, block.TxCount)
	reply := &fpga.BlockReply{}
	reply.BlockId = block.BlockId
	reply.TxReplies = make([]*fpga.BlockReply_TXReply, block.TxCount)

	for _, tx := range block.Tx {
		txReply := &fpga.BlockReply_TXReply{}
		txReply.TxId = tx.TxId
		txReply.IndexInBlock = tx.IndexInBlock
		txReply.TxValid = true
		txReply.SgValid = true

		// vscc
		logger.Debugf("tx[%d - %s]: The amount of signatures is [%d]", tx.IndexInBlock, tx.TxId, len(tx.Signatures))
		for i, sig := range tx.Signatures {
			intR := big.Int{}
			intR.SetBytes(sig.SignR)
			intS := big.Int{}
			intS.SetBytes(sig.SignW)

			x := big.Int{}
			x.SetBytes(sig.PkX)
			y := big.Int{}
			y.SetBytes(sig.PkY)
			pubkey := ecdsa.PublicKey{elliptic.P256(), &x, &y}

			valid := ecdsa.Verify(&pubkey, sig.E, &intR, &intS)
			if !valid {
				logger.Warnf("tx[%d] signature[%d] verification failed!", tx.IndexInBlock, i)
				txReply.SgValid = false
				break
			}
		}

		// mvcc
		logger.Debugf("tx[%d - %s]: RdCount is [%d]", tx.IndexInBlock, tx.TxId, tx.RdCount)
		txReply.RdChecks = make([]*fpga.BlockReply_TXReply_ReadReply, tx.RdCount)
		for i, read := range tx.Reads {
			readReply := &fpga.BlockReply_TXReply_ReadReply{RdKey:read.Key, RdValid:true}
			v, exists := dbKeyVersion[read.Key]
			if exists {
				blkNumInput := binary.BigEndian.Uint64(read.Version[:8])
				blkNumExisting := binary.BigEndian.Uint64(v[:8])
				if blkNumInput != blkNumExisting {
					readReply.RdValid = false
				} else {
					txNumInput := binary.BigEndian.Uint64(read.Version[9:])
					txNumExisting := binary.BigEndian.Uint64(v[9:])
					if txNumInput != txNumExisting {
						readReply.RdValid = false
					}
				}
			}
			txReply.RdChecks[i] = readReply
		}

		logger.Debugf("tx[%d - %s]: WtCount is [%d]", tx.IndexInBlock, tx.TxId, tx.WtCount)
		for _, write := range tx.Writes {
			versionBytes := make([]byte, 40)
			binary.BigEndian.PutUint64(versionBytes[:8], block.BlockId)
			binary.BigEndian.PutUint64(versionBytes[9:], uint64(tx.IndexInBlock))
			dbKeyVersion[write.Key] = versionBytes
			logger.Debugf("updated dbKeyVersion: [key:%s - blockNum: %d - txIndex: %d]", write.Key, block.BlockId, tx.IndexInBlock)
			v, _ := dbKeyVersion[write.Key]
			blkNumExisting := binary.BigEndian.Uint64(v[:8])
			txNumExisting := binary.BigEndian.Uint64(v[9:])
			logger.Debugf("updated dbKeyVersion: [key:%s - blockNum: %d - txIndex: %d]", write.Key, blkNumExisting, txNumExisting)

		}
		reply.TxReplies[tx.IndexInBlock] = txReply
	}
	return reply, nil
}

func newServer() *fpgaServer {
	s := &fpgaServer{}
	return s
}

func start() {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	//fpga.RegisterFpgaServer(grpcServer, newServer())
	fpga.RegisterBlockRPCServer(grpcServer, newServer())
	logger.Debugf("start a fpga mock server.")
	lis, err := net.Listen("tcp", "0.0.0.0:10000")
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatalf("grpcServer.Serve(lis) failed.")
	}
}