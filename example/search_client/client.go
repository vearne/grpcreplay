package main

import (
	"context"
	"encoding/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip" // Registration of gzip Compressor will be completed
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"math/rand"
	"time"

	pb "github.com/vearne/grpcreplay/example/search_proto"
)

const PORT = "35001"

func main() {
	conn, err := grpc.Dial(":"+PORT,
		grpc.WithInsecure(),
		//grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")),
	)
	if err != nil {
		log.Fatalf("grpc.Dial err: %v", err)
	}
	defer conn.Close()

	client := pb.NewSearchServiceClient(conn)
	counter := 0
	for i := 0; i < 1000000; i++ {
		if rand.Intn(1000)%2 == 0 {
			counter++
			sendSearch(client, counter)
		} else {
			sendCurrTime(client)
		}
		time.Sleep(10 * time.Second)
	}
}

func sendSearch(client pb.SearchServiceClient, i int) {
	// add some headers
	md := metadata.New(map[string]string{
		"testkey1": "testvalue1",
		"testkey2": "testvalue2",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.Search(ctx,
		&pb.SearchRequest{
			StaffName: "zhangsan",
			Age:       uint32(i),
			Gender:    true,
			Extra: &pb.ExtraInfo{
				JobTitle:   "software engineer",
				Location:   "Beijing",
				Department: "Back Office Department",
			},
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
}

func sendCurrTime(client pb.SearchServiceClient) {
	md := metadata.New(map[string]string{
		"testkey3": "testvalue3",
		"testkey4": "testvalue4",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.CurrentTime(
		ctx,
		&pb.TimeRequest{},
	)
	if err != nil {
		log.Fatalf("client.CurrentTime err: %v", err)
	}
	bt, _ := json.Marshal(resp)
	log.Println("resp:", string(bt))
}
