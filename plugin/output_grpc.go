package plugin

import (
	"context"
	"fmt"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/patrickmn/go-cache"
	"github.com/vearne/grpcreplay/protocol"
	slog "github.com/vearne/simplelog"
	"google.golang.org/grpc"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"os"
	"strings"
)

type DescSrcWrapper struct {
	descSource grpcurl.DescriptorSource
	innerCache *cache.Cache
}

func NewDescSrcWrapper(descSource grpcurl.DescriptorSource) *DescSrcWrapper {
	var s DescSrcWrapper
	s.descSource = descSource
	s.innerCache = cache.New(cache.NoExpiration, cache.NoExpiration)
	return &s
}

func (s *DescSrcWrapper) ListServices() ([]string, error) {
	if value, exist := s.innerCache.Get("ListServices"); exist {
		return (value).([]string), nil
	}

	itemList, err := s.descSource.ListServices()
	if err != nil {
		return nil, err
	}

	s.innerCache.Set("ListServices", itemList, cache.NoExpiration)
	return itemList, nil
}

func (s *DescSrcWrapper) FindSymbol(fullyQualifiedName string) (desc.Descriptor, error) {
	key := "fullyQualifiedName:" + fullyQualifiedName

	if value, exist := s.innerCache.Get(key); exist {
		return (value).(desc.Descriptor), nil
	}

	descriptor, err := s.descSource.FindSymbol(fullyQualifiedName)
	if err != nil {
		return nil, err
	}

	s.innerCache.Set(key, descriptor, cache.NoExpiration)
	return descriptor, nil
}

func (s *DescSrcWrapper) AllExtensionsForType(typeName string) ([]*desc.FieldDescriptor, error) {
	key := "AllExtensionsForType:" + typeName

	if value, exist := s.innerCache.Get(key); exist {
		return (value).([]*desc.FieldDescriptor), nil
	}

	descriptors, err := s.descSource.AllExtensionsForType(typeName)
	if err != nil {
		return nil, err
	}

	s.innerCache.Set(key, descriptors, cache.NoExpiration)
	return descriptors, nil
}

type GRPCOutput struct {
	descSource grpcurl.DescriptorSource
	cc         *grpc.ClientConn
	msgChannel chan *protocol.Message
}

func NewGRPCOutput(addr string, workerNum int) *GRPCOutput {
	var err error
	var o GRPCOutput

	ctx := context.Background()
	network := "tcp"
	o.cc, err = grpcurl.BlockingDial(ctx, network, addr, nil)
	if err != nil {
		slog.Fatal("grpcurl.BlockingDial :%v", err)
	}
	// 通过反射获取接口定义
	// *grpcreflect.Client
	var refClient = grpcreflect.NewClientV1Alpha(ctx, reflectpb.NewServerReflectionClient(o.cc))
	o.descSource = NewDescSrcWrapper(grpcurl.DescriptorSourceFromServer(ctx, refClient))

	o.msgChannel = make(chan *protocol.Message, 100)

	for i := 0; i < workerNum; i++ {
		worker := NewGrpcWorker(addr, o.msgChannel, o.descSource)
		go worker.execute()
	}

	slog.Info("create grpc output, addr:%v", addr)
	return &o
}

func (o *GRPCOutput) Close() error {
	close(o.msgChannel)
	return o.cc.Close()
}

func (o *GRPCOutput) Write(msg *protocol.Message) (err error) {
	o.msgChannel <- msg
	return nil
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

type GrpcWorker struct {
	msgChannel chan *protocol.Message
	descSource grpcurl.DescriptorSource
	cc         *grpc.ClientConn
}

func NewGrpcWorker(addr string, msgChannel chan *protocol.Message, descSource grpcurl.DescriptorSource) *GrpcWorker {
	var err error
	var w GrpcWorker
	w.msgChannel = msgChannel
	w.descSource = descSource

	w.cc, err = grpcurl.BlockingDial(context.Background(), "tcp", addr, nil)
	if err != nil {
		slog.Fatal("grpcurl.BlockingDial :%v", err)
	}

	return &w
}

func (w *GrpcWorker) execute() {
	for msg := range w.msgChannel {
		err := w.Call(msg)
		if err != nil {
			slog.Error("Call, message:%v, error:%v", msg.Data.Method, err)
		}
	}
}

func (w *GrpcWorker) Call(msg *protocol.Message) (err error) {
	if len(msg.Data.Method) <= 0 {
		slog.Error("invalid msg:%v", msg)
		return fmt.Errorf("invalid msg:%v", msg)
	}

	in := strings.NewReader(msg.Data.Request)

	slog.Debug("Request:%v", msg.Data.Request)
	// if not verbose output, then also include record delimiters
	// between each message, so output could potentially be piped
	// to another grpcurl process
	options := grpcurl.FormatOptions{
		EmitJSONDefaultFields: false,
		IncludeTextSeparator:  true,
		AllowUnknownFields:    false,
	}
	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, w.descSource, in, options)
	if err != nil {
		slog.Fatal("grpcurl.RequestParserAndFormatter :%v", err)
	}

	h := &grpcurl.DefaultEventHandler{
		Out:            os.Stdout,
		Formatter:      formatter,
		VerbosityLevel: 0,
	}

	symbol := msg.Data.Method
	// /proto.SearchService/Search  ->  proto.SearchService/Search
	if strings.HasPrefix(msg.Data.Method, "/") {
		symbol = symbol[1:]
	}

	headers := convertHeader(msg)
	err = grpcurl.InvokeRPC(context.Background(), w.descSource, w.cc, symbol, headers, h, rf.Next)
	return err
}
