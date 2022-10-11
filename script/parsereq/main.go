package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/proto"

	//"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"strings"
)

func main() {
	network := "tcp"
	target := "127.0.0.1:8080"
	svcAndMethod := "/proto.SearchService/Search"

	ctx := context.Background()

	var cc *grpc.ClientConn
	cc, err := grpcurl.BlockingDial(ctx, network, target, nil)
	if err != nil {
		panic(err)
	}
	md := grpcurl.MetadataFromHeaders(nil)
	refCtx := metadata.NewOutgoingContext(ctx, md)
	refClient := grpcreflect.NewClient(refCtx, reflectpb.NewServerReflectionClient(cc))
	descSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)
	fmt.Println("descSource", descSource)
	svc, method := parseSymbol(svcAndMethod)
	dsc, err := descSource.FindSymbol(svc)
	if err != nil {
		panic(err)
	}
	sd, ok := dsc.(*desc.ServiceDescriptor)
	if !ok {
		panic(err)
	}
	mtd := sd.FindMethodByName(method)
	inputType := mtd.GetInputType()
	fmt.Println("mtd.GetInputType()", inputType.GetName(), inputType.GetFullyQualifiedName())
	p := inputType.AsProto()
	data := []byte{}
	err = proto.Unmarshal(data, p)
	if err != nil {
		panic(err)
	}
	bt, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	fmt.Println("json.Marshal", string(bt))
}

func parseSymbol(svcAndMethod string) (string, string) {
	if svcAndMethod[0] == '/' {
		svcAndMethod = svcAndMethod[1:]
	}
	pos := strings.LastIndex(svcAndMethod, "/")
	if pos < 0 {
		pos = strings.LastIndex(svcAndMethod, ".")
		if pos < 0 {
			return "", ""
		}
	}
	return svcAndMethod[:pos], svcAndMethod[pos+1:]
}
