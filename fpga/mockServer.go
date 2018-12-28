package fpga

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/binary"
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

func init() {
	logger.Infof("!!!!!!!!!!!!!!!!!!!!!!!!!--%v", os.Getenv("mockserver"))
	if os.Getenv("mockserver") == "1" {
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
	intR.SetString(env.SignR, 10)
	intS := big.Int{}
	intS.SetString(env.SignS, 10)

	x := big.Int{}
	x.SetString(env.PkX, 10)
	y := big.Int{}
	y.SetString(env.PkY, 10)
	pubkey := ecdsa.PublicKey{elliptic.P256(), &x, &y}

	digest, _ := base64.StdEncoding.DecodeString(env.E)
	valid := ecdsa.Verify(&pubkey, digest, &intR, &intS)
	if !valid {
		logger.Warnf("grpc server verify result: false")
	}

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

func newServer() *fpgaServer {
	s := &fpgaServer{}
	return s
}

func start() {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	fpga.RegisterFpgaServer(grpcServer, newServer())
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
