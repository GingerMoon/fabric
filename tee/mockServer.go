package tee

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/tee"
	"google.golang.org/grpc"
	"math/big"
	"net"
	"os"
)

var logger = flogging.MustGetLogger("tee")

func init() {
	if os.Getenv("TEE_FPGA_MOCK") == "1" {
		go start()
	}
}

type fpgaServer struct {
	datakey []byte
}

func fromBase10(base10 string) *big.Int {
	i, ok := new(big.Int).SetString(base10, 10)
	if !ok {
		panic("bad number: " + base10)
	}
	return i
}

func (s *fpgaServer) ExchangeDataKey(ctx context.Context, args *pb.DataKeyArgs) (*empty.Empty, error) {

	var test2048Key = &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: fromBase10("14314132931241006650998084889274020608918049032671858325988396851334124245188214251956198731333464217832226406088020736932173064754214329009979944037640912127943488972644697423190955557435910767690712778463524983667852819010259499695177313115447116110358524558307947613422897787329221478860907963827160223559690523660574329011927531289655711860504630573766609239332569210831325633840174683944553667352219670930408593321661375473885147973879086994006440025257225431977751512374815915392249179976902953721486040787792801849818254465486633791826766873076617116727073077821584676715609985777563958286637185868165868520557"),
			E: 3,
		},
		D: fromBase10("9542755287494004433998723259516013739278699355114572217325597900889416163458809501304132487555642811888150937392013824621448709836142886006653296025093941418628992648429798282127303704957273845127141852309016655778568546006839666463451542076964744073572349705538631742281931858219480985907271975884773482372966847639853897890615456605598071088189838676728836833012254065983259638538107719766738032720239892094196108713378822882383694456030043492571063441943847195939549773271694647657549658603365629458610273821292232646334717612674519997533901052790334279661754176490593041941863932308687197618671528035670452762731"),
		Primes: []*big.Int{
			fromBase10("130903255182996722426771613606077755295583329135067340152947172868415809027537376306193179624298874215608270802054347609836776473930072411958753044562214537013874103802006369634761074377213995983876788718033850153719421695468704276694983032644416930879093914927146648402139231293035971427838068945045019075433"),
			fromBase10("109348945610485453577574767652527472924289229538286649661240938988020367005475727988253438647560958573506159449538793540472829815903949343191091817779240101054552748665267574271163617694640513549693841337820602726596756351006149518830932261246698766355347898158548465400674856021497190430791824869615170301029"),
		},
	}
	test2048Key.Precompute()

	rng := rand.Reader

	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, test2048Key, args.Datakey, args.Label)
	s.datakey = plaintext
	if err != nil {
		logger.Errorf("Error from rsa decryption: %s\n", err)
		return &empty.Empty{}, err
	}
	logger.Infof("The received AES datakey is: %s\n", base64.StdEncoding.EncodeToString(s.datakey))
	return &empty.Empty{}, nil
}

func (s *fpgaServer) Execute(ctx context.Context, args *pb.TeeArgs) (*pb.PlainCiphertexts, error) {

	// decrypte the TEE execution args
	ciphertextArgs, err := s.decryptExecuteArgs(args.PlainCipherTexts.Feed4Decryptions)
	if err != nil {
		return nil, err
	}

	// execution
	elf := string(args.Elf)
	if elf == "paymentCCtee" {
		plaintexts, plaintexts2encrypted, err := paymentCCtee(args.PlainCipherTexts.Plaintexts[:], ciphertextArgs)
		if err != nil {
			return nil, err
		}
		// encrypt the confidential results. due to the blockchain rules, different HWs must use the same nonces for the encryption here
		feed4Decryptions, err := s.encryptReturnedArgs(plaintexts2encrypted, args.Nonces)
		if err != nil {
			return nil, err
		}
		return &pb.PlainCiphertexts{Plaintexts:plaintexts, Feed4Decryptions:feed4Decryptions}, nil
	} else {
		return nil, errors.New(fmt.Sprintf("unsupported function call. ELF: %s", elf))
	}

	return nil, errors.New(fmt.Sprintf("unexpected Execute result"))
}

func (s *fpgaServer) decryptExecuteArgs(args []*pb.Feed4Decryption) ([][]byte, error) {
	results := make([][]byte, len(args))
	for i, arg := range args {
		block, err := aes.NewCipher(s.datakey)
		if err != nil {
			return nil, err
		}

		aesgcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}

		plaintext, err := aesgcm.Open(nil, arg.Nonce, arg.Ciphertext, nil)
		if err != nil {
			return nil, err
		}
		results[i] = plaintext
	}
	return results, nil
}

func (s *fpgaServer) encryptReturnedArgs(plaintexts, nonces [][]byte) ([]*pb.Feed4Decryption, error) {
	if len(nonces) != len(plaintexts) {
		return nil, errors.New(fmt.Sprintf(
			"there is not enough nonces for encrypting the execution results. Expectd: %d nonces, actually got %d nonces",
			len(plaintexts), len(nonces)))
	}

	results := make([]*pb.Feed4Decryption, len(plaintexts))
	for i, plaintext := range plaintexts {
		block, err := aes.NewCipher(s.datakey)
		if err != nil {
			panic(err.Error())
		}

		aesgcm, err := cipher.NewGCM(block)
		if err != nil {
			panic(err.Error())
		}

		ciphertext := aesgcm.Seal(nil, nonces[i], plaintext, nil)
		results[i] = &pb.Feed4Decryption{Ciphertext:ciphertext, Nonce:nonces[i]}
	}
	return results, nil
}

func paymentCCtee(plaintextArgs [][]byte, ciphertextArgs [][]byte) (plaintext, ciphertext [][]byte, err error) {
	if len(ciphertextArgs) != 3 {
		return nil, nil, errors.New("paymentCCtee need 3 args!")
	}
	valueA := binary.BigEndian.Uint32(ciphertextArgs[0])
	valueB := binary.BigEndian.Uint32(ciphertextArgs[1])
	x := binary.BigEndian.Uint32(ciphertextArgs[2])

	if int(valueA) - int(x) < 0 {
		logger.Infof("not enough balance! valueA: %d, valueB: %d, x: %d", valueA, valueB, x)
		return nil, nil, errors.New("not enough balance!")
	}
	valueA -= x
	bsA := make([]byte, 4)
	binary.BigEndian.PutUint32(bsA, valueA)
	ciphertext = append(ciphertext, bsA)

	valueB += x
	bsB := make([]byte, 4)
	binary.BigEndian.PutUint32(bsB, valueB)
	ciphertext = append(ciphertext, bsB)

	return nil, ciphertext, nil
}

func newServer() *fpgaServer {
	s := &fpgaServer{}
	return s
}

func start() {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	//fpga.RegisterFpgaServer(grpcServer, newServer())
	pb.RegisterTeeServer(grpcServer, newServer())
	logger.Infof("start a tee fpga mock server.")
	lis, err := net.Listen("tcp", "0.0.0.0:20000")
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatalf("grpcServer.Serve(lis) failed.")
	}
}