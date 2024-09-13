package main

import (
	"context"
	"encoding/json"
	pb "github.com/vearne/grpcreplay/example/service_proto"
	"github.com/vearne/grpcreplay/example/service_proto/another"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip" // Registration of gzip Compressor will be completed
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"strconv"
	"time"
)

const address = "127.0.0.1:35001"

func main() {
	conn, err := grpc.Dial(address,
		grpc.WithInsecure(),
		//grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")),
	)
	if err != nil {
		log.Fatalf("grpc.Dial err: %v", err)
	}
	defer conn.Close()

	client := pb.NewSearchServiceClient(conn)
	counter := 0
	timeCounter := 0
	muchCounter := 0
	for i := 0; i < 1000000; i++ {
		//value := rand.Intn(100)
		//if value <= 5 {
		//	counter++
		//	sendSearch(client, counter)
		//} else if value <= 20 {
		//muchCounter++
		//	sendMuch(client, uint64(muchCounter))
		//} else {
		//	timeCounter++
		//	sendCurrTime(client, uint64(timeCounter))
		//}
		counter++
		sendSearch(client, counter)
		timeCounter++
		sendCurrTime(client, uint64(timeCounter))
		muchCounter++
		sendMuch(client, muchCounter)
		//time.Sleep(100 * time.Millisecond)
		time.Sleep(3 * time.Second)
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
				Department: &another.Department{Id: 2000, Name: "backend"},
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

func sendCurrTime(client pb.SearchServiceClient, id uint64) {
	md := metadata.New(map[string]string{
		"testkey3": "testvalue3",
		"testkey4": "testvalue4",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.CurrentTime(
		ctx,
		&pb.TimeRequest{RequestId: id},
	)
	if err != nil {
		log.Fatalf("client.CurrentTime err: %v", err)
	}
	bt, _ := json.Marshal(resp)
	log.Println("resp:", string(bt))
}

func sendMuch(client pb.SearchServiceClient, id int) {
	md := metadata.New(map[string]string{
		"testkey5": "testvalue5",
		"testkey6": "testvalue6",
	})
	req := &pb.MuchRequest{RequestId: uint64(id)}
	req.Books = make([]*pb.Book, 1000)
	for i := 0; i < 1000; i++ {
		req.Books[i] = &pb.Book{Name: "name" + strconv.Itoa(id*1000+i), Id: int32(id)}
	}
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.SendMuchData(
		ctx,
		req,
	)
	if err != nil {
		log.Fatalf("client.SendMuchData err: %v", err)
	}

	log.Println("id", id, resp.RequestId)
	bt, _ := json.Marshal(resp)
	log.Println("resp:", string(bt))
}
