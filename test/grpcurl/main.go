package main

import (
	"context"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"io"
	"log"
	"os"

	//"os"
	"strings"
)

type multiString []string

func main() {
	ctx := context.Background()
	network := "tcp"
	target := "127.0.0.1:8080"
	var creds credentials.TransportCredentials = nil
	// 创建连接
	var cc *grpc.ClientConn
	cc, err := grpcurl.BlockingDial(ctx, network, target, creds)
	if err != nil {
		panic(err)
	}
	// 通过反射获取接口定义
	var descSource grpcurl.DescriptorSource
	var refClient *grpcreflect.Client

	var addlHeaders multiString
	var reflHeaders multiString
	var rpcHeaders multiString

	log.Println("cc", cc.Target())
	md := grpcurl.MetadataFromHeaders(append(addlHeaders, reflHeaders...))
	refCtx := metadata.NewOutgoingContext(ctx, md)
	refClient = grpcreflect.NewClient(refCtx, reflectpb.NewServerReflectionClient(cc))
	descSource = grpcurl.DescriptorSourceFromServer(ctx, refClient)

	var data string = `{"request": "gRPC"}`
	var in io.Reader
	in = strings.NewReader(data)
	// if not verbose output, then also include record delimiters
	// between each message, so output could potentially be piped
	// to another grpcurl process
	includeSeparators := true
	options := grpcurl.FormatOptions{
		EmitJSONDefaultFields: false,
		IncludeTextSeparator:  includeSeparators,
		AllowUnknownFields:    false,
	}

	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, descSource, in, options)
	if err != nil {
		panic(err)
	}

	h := &grpcurl.DefaultEventHandler{
		Out:            os.Stdout,
		Formatter:      formatter,
		VerbosityLevel: 0,
	}

	var symbol string = "proto.SearchService/Search"
	err = grpcurl.InvokeRPC(ctx, descSource, cc, symbol, append(addlHeaders, rpcHeaders...), h, rf.Next)
	if err != nil {
		log.Println("err", err)
	}
}
