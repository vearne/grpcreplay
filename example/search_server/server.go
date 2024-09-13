package main

import (
	"context"
	"encoding/json"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	pb "github.com/vearne/grpcreplay/example/service_proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip" // Registration of gzip Compressor will be completed
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"runtime/debug"
	"time"
)

const address = "127.0.0.1:35001"

type SearchServer struct{}

func (s SearchServer) SendMuchData(ctx context.Context, request *pb.MuchRequest) (*pb.MuchResponse, error) {
	return &pb.MuchResponse{RequestId: request.RequestId}, nil
}

func (s SearchServer) Search(ctx context.Context, in *pb.SearchRequest) (*pb.SearchResponse, error) {
	return &pb.SearchResponse{StaffID: 100, StaffName: in.StaffName}, nil
}

func (s SearchServer) CurrentTime(ctx context.Context, request *pb.TimeRequest) (*pb.TimeResponse, error) {
	return &pb.TimeResponse{CurrentTime: time.Now().Format(time.RFC3339)}, nil
}

func main() {
	opts := []grpc.ServerOption{
		//grpc.Creds(c),
		grpc_middleware.WithUnaryServerChain(
			RecoveryInterceptor,
			LoggingInterceptor,
		),
		//grpc.HeaderTableSize(0),
		//grpc.WithDisableRetry(),
	}

	server := grpc.NewServer(opts...)
	pb.RegisterSearchServiceServer(server, &SearchServer{})

	// 注册反射服务
	// Register reflection service on gRPC server.
	reflection.Register(server)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}

	server.Serve(lis)
}

func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Printf("gRPC method: %s, %v", info.FullMethod, req)
	resp, err := handler(ctx, req)
	bt, _ := json.Marshal(req)
	log.Println("body", string(bt))
	log.Printf("gRPC method: %s, %v", info.FullMethod, resp)
	return resp, err
}

func RecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			debug.PrintStack()
			err = status.Errorf(codes.Internal, "Panic err: %v", e)
		}
	}()

	return handler(ctx, req)
}
