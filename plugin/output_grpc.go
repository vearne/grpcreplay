package plugin

import (
	"context"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/vearne/grpcreplay/protocol"
	slog "github.com/vearne/simplelog"
	"google.golang.org/grpc"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"io"
	"log"
	"os"
	"strings"
)

type GRPCOutput struct {
	descSource grpcurl.DescriptorSource
	cc         *grpc.ClientConn
}

func NewGRPCOutput(addr string) *GRPCOutput {
	var err error
	var o GRPCOutput

	ctx := context.Background()
	network := "tcp"
	o.cc, err = grpcurl.BlockingDial(ctx, network, addr, nil)
	if err != nil {
		log.Fatal("grpcurl.BlockingDial :%v", err)
	}
	// 通过反射获取接口定义
	var refClient *grpcreflect.Client

	refClient = grpcreflect.NewClient(ctx, reflectpb.NewServerReflectionClient(o.cc))
	o.descSource = grpcurl.DescriptorSourceFromServer(ctx, refClient)

	slog.Info("create grpc output, addr:%v", addr)
	return &o
}

func (o *GRPCOutput) Close() error {
	return o.cc.Close()
}

func (o *GRPCOutput) Write(msg *protocol.Message) (err error) {
	var in io.Reader
	in = strings.NewReader(msg.Data.Request)

	slog.Debug("Request:%v", msg.Data.Request)
	// if not verbose output, then also include record delimiters
	// between each message, so output could potentially be piped
	// to another grpcurl process
	options := grpcurl.FormatOptions{
		EmitJSONDefaultFields: false,
		IncludeTextSeparator:  true,
		AllowUnknownFields:    false,
	}
	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, o.descSource, in, options)
	if err != nil {
		log.Fatal("grpcurl.RequestParserAndFormatter :%v", err)
	}

	h := &grpcurl.DefaultEventHandler{
		Out:            os.Stdout,
		Formatter:      formatter,
		VerbosityLevel: 0,
	}

	// /proto.SearchService/Search  ->  proto.SearchService/Search
	symbol := msg.Data.Method[1:]

	headers := convertHeader(msg)
	err = grpcurl.InvokeRPC(context.Background(), o.descSource, o.cc, symbol, headers, h, rf.Next)
	return err
}

func convertHeader(msg *protocol.Message) (headers []string) {
	headers = make([]string, 0, len(msg.Data.Headers))
	for key, value := range msg.Data.Headers {
		if !IsPseudo(key) {
			headers = append(headers, key+":"+value)
		}
	}
	return headers
}

func IsPseudo(key string) bool {
	return strings.HasPrefix(key, ":")
}
