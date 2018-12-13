package fpga

import (
    "fmt"
    "context"

    "google.golang.org/grpc"
)


const (
    address = "localhost:50051"
    defaultName = "accelor"
)

int blocksize = 10
uint64 blockid = 0
uint64 txid = 0

func SendBlockSize4vscc(BlockDataSize4vscc size) {
    blocksize = size
}

func VerifySig4vscc(VsccEnvelope *env) {
    BlockRequest bq

    bq.set_block_id(blockid++)
    bq.set_cold_start(0)
    bq.set_tx_count(blocksize)
    
    {
        BlockRequest_Transaction* tx = bq.add_tx()
        tx->set_tx_id(txid++)
    }

    bq.set_crc(0x1234)

    r, err := fpgaService(blockRequest)
    return r
}

func SendBlock4mvcc(Block4mvcc *block) {
    BlockRequest bq

    bq.set_block_id(blockid++) // ask if fabrc could provide one
    bq.set_cold_start(0) // cold start is coming with write-only data
    bq.set_tx_count(block->num) // num is txCount?

    {
        BlockRequest_Transacton* tx = bq.add_tx()
        tx->set_tx_id(txid++) // ask if fabrc could provde one
    }

    bq.set_crc(0x1234) // compute crc here? or in drver ?   

    r, err := fpgaService(bq)
    return r
}

// Send blockRequest t FGPA driver
func fpgaService(BlockRequest block) {
    conn, err := grpc.Dail(address, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }

    defer conn.close()

    c := pb.NewBlockRPCClient(conn)

    name := defaultName
	
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)

    defer cancel()
    r, err := c.SendBlockData(ctx, &pb.BlockRequest{Name: name})

    return r
}
