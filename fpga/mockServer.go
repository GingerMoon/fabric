package fpga

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"github.com/hyperledger/fabric/protos/fpga"
	"google.golang.org/grpc"
	"math/big"
	"net"
        "githun.com/golang/protobuf/proto"
        "log"
)

func init() {
	go start()
}

type fpgaServer struct {

}

func (s *fpgaServer) VerifySig4Vscc(cxt context.Context, env *fpga.VsccEnvelope) (*fpga.VsccResponse, error) {
	logger.Infof("mock fpga  receive a VerifySig4Vscc request")

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

        // compose block_rpc data structure
        BlockRequest bq
        bq.BlockId = 0
        bq.ColdStart = 0
        bq.TxCount = 1

        txs := make([]*BlockRequest_Transaction, 0, 1)
        txs[0].TxId = ""
        txs[0].RdCount = 0
        txs[0].WtCount = 0
        txs[0].SgCount = 1
        txs[0].DbgTxValid = false
        
        BlockRequest_Transaction_SignatureStruct sig
        sig.SignR = intR
        sig.SignW = intS
        sig.PkX = x
        sig.PkY = y
        sig.E = digest

        txs[0].Signatures = sig
        bq.Transaction = txs
        bq.Crc = 0x1234

        // serialize to stream
        data, err := proto.Marshal()
        if err != nil {
            log.Fatalln("Marshal data error:", err)
        }
	
        return &fpga.VsccResponse{Result:valid}, nil
}

func (s *fpgaServer) SendBlock4Mvcc(cxt context.Context, env *fpga.Block4Mvcc) (*fpga.MvccResponse, error) {
	logger.Infof("mock fpga  receive a SendBlock4Mvcc request")
        
        // compose block_rpc data structure
        BlockRequest bq
        bq.BlockId = 0
        bq.ColdStart = 1
        bq.TxCount = env.num

        txs := make([]*BlockRequest_Transaction, 0, bq.TxCount)
        for i := 0; i < bq.TxCount; i++ {
            rd := make([]*BlockRequest_Transaction_ReadStruct, 0, 1)
            rd[0].Key = env.Txs[i].Rs[0].Key
            rd[0].Version = env.Txs[i].Rs[0].Version

            wt := make([]*BlockRequest_Transaction_WriteStruct, 0, 1)
            wt[0].Key = env.Txs[i].Rs[0].Key
            wt[0].Version = env.Txs[i].Rs[0].Version
            wt[0].IsDel = env.Txs[i].Rs[0].IsDel

            txs[i].IndexInBlock = env.Txs[i].IndexInBlock
            txs[i].Id = env.Txs[i].Id
            txs[i].RdCount = 1
            txs[i].WtCount = 1
            txs[i].Reads = rd
            txs[i].Writes = wt
        }
        
        bq.Transaction = txs
        bq.Crc = 0x1234

        // serialize to stream
        data, err := proto.Marshal(bq)
        if err != nil {
            log.Fatalln("Marshal data error:", err)
        }

	return &fpga.MvccResponse{Result:true}, nil
}

func newServer() *fpgaServer {
	s := &fpgaServer{}
	return s
}

func start() {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	fpga.RegisterFpgaServer(grpcServer, newServer())
	logger.Infof("start a fpga mock server.")
	lis, err := net.Listen("tcp", "0.0.0.0:10000")
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatalf("grpcServer.Serve(lis) failed.")
	}
}
