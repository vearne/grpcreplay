package http2

import (
	"context"
	"fmt"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/desc" //nolint: staticcheck
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/patrickmn/go-cache"
	"github.com/vearne/grpcreplay/util"
	slog "github.com/vearne/simplelog"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"strings"
)

type MethodInputOutput struct {
	InType  proto.Message
	OutType proto.Message
}

type PBFinder interface {
	// svcAndMethod looks like"/helloworld.Greeter/SayHello"
	Get(svcAndMethod string) (*MethodInputOutput, error)
	GetDescriptorSource() grpcurl.DescriptorSource
}

type FilePBFinder struct {
	ds         grpcurl.DescriptorSource
	innerCache *cache.Cache
	protoFiles []string
}

func NewFilePBFinder(protoFiles []string) *FilePBFinder {
	var f FilePBFinder
	var err error
	f.protoFiles = protoFiles
	f.ds, err = grpcurl.DescriptorSourceFromProtoSets(protoFiles...)
	if err != nil {
		slog.Fatal("NewFilePBFinder, %v", err)
	}
	f.innerCache = cache.New(cache.NoExpiration, cache.NoExpiration)
	return &f
}

func (f *FilePBFinder) GetDescriptorSource() grpcurl.DescriptorSource {
	return f.ds
}

func (f *FilePBFinder) Get(svcAndMethod string) (*MethodInputOutput, error) {
	if v, ok := f.innerCache.Get(svcAndMethod); ok {
		slog.Debug("FilePBFinder.Get,svcAndMethod:%v, hit cache", svcAndMethod)

		cached := v.(*MethodInputOutput)
		// return a fresh copy – no shared state
		return &MethodInputOutput{
			InType:  proto.Clone(cached.InType),
			OutType: proto.Clone(cached.OutType),
		}, nil
	}

	m, err := Find(f.ds, svcAndMethod)
	if err != nil {
		slog.Warn("FilePBFinder.Get,svcAndMethod:%v, error:%v", svcAndMethod, err)
		return nil, err
	}

	f.innerCache.Set(svcAndMethod, m, cache.NoExpiration)
	return m, nil
}

type ReflectionPBFinder struct {
	ds         grpcurl.DescriptorSource
	innerCache *cache.Cache
	// server address
	addr string
}

func NewReflectionPBFinder(addr string) *ReflectionPBFinder {
	var f ReflectionPBFinder
	f.innerCache = cache.New(cache.NoExpiration, cache.NoExpiration)
	f.addr = addr

	ctx := context.Background()
	cc, err := grpcurl.BlockingDial(ctx, "tcp", addr, nil)
	if err != nil {
		slog.Fatal("PBMessageFinder.FindMethodInput,addr:%v, error:%v, enable grpc reflection service?",
			f.addr, err)
	}
	refClient := grpcreflect.NewClientV1Alpha(ctx, reflectpb.NewServerReflectionClient(cc))
	f.ds = grpcurl.DescriptorSourceFromServer(ctx, refClient)

	return &f
}

func (f *ReflectionPBFinder) Get(svcAndMethod string) (*MethodInputOutput, error) {
	if v, ok := f.innerCache.Get(svcAndMethod); ok {
		slog.Debug("ReflectionPBFinder.Get,svcAndMethod:%v, hit cache", svcAndMethod)

		cached := v.(*MethodInputOutput)
		// return a fresh copy – no shared state
		return &MethodInputOutput{
			InType:  proto.Clone(cached.InType),
			OutType: proto.Clone(cached.OutType),
		}, nil
	}

	m, err := Find(f.ds, svcAndMethod)
	if err != nil {
		slog.Warn("ReflectionPBFinder.Get,svcAndMethod:%v, error:%v", svcAndMethod, err)
		return nil, err
	}

	f.innerCache.Set(svcAndMethod, m, cache.NoExpiration)
	return m, nil
}

func (f *ReflectionPBFinder) GetDescriptorSource() grpcurl.DescriptorSource {
	return f.ds
}

func getDataType(dataType *desc.MessageDescriptor) (protoreflect.MessageDescriptor, error) {
	// get FileDescriptor
	strSet := util.NewStringSet()
	fdSet := &descriptorpb.FileDescriptorSet{}
	constructFileDescriptorSet(strSet, fdSet, dataType.GetFile())
	prFiles, err := protodesc.NewFiles(fdSet)
	if err != nil {
		return nil, fmt.Errorf("dataType:%v,error:%w", dataType.GetFullyQualifiedName(), err)
	}
	pfd, err := prFiles.FindDescriptorByName(protoreflect.FullName(dataType.GetFullyQualifiedName()))
	if err != nil {
		return nil, fmt.Errorf("dataType:%v,error:%w", dataType.GetFullyQualifiedName(), err)
	}

	pfmd, ok := pfd.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("pfd.(protoreflect.MessageDescriptor),dataType:%v,type:%T",
			dataType.GetFullyQualifiedName(), pfd)
	}
	return pfmd, nil
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

func constructFileDescriptorSet(set *util.StringSet, fdSet *descriptorpb.FileDescriptorSet, fd *desc.FileDescriptor) {
	if !set.Has(fd.GetName()) {
		fdSet.File = append(fdSet.File, fd.AsFileDescriptorProto())
		set.Add(fd.GetName())
	}
	for _, dependentItem := range fd.GetDependencies() {
		constructFileDescriptorSet(set, fdSet, dependentItem)
	}
}

func Find(ds grpcurl.DescriptorSource, svcAndMethod string) (*MethodInputOutput, error) {
	slog.Debug("FilePBFinder, svcAndMethod:%v", svcAndMethod)
	svc, method := parseSymbol(svcAndMethod)
	slog.Info("parseSymbol, svc:%v, method:%v", svc, method)
	dsc, err := ds.FindSymbol(svc)
	if err != nil {
		return nil, fmt.Errorf("descSource.FindSymbol,service:%v,method:%v,error:%w", svc, method, err)
	}
	sd, ok := dsc.(*desc.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("change to *desc.ServiceDescriptor,service:%v, method:%v, type:%T",
			svc, method, dsc)
	}
	mtd := sd.FindMethodByName(method)
	inType, err := getDataType(mtd.GetInputType())
	if err != nil {
		slog.Error("Find, svc:%v, method:%v, error:%v", svc, method, err)
		return nil, err
	}
	outType, err := getDataType(mtd.GetOutputType())
	if err != nil {
		slog.Error("Find, svc:%v, method:%v, error:%v", svc, method, err)
		return nil, err
	}

	var result MethodInputOutput
	result.InType = dynamicpb.NewMessage(inType)
	result.OutType = dynamicpb.NewMessage(outType)
	return &result, nil
}
