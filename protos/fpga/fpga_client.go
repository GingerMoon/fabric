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


func fpgaService() {
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
}
