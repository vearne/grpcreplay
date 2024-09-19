package http2

import (
	"context"
	"fmt"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/vearne/grpcreplay/protocol"
	slog "github.com/vearne/simplelog"
	"google.golang.org/grpc"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"strings"
	"sync"
)

type Processor struct {
	ConnRepository map[DirectConn]*Http2Conn
	InputChan      chan *NetPkg
	OutputChan     chan *protocol.Message
	Finder         *PBMessageFinder
}

func NewProcessor(input chan *NetPkg, svcAddr string) *Processor {
	var p Processor
	p.ConnRepository = make(map[DirectConn]*Http2Conn, 100)
	p.InputChan = input
	p.OutputChan = make(chan *protocol.Message, 100)
	p.Finder = NewPBMessageFinder(svcAddr)
	slog.Info("create new Processer")
	return &p
}

func (p *Processor) ProcessTCPPkg() {
	for pkg := range p.InputChan {
		dc := pkg.DirectConn()
		payload := pkg.TCP.Payload
		slog.Debug("Connection:%v, seq:%v, length:%v", dc.String(), pkg.TCP.Seq, len(payload))

		if _, ok := p.ConnRepository[dc]; !ok {
			p.ConnRepository[dc] = NewHttp2Conn(dc, http2initialHeaderTableSize, p)
		}
		hc := p.ConnRepository[dc]

		// SYN/ACK/FIN
		if len(payload) <= 0 {
			if pkg.TCP.FIN {
				slog.Info("got Fin package, close connection:%v", dc.String())
				hc.TCPBuffer.Close()
				delete(p.ConnRepository, dc)
			} else {
				hc.TCPBuffer.expectedSeq = int64(pkg.TCP.Seq) + int64(len(pkg.TCP.Payload))
				hc.TCPBuffer.leftPointer = hc.TCPBuffer.expectedSeq
			}
			continue
		}

		// connection preface
		if IsConnPreface(payload) {
			hc.TCPBuffer.expectedSeq = int64(pkg.TCP.Seq) + int64(len(pkg.TCP.Payload))
			hc.TCPBuffer.leftPointer = hc.TCPBuffer.expectedSeq
			continue
		}

		slog.Debug("[AddTCP]Connection:%v, seq:%v, length:%v", dc.String(), pkg.TCP.Seq, len(payload))
		hc.TCPBuffer.AddTCP(pkg.TCP)
	}
}

func IsConnPreface(payload []byte) bool {
	if len(payload) >= ConnectionPrefaceSize {
		b := payload[0:ConnectionPrefaceSize]
		if string(b) == PrefaceEarly || string(b) == PrefaceSTD {
			return true
		} else {
			return false
		}
	}
	return false
}

func getCodecType(headers map[string]string) int {
	contentType, ok := headers["content-type"]
	if !ok || contentType == "application/grpc" {
		return CodecProtobuf
	} else {
		return CodecOther
	}
}

type PBMessageFinder struct {
	cacheMu   sync.Mutex
	symbolMsg map[string]proto.Message
	// server address
	addr string
}

func NewPBMessageFinder(addr string) *PBMessageFinder {
	var f PBMessageFinder
	// svcAndMethod -> proto.Message
	f.symbolMsg = make(map[string]proto.Message)
	f.addr = addr
	return &f
}

func (f *PBMessageFinder) HandleRequestToJson(svcAndMethod string, data []byte) ([]byte, error) {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()

	pbMsg, err := f.FindMethodInputWithCache(svcAndMethod)
	if err != nil {
		slog.Error("finder.FindMethodInputWithCache, error:%v", err)
		return nil, err
	}
	err = proto.Unmarshal(data, pbMsg)
	if err != nil {
		slog.Error("method:%v, proto.Unmarshal:%v", svcAndMethod, err)
		return nil, err
	}

	result, err := protojson.Marshal(pbMsg)
	if err != nil {
		slog.Error("method:%v, json.Marshal:%v", svcAndMethod, err)
		return nil, err
	}
	return result, nil
}

func (f *PBMessageFinder) FindMethodInputWithCache(svcAndMethod string) (proto.Message, error) {
	slog.Debug("FindMethodInputWithCache, svcAndMethod:%v", svcAndMethod)

	m, ok := f.symbolMsg[svcAndMethod]
	if ok {
		slog.Debug("FindMethodInputWithCache,svcAndMethod:%v, hit cache", svcAndMethod)
		return m, nil
	}

	msg, err := f.FindMethodInput(svcAndMethod)
	if err != nil {
		return nil, err
	}

	f.symbolMsg[svcAndMethod] = msg
	return msg, nil
}

func (f *PBMessageFinder) FindMethodInput(svcAndMethod string) (proto.Message, error) {
	slog.Debug("FindMethodInput, svcAndMethod:%v", svcAndMethod)

	var cc *grpc.ClientConn
	network := "tcp"
	ctx := context.Background()
	cc, err := grpcurl.BlockingDial(ctx, network, f.addr, nil)
	if err != nil {
		slog.Fatal("PBMessageFinder.FindMethodInput,addr:%v,error:%v,enable grpc reflection service?",
			f.addr, err)
	}
	refClient := grpcreflect.NewClientV1Alpha(ctx, reflectpb.NewServerReflectionClient(cc))
	descSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)
	svc, method := parseSymbol(svcAndMethod)
	slog.Info("parseSymbol, svc:%v, method:%v", svc, method)
	dsc, err := descSource.FindSymbol(svc)
	if err != nil {
		return nil, fmt.Errorf("descSource.FindSymbol,service:%v,method:%v,error:%w", svc, method, err)
	}
	sd, ok := dsc.(*desc.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("change to *desc.ServiceDescriptor,service:%v,method:%v, type:%T",
			svc, method, dsc)
	}
	mtd := sd.FindMethodByName(method)
	inputType := mtd.GetInputType()
	// get FileDescriptor
	strSet := NewStringSet()
	fdSet := &descriptorpb.FileDescriptorSet{}
	ConstructFileDescriptorSet(strSet, fdSet, inputType.GetFile())
	prFiles, err := protodesc.NewFiles(fdSet)
	if err != nil {
		return nil, fmt.Errorf("protodesc.NewFiles,service:%v,method:%v,error:%w", svc, method, err)
	}
	pfd, err := prFiles.FindDescriptorByName(protoreflect.FullName(inputType.GetFullyQualifiedName()))
	if err != nil {
		return nil, fmt.Errorf("prFiles.FindDescriptorByName,service:%v,method:%v,error:%w",
			svc, method, err)
	}

	pfmd, ok := pfd.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("pfd.(protoreflect.MessageDescriptor),service:%v,method:%v,type:%T",
			svc, method, pfd)
	}
	return dynamicpb.NewMessage(pfmd), nil
}

func ConstructFileDescriptorSet(set *StringSet, fdSet *descriptorpb.FileDescriptorSet, fd *desc.FileDescriptor) {
	if !set.Has(fd.GetName()) {
		fdSet.File = append(fdSet.File, fd.AsFileDescriptorProto())
		set.Add(fd.GetName())
	}
	for _, dependentItem := range fd.GetDependencies() {
		ConstructFileDescriptorSet(set, fdSet, dependentItem)
	}
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
