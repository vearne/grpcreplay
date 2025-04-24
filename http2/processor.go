package http2

import (
	"context"
	"fmt"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/vearne/grpcreplay/protocol"
	"github.com/vearne/grpcreplay/util"
	slog "github.com/vearne/simplelog"
	"google.golang.org/grpc"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"math"
	"sync"
)

type Processor struct {
	ConnRepository map[DirectConn]*Http2Conn
	InputChan      chan *NetPkg
	OutputChan     chan *protocol.Message
	Finder         PBFinder
	RecordResponse bool
}

func NewProcessor(input chan *NetPkg, svcAddr string, recordResponse bool) *Processor {
	var p Processor
	p.ConnRepository = make(map[DirectConn]*Http2Conn, 100)
	p.InputChan = input
	p.OutputChan = make(chan *protocol.Message, 100)
	p.Finder = NewReflectionPBFinder(svcAddr)
	p.RecordResponse = recordResponse
	slog.Info("create new Processor")
	return &p
}

func (p *Processor) ProcessIncomingTCPPkg(pkg *NetPkg) {
	dc := pkg.DirectConn()
	payload := pkg.TCP.Payload
	slog.Debug("Connection:%v, seq:%v, length:%v", dc.String(), pkg.TCP.Seq, len(payload))

	if _, ok := p.ConnRepository[dc]; !ok {
		p.ConnRepository[dc] = NewHttp2Conn(dc, http2initialHeaderTableSize, p)
	}
	hc := p.ConnRepository[dc]

	payloadSize := uint32(len(payload))

	// SYN/ACK/FIN
	if len(payload) <= 0 {
		if pkg.TCP.FIN {
			slog.Info("got Fin package, close connection:%v", dc.String())
			hc.Input.TCPBuffer.Close()
			delete(p.ConnRepository, dc)
		} else {
			hc.Input.TCPBuffer.expectedSeq = (pkg.TCP.Seq + payloadSize) % math.MaxUint32
		}
		return
	}

	// connection preface
	if IsConnPreface(payload) {
		hc.Input.TCPBuffer.expectedSeq = (pkg.TCP.Seq + payloadSize) % math.MaxUint32
		return
	}

	slog.Debug("[AddTCP]Connection:%v, seq:%v, length:%v", dc.String(), pkg.TCP.Seq, len(payload))
	hc.Input.TCPBuffer.AddTCP(pkg.TCP)
}

func (p *Processor) ProcessOutComingTCPPkg(pkg *NetPkg) {
	dc := pkg.DirectConn()
	payload := pkg.TCP.Payload
	slog.Debug("Connection:%v, seq:%v, length:%v", dc.String(), pkg.TCP.Seq, len(payload))

	rDirect := dc.Reverse()
	if _, ok := p.ConnRepository[rDirect]; !ok {
		p.ConnRepository[rDirect] = NewHttp2Conn(rDirect, http2initialHeaderTableSize, p)
	}
	hc := p.ConnRepository[rDirect]

	payloadSize := uint32(len(payload))

	// SYN/ACK/FIN
	if len(payload) <= 0 {
		if pkg.TCP.FIN {
			slog.Info("got Fin package, close connection:%v", rDirect.String())
			hc.Output.TCPBuffer.Close()
			delete(p.ConnRepository, rDirect)
		} else {
			hc.Output.TCPBuffer.expectedSeq = (pkg.TCP.Seq + payloadSize) % math.MaxUint32
		}
		return
	}

	slog.Debug("[AddTCP]Connection:%v, seq:%v, length:%v", dc.String(), pkg.TCP.Seq, len(payload))
	hc.Output.TCPBuffer.AddTCP(pkg.TCP)

}

func (p *Processor) ProcessTCPPkg() {
	// need to handle both inbound and outbound traffic
	for pkg := range p.InputChan {
		if pkg.Direction == DirIncoming {
			p.ProcessIncomingTCPPkg(pkg)
		} else if p.RecordResponse && pkg.Direction == DirOutcoming {
			p.ProcessOutComingTCPPkg(pkg)
		}
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
	strSet := util.NewStringSet()
	fdSet := &descriptorpb.FileDescriptorSet{}
	constructFileDescriptorSet(strSet, fdSet, inputType.GetFile())
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
