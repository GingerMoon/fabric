package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"github.com/hyperledger/fabric/common/flogging"
	pb "github.com/hyperledger/fabric/protos/tee"
	"google.golang.org/grpc"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"sync"
	"time"
)

var logger = flogging.MustGetLogger("limin")

var datakey = []byte {
	0xee, 0xbc, 0x1f, 0x57, 0x48, 0x7f, 0x51, 0x92, 0x1c, 0x04, 0x65, 0x66,
	0x5f, 0x8a, 0xe6, 0xd1, 0x65, 0x8b, 0xb2, 0x6d, 0xe6, 0xf8, 0xa0, 0x69,
	0xa3, 0x52, 0x02, 0x93, 0xa5, 0x72, 0x07, 0x8f,
}

func createTeeClient() pb.TeeClient {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial("localhost:50121", opts...)
	if err != nil {
		logger.Fatalf("fail to dial: %v", err)
	}
	//defer conn.Close()
	return pb.NewTeeClient(conn)
}

func exchangeDataKey() {
	N, _ := new(big.Int).SetString("ae45ed5601cec6b8cc05f803935c674ddbe0d75c4c09fd7951fc6b0caec313a8df39970c518bffba5ed68f3f0d7f22a4029d413f1ae07e4ebe9e4177ce23e7f5404b569e4ee1bdcf3c1fb03ef113802d4f855eb9b5134b5a7c8085adcae6fa2fa1417ec3763be171b0c62b760ede23c12ad92b980884c641f5a8fac26bdad4a03381a22fe1b754885094c82506d4019a535a286afeb271bb9ba592de18dcf600c2aeeae56e02f7cf79fc14cf3bdc7cd84febbbf950ca90304b2219a7aa063aefa2c3c1980e560cd64afe779585b6107657b957857efde6010988ab7de417fc88d8f384c4e6e72c3f943e0c31c0c4a5cc36f879d8a3ac9d7d59860eaada6b83bb", 16)
	publicKey := rsa.PublicKey{N, 0x10001}

	logger.Infof("the AES datakey is: %s. len: %d.", hex.EncodeToString(datakey), len(datakey))
	// shenyaming, 2019/06/04
	// TEE only support that  label is an empty string.
	//label := []byte("label_abcflabel_abcf")
	// And the corresponding hash value lHash of the empty string is :
	// SHA-1: (0x)da39a3ee 5e6b4b0d 3255bfef 95601890 afd80709

	label := []byte{}

	// Exmaple 1: only when label is an empty string.
	// label_hash := []byte{0xda,0x39,0xa3,0xee,0x5e,0x6b,0x4b,0x0d,0x32,0x55,0xbf,0xef,0x95,0x60,0x18,0x90,0xaf,0xd8,0x07,0x09};

	// Example 2: when label is
	label_hash := sha1.New().Sum(label)

	rng := rand.Reader

	ciphertext, err := rsa.EncryptOAEP(sha1.New(), rng, &publicKey, datakey, label)

	d := new(big.Int)
	d.SetString("56b04216fe5f354ac77250a4b6b0c8525a85c59b0bd80c56450a22d5f438e596a333aa875e291dd43f48cb88b9d5fc0d499f9fcd1c397f9afc070cd9e398c8d19e61db7c7410a6b2675dfbf5d345b804d201add502d5ce2dfcb091ce9997bbebe57306f383e4d588103f036f7e85d1934d152a323e4a8db451d6f4a5b1b0f102cc150e02feee2b88dea4ad4c1baccb24d84072d14e1d24a6771f7408ee30564fb86d4393a34bcf0b788501d193303f13a2284b001f0f649eaf79328d4ac5c430ab4414920a9460ed1b7bc40ec653e876d09abc509ae45b525190116a0c26101848298509c1c3bf3a483e7274054e15e97075036e989f60932807b5257751e79", 16)
	private := new(rsa.PrivateKey)
	private.PublicKey = rsa.PublicKey{publicKey.N, publicKey.E}
	private.D = d

	out, out_err    := rsa.DecryptOAEP(sha1.New(), nil, private, ciphertext, nil)
	if out_err != nil {
		return
	}
	logger.Infof("Out(RSA) of AES datakey: %s. len is %d", hex.EncodeToString(out), len(out))

	if err != nil {
		logger.Errorf("Error from encryption: %s", err)
		return
	}
	logger.Infof("Ciphertext(RSA) of AES datakey: %s. len is %d", hex.EncodeToString(ciphertext), len(ciphertext))
	logger.Infof("RSA label is: label_abc. hex encoded string is: %s. len is %d", hex.EncodeToString(label_hash), len(label_hash))

	logger.Errorf("洪明希望不要发送exchangedatakey rpc, 所以先暂时不发送rpc. 如果洪明需要测试这个rpc, 李明请把当前这条log信息下的代码注释去掉即可.")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := createTeeClient().ExchangeDataKey(ctx, &pb.DataKeyArgs{Datakey:ciphertext, Label:label_hash})
	if err != nil {
		logger.Errorf("exchange datakey failed. err: %s", err.Error())
	} else {
		logger.Infof("exchange datakey succeeded! response: %s", resp.Result)
	}
}

type encryptedContent struct {
	Content []byte `json:content`
	Nonce   []byte `json:nonce` // used for decrypting amount
}

func aesEncrypt(plaintext []byte) *encryptedContent {
	block, err := aes.NewCipher(datakey)
	if err != nil {
		panic(err.Error())
	}

	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	logger.Infof("AES 明文: %s. len: %d", hex.EncodeToString(plaintext), len(plaintext))
	logger.Infof("AES 密文: %s. len: %d", hex.EncodeToString(ciphertext), len(ciphertext))
	logger.Infof("AES nonce: %s. len: %d", hex.EncodeToString(nonce), len(nonce))
	//logger.Infof("plaintext before aes encrypted is: %s, aes encrypted ciphertext is: %s, nonce is: %s",
	//	hex.EncodeToString(plaintext), hex.EncodeToString(ciphertext), hex.EncodeToString(nonce))
	return &encryptedContent{ciphertext, nonce}
}

func aesDecrypt(ciphertext, nonce []byte) []byte {
	block, err := aes.NewCipher(datakey)
	if err != nil {
		panic(err.Error())
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		logger.Fatalf("aesDecrypt (ciphertext: %s, nonce: %s) - aesgcm.Open failed. error: %s", hex.EncodeToString(ciphertext), hex.EncodeToString(nonce), err.Error())
	}
	return plaintext
}

func getCiphertextOfData() (balance, x, elf *encryptedContent) {
	logger.Infof("AES加密余额 100 ")
	plaintextBalance := make([]byte, 16)
	binary.BigEndian.PutUint32(plaintextBalance, 100)
	balance = aesEncrypt(plaintextBalance)

	logger.Infof("AES加密余额 80 ")
	plaintextX := make([]byte, 16)
	binary.BigEndian.PutUint32(plaintextX, uint32(80))
	x = aesEncrypt(plaintextX)

	plaintextElf, err := ioutil.ReadFile("./elf_payment.bin")
	if err != nil {
		panic(err.Error())
	}
	paddingCount := (len(plaintextElf) / 32 + 1) * 32 - len(plaintextElf) // elf/hex has to be integral multiples of 256 bits/32 bytes
	for i := 0; i < paddingCount; i++ {
		plaintextElf = append(plaintextElf, 0)
	}

	// yaming elf need an extra reverse operation due to the hw bug
	var plaintextBin []byte
	for i := 0; i < len(plaintextElf); i = i + 8 {
		//	fmt.Println("==============")
		//	fmt.Println(i)
		for j := 0; j < 8; j++ {
			plaintextBin = append(plaintextBin, plaintextElf[i+7-j])
			//  fmt.Println(plaintextElf[i+j])
		}

	}
	//logger.Infof("AES加密ELF...")
	elf = aesEncrypt(plaintextBin)
	//elf = aesEncrypt([]byte(hex.EncodeToString(plaintextElf)))
	return
}

func testExecute(wg *sync.WaitGroup) {
	defer wg.Done()
	nonces := make([][]byte, 2)
	for i:=0; i < 2; i++ {
		nonce := make([]byte, 12)
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			panic(err.Error())
		}
		nonces[i] = nonce
	}

	balance, x, elf := getCiphertextOfData()

	var feed4decrytions []*pb.Feed4Decryption
	feed4decrytions = append(feed4decrytions, &pb.Feed4Decryption{Ciphertext:elf.Content, Nonce:elf.Nonce})
	feed4decrytions = append(feed4decrytions, &pb.Feed4Decryption{Ciphertext:balance.Content, Nonce:balance.Nonce})
	feed4decrytions = append(feed4decrytions, &pb.Feed4Decryption{Ciphertext:balance.Content, Nonce:balance.Nonce})
	feed4decrytions = append(feed4decrytions, &pb.Feed4Decryption{Ciphertext:x.Content, Nonce:x.Nonce})

	in := &pb.TeeArgs{
		Elf:[]byte("tee_execute_method_name"),
		PlainCipherTexts:&pb.PlainCiphertexts{
			Plaintexts:nil,
			Feed4Decryptions:feed4decrytions,
		},
		Nonces:nonces,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	response, err := createTeeClient().Execute(ctx, in)
	if err != nil {
		logger.Fatalf("createTeeClient().Execute(ctx, in) return error: %s", err.Error())
	}

	balanceABytes := aesDecrypt(response.Feed4Decryptions[0].Ciphertext, response.Feed4Decryptions[0].Nonce)
	balanceA := binary.LittleEndian.Uint32(balanceABytes)
	balanceBBytes := aesDecrypt(response.Feed4Decryptions[1].Ciphertext, response.Feed4Decryptions[1].Nonce)
	balanceB := binary.LittleEndian.Uint32(balanceBBytes)

	if balanceA != 20 || balanceB != 180 {
		logger.Errorf("test failed!!! transfer 80 from accountA(balanceA: 100) to accountB(balanceB: 100), the expected result is, balanceA: 20, balanceB: 180")
		logger.Errorf("but the result is: balanceA: %v, balanceB: %v", balanceA, balanceB)
		logger.Errorf("encrypted elf hex: %s. the nonce is: %s.", hex.EncodeToString(elf.Content), hex.EncodeToString(elf.Nonce))
		logger.Errorf("tee args are below:")
		for i, e := range in.Nonces {
			logger.Errorf("tee.args.nonces[%d] for hw encryption: %s", i, hex.EncodeToString(e))
		}
		for i, e := range in.PlainCipherTexts.Feed4Decryptions {
			logger.Errorf("tee.args.PlainCipherTexts.Feed4Decryptions[%d]: [ciphtertext %s], [nonce %s]", i, hex.EncodeToString(e.Ciphertext), hex.EncodeToString(e.Nonce))
		}
		for i, e := range response.Feed4Decryptions {
			logger.Errorf("received tee execute response.Feed4Decryptions[%d]: [ciphtertext %s], [nonce %s]", i, hex.EncodeToString(e.Ciphertext), hex.EncodeToString(e.Nonce))
		}
	} else {
		logger.Infof("execute rpc succeeded!")
	}

	//if bytes.Compare(feed4decrytions[1].Ciphertext, response.Feed4Decryptions[0].Ciphertext) != 0 ||
	//	bytes.Compare(feed4decrytions[1].Nonce, response.Feed4Decryptions[0].Nonce) != 0 {
	//		logger.Errorf("response.Feed4Decryptions[0] (ciphertext: %s, nonce: %s) does equal to feed4decrytions[1]",
	//			response.Feed4Decryptions[0].Ciphertext, response.Feed4Decryptions[0].Nonce)
	//}
	//
	//if bytes.Compare(feed4decrytions[2].Ciphertext, response.Feed4Decryptions[1].Ciphertext) != 0 ||
	//	bytes.Compare(feed4decrytions[2].Nonce, response.Feed4Decryptions[1].Nonce) != 0 {
	//		logger.Errorf("response.Feed4Decryptions[1] (ciphertext: %s, nonce: %s) does equal to feed4decrytions[2]",
	//			response.Feed4Decryptions[1].Ciphertext, response.Feed4Decryptions[1].Nonce)
	//}
	//logger.Infof("execute rpc succeeded!")
}

func main() {

	if len(os.Args) != 2 {
		logger.Fatalf("please set the thread number as the command line arg!")
	}
	threaNum, err := strconv.Atoi(os.Args[1])
	if err != nil {
		logger.Fatalf("please set the thread number (%s) correctly.", os.Args[1])
	}

	logger.Info("testing exchangeDataKey")
	exchangeDataKey()

	wg := sync.WaitGroup{}
	wg.Add(threaNum)
	logger.Infof("testing execute rpc concurrently ")
	for i := 0; i < threaNum; i++ {
		go testExecute(&wg)
	}
	wg.Wait()
	logger.Infof("test execute rpc concurrently finished.")
}
