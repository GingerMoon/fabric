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
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/tee"
	"google.golang.org/grpc"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"strconv"
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

func (s *fpgaServer) ExchangeDataKey(ctx context.Context, args *pb.DataKeyArgs) (*pb.ErrorInfo, error) {

	errorInfo := &pb.ErrorInfo{Err:""}

	// serialize args to file.
	logger.Errorf("serialize args to file for hw test. begin")
	fname := "auction.tee.ExchangeDataKey.args"
	out, err := proto.Marshal(args)
	if err != nil {
		logger.Fatalf("Failed to encode ExchangeDataKey args:", err)
	}
	if err := ioutil.WriteFile(fname, out, 0644); err != nil {
		logger.Fatalf("Failed to write ExchangeDataKey args:", err)
	}

	in, err := ioutil.ReadFile(fname)
	if err != nil {
		logger.Fatalf("Error reading file:", err)
	}
	tmp := &pb.DataKeyArgs{}
	if err := proto.Unmarshal(in, tmp); err != nil {
		logger.Fatalf("Failed to parse ExchangeDataKey args:", err)
	}
	logger.Errorf("serialize args to file for hw test. end")

	/*
	D,N,E are decided by only two of three.
	Private key: D, N
	Public key: N, E
	D: %s a3cb5fde7b66d79ac4265f9385475f785b37864e8a7dd8e7a0a4d1cb588dfa8996e7d5812b58e1ba521e7e572953883e2d8374c7f706be72136a2c87a6aa970113b2c16d44919b06d6ff41d911a3b053567aa120774626a788fece38fdeabbaa34208aa583e301565956c83b2f9097b89590b6fe29829943d8eba23498f424b08156a46b72d9adef14baa196e83b3c9b5020af3d36fe8113f219a0459fb1119c6fe48bd432643a6b0e06227c722bd0e947f060991c59018d8e771bf50f22714d523c42a6aaf2eb2d1ed35c2051162ff2f340c9c72dd11b11eedc9a6c7e3c82e82cc3302c368da6ad48bc66cf50b522e8888cd766b70a65940a9e50b3b8ca1399
	N: %s a7d134aef43b25bf19bcfcbff61e4a84bcbd62ff31dc2ba93d7768c0977a4f313d6c1d75ac861b880c33a530bc8f171d787e9abc6326c9579d2e3554e5dbec6b9684f06a72d3120d26ad4ba22c0ef5b5ec826f8b2be9ee96e9284010b28ab0211ad135d22138403313ed5722586e1a87e2a546271c5cb349fdf6ffedcb82d60ae9f874a6e1dbfbc5e58cec957ecc5706fdcb03390c496fc436b1359a0df4bab5d0ffa049f040177b17950269e86546274f679e921eda82e6deb761fb624cced8830bfd21c9c14ff77fe6ef0bb11d0653e97be01c48fe79a6433525512f8bbf6a116291c873bea99e4405f72c109d6d42020124e18872d4921c9984ae30c9a3f9
	E: %d 65537
	*/
	N, _ := new(big.Int).SetString("a7d134aef43b25bf19bcfcbff61e4a84bcbd62ff31dc2ba93d7768c0977a4f313d6c1d75ac861b880c33a530bc8f171d787e9abc6326c9579d2e3554e5dbec6b9684f06a72d3120d26ad4ba22c0ef5b5ec826f8b2be9ee96e9284010b28ab0211ad135d22138403313ed5722586e1a87e2a546271c5cb349fdf6ffedcb82d60ae9f874a6e1dbfbc5e58cec957ecc5706fdcb03390c496fc436b1359a0df4bab5d0ffa049f040177b17950269e86546274f679e921eda82e6deb761fb624cced8830bfd21c9c14ff77fe6ef0bb11d0653e97be01c48fe79a6433525512f8bbf6a116291c873bea99e4405f72c109d6d42020124e18872d4921c9984ae30c9a3f9", 16)
	d, _ := new(big.Int).SetString("a3cb5fde7b66d79ac4265f9385475f785b37864e8a7dd8e7a0a4d1cb588dfa8996e7d5812b58e1ba521e7e572953883e2d8374c7f706be72136a2c87a6aa970113b2c16d44919b06d6ff41d911a3b053567aa120774626a788fece38fdeabbaa34208aa583e301565956c83b2f9097b89590b6fe29829943d8eba23498f424b08156a46b72d9adef14baa196e83b3c9b5020af3d36fe8113f219a0459fb1119c6fe48bd432643a6b0e06227c722bd0e947f060991c59018d8e771bf50f22714d523c42a6aaf2eb2d1ed35c2051162ff2f340c9c72dd11b11eedc9a6c7e3c82e82cc3302c368da6ad48bc66cf50b522e8888cd766b70a65940a9e50b3b8ca1399", 16)
	privateKey := rsa.PrivateKey{D:d}
	privateKey.N = N
	privateKey.E = 65537

	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, &privateKey, args.Datakey, args.Label)
	s.datakey = plaintext
	if err != nil {
		logger.Errorf("Error from rsa decryption: %s\n", err)
		return errorInfo, err
	}
	logger.Infof("The received AES datakey is: %s\n", base64.StdEncoding.EncodeToString(s.datakey))
	return errorInfo, nil
}

func (s *fpgaServer) Execute(ctx context.Context, args *pb.TeeArgs) (*pb.PlainCiphertexts, error) {

	// serialize args to file.
	logger.Errorf("serialize TEE args to file for hw test. begin")
	fname := "auction.tee.Execute.args"
	out, err := proto.Marshal(args)
	if err != nil {
		logger.Fatalf("Failed to encode Execute args:", err)
	}
	if err := ioutil.WriteFile(fname, out, 0644); err != nil {
		logger.Fatalf("Failed to write Execute args:", err)
	}

	in, err := ioutil.ReadFile(fname)
	if err != nil {
		logger.Fatalf("Error reading file:", err)
	}
	tmp := &pb.TeeArgs{}
	if err := proto.Unmarshal(in, tmp); err != nil {
		logger.Fatalf("Failed to address Execute args:", err)
	}
	logger.Errorf("serialize TEE args to file for hw test. end")

	// decrypte the TEE execution args
	ciphertextArgs, err := s.decryptExecuteArgs(args.PlainCipherTexts.Feed4Decryptions)
	if err != nil {
		return nil, err
	}

	// execution
	elf := string(args.Elf)
	if elf == "paymentCCtee" {
		plaintexts, plaintexts2encrypted, err := paymentCCtee(args.PlainCipherTexts.Plaintexts[:], ciphertextArgs[1:])
		if err != nil {
			return nil, err
		}
		// encrypt the confidential results. due to the blockchain rules, different HWs must use the same nonces for the encryption here
		feed4Decryptions, err := s.encryptReturnedArgs(plaintexts2encrypted, args.Nonces)
		if err != nil {
			return nil, err
		}
		return &pb.PlainCiphertexts{Plaintexts:plaintexts, Feed4Decryptions:feed4Decryptions}, nil
	} else if elf == "compare" {
		plaintexts, err := compare(ciphertextArgs)
		if err != nil {
			return nil, err
		}
		return &pb.PlainCiphertexts{Plaintexts:plaintexts, Feed4Decryptions:nil}, nil
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



//输入参数: plaintextArgs 在这个demo里是空的. ciphertextArgs里的内容是经过硬件data key解密的.
//返回值: plaintext代表不需要硬件加密. ciphertext代表硬件需要对其中的元素加密.
//两个demo中, 数字分别用字节数组和字符串表示. 看硬件实现起来哪个方便就用哪个. 我可以做相应修改.
//
//****************************************************************

// payment demo
// A 给 B 转账. ciphertextArgs[0]是解密后A的余额, ciphertextArgs[1] 是解密后B的余额. ciphertextArgs[2] 是要转账的钱的数量x.
// 业务逻辑就是, A的余额必须大于x, 然后valueA = valueA-x, valueB = valueB+x, 将新的valueA和valueB放到ciphertext返回回去. 硬件接下来会对ciphertext里的内容用datakey加密.
// 使用大端字节序
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

// auction demo
// 比较 ciphertextArgs[0] 和 ciphertextArgs[1] 的大小
// 此demo中的数字用字符串表示.
func compare(ciphertextArgs [][]byte) (plaintext [][]byte, err error) {
	if len(ciphertextArgs) != 2 {
		return nil, errors.New("compare need 2 args!")
	}
	valueA, err := strconv.Atoi(string(ciphertextArgs[0]))
	if err != nil {
		return nil, errors.New("compare operand[0] should be number!")
	}
	valueB, err := strconv.Atoi(string(ciphertextArgs[1]))
	if err != nil {
		return nil, errors.New("compare operand[1] should be number!")
	}

	if valueA == valueB {
		plaintext = append(plaintext, []byte("0"))
	} else if valueA > valueB {
		plaintext = append(plaintext, []byte("1"))
	} else {
		plaintext = append(plaintext, []byte("-1"))
	}

	return
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