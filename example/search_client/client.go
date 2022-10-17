package main

import (
	"context"
	"encoding/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"time"

	pb "github.com/vearne/grpcreplay/example/search_proto"
)

const PORT = "35001"

func main() {
	conn, err := grpc.Dial(":"+PORT, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("grpc.Dial err: %v", err)
	}
	defer conn.Close()

	// add some headers
	md := metadata.New(map[string]string{
		"testkey1": "testvalue1",
		"testkey2": "testvalue2",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	client := pb.NewSearchServiceClient(conn)
	for i := 0; i < 1000000; i++ {
		resp, err := client.Search(ctx,
			&pb.SearchRequest{
				StaffName: "zhangsan",
				Age:       uint32(i),
				Gender:    true,
			},
		)
		if err != nil {
			statusErr, ok := status.FromError(err)
			if ok {
				if statusErr.Code() == codes.DeadlineExceeded {
					log.Fatalln("client.Search err: deadline")
				}
			}

			log.Fatalf("client.Search err: %v", err)
		}

		bt, _ := json.Marshal(resp)
		log.Println("resp:", string(bt))
		time.Sleep(10 * time.Second)
	}
}
