package http2

import (
	"bytes"
	"compress/gzip"
	"context"
	"github.com/fullstorydev/grpcurl"
	"google.golang.org/protobuf/types/descriptorpb"
	"io"
	//"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	var f *FrameBase
	var err error
	for pkg := range p.InputChan {
		payload := pkg.TCP.Payload

		// connection preface
		if IsConnPreface(payload) {
			continue
		}

		for len(payload) >= HeaderSize {
			f, err = ParseFrameBase(payload)
			if err != nil {
				slog.Error("ProcessTCPPkg error:%v", err)
				continue
			}

			dc := pkg.DirectConn()
			f.DirectConn = &dc
			slog.Debug("Connection:%v, seq:%v, FrameType:%v, length:%v, streamID:%v",
				f.DirectConn, pkg.TCP.Seq, GetFrameType(f.Type), f.Length, f.StreamID)

			var ok bool
			if _, ok = p.ConnRepository[dc]; !ok {
				p.ConnRepository[dc] = NewHttp2Conn(dc, http2initialHeaderTableSize)
			}

			// Separate processing according to frame type
			p.ProcessFrame(f)

			if len(payload) >= int(HeaderSize+f.Length) {
				payload = payload[HeaderSize+f.Length:]
			} else {
				slog.Error("get TCP pkg:%v, seq:%v, tcp flags:%v",
					&dc, pkg.TCP.Seq, pkg.TCPFlags())
				payload = []byte{}
			}
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

func (p *Processor) ProcessFrame(f *FrameBase) {
	switch f.Type {
	case FrameTypeData:
		// parse data
		p.processFrameData(f)
	case FrameTypeHeader:
		// parse header
		p.processFrameHeader(f)
	case FrameTypeContinuation:
		p.processFrameContinuation(f)
	case FrameTypeRSTStream:
		// close stream
		p.processFrameRSTStream(f)
	case FrameTypeGoAway:
		// close connection
		p.processFrameGoAway(f)
	case FrameTypeSetting:
		p.processFrameSetting(f)
	default:
		// ignore the frame
		slog.Debug("ignore Frame:%v", GetFrameType(f.Type))
	}
}

func (p *Processor) processFrameData(f *FrameBase) {
	fd, err := ParseFrameData(f)
	if err != nil {
		slog.Error("ParseFrameData:%v", err)
		return
	}

	hc := p.ConnRepository[*f.DirectConn]
	// 设置stream的状态
	index := f.StreamID % StreamArraySize
	stream := hc.Streams[index]
	var gzipReader *gzip.Reader

	// 把protobuf转换为JSON字符串
	if len(fd.Data) > 0 && !strings.Contains(stream.Method, "grpc.reflection") {
		msg, _ := fd.ParseGRPCMessage()
		// 开启压缩了
		if msg.PayloadFormat == compressionMade {
			// only support gzip
			gzipReader, err = gzip.NewReader(bytes.NewReader(msg.EncodedMessage))
			if err != nil {
				slog.Error("processFrameData, gunzip error:%v", err)
				return
			}
			msg.EncodedMessage, err = io.ReadAll(gzipReader)
			if err != nil {
				slog.Error("processFrameData, gunzip error:%v", err)
				return
			}
		}

		codecType := getCodecType(stream.Headers)
		if codecType == CodecProtobuf {
			// 注意:暂时只处理 编码方式为Protobuf的情况
			pbMsg := p.Finder.FindMethodInput(stream.Method)

			err = proto.Unmarshal(msg.EncodedMessage, pbMsg)
			if err != nil {
				slog.Error("method:%v, proto.Unmarshal:%v", stream.Method, err)
			}

			stream.Request, err = protojson.Marshal(pbMsg)
			if err != nil {
				slog.Error("method:%v, json.Marshal:%v", stream.Method, err)
			}
		} else {
			stream.Request = msg.EncodedMessage
		}
	}

	if fd.EndStream {
		p.OutputChan <- stream.toMsg()
		stream.Reset()
	}
}

func getCodecType(headers map[string]string) int {
	contentType, ok := headers["content-type"]
	if !ok || contentType == "application/grpc" {
		return CodecProtobuf
	} else {
		return CodecOther
	}
}

func (p *Processor) processFrameHeader(f *FrameBase) {
	fh, err := ParseFrameHeader(f)
	if err != nil {
		slog.Error("ProcessFrameHeader:%v", err)
		return
	}

	hc := p.ConnRepository[*f.DirectConn]
	// 设置stream的状态
	index := f.StreamID % StreamArraySize
	stream := hc.Streams[index]
	stream.StreamID = f.StreamID
	stream.EndStream = fh.EndStream
	stream.EndHeader = fh.EndHeader
	slog.Debug("Connection:%v, stream:%v, EndHeader:%v, EndStream:%v, MaxDynamicTableSize:%v",
		hc.DirectConn.String(), stream.StreamID, stream.EndHeader, stream.EndStream, hc.MaxDynamicTableSize)

	hdec := hc.HeaderDecoder
	hdec.SetMaxStringLength(int(hc.MaxHeaderStringLen))
	fields, err := hdec.DecodeFull(fh.HeaderBlockFragment)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	for _, field := range fields {
		stream.Headers[field.Name] = field.Value
		if field.Name == PseudoHeaderPath {
			stream.Method = field.Value
		}
		slog.Debug(field.String())
	}

	if fh.EndStream {
		p.OutputChan <- stream.toMsg()
		stream.Reset()
	}
}

func (p *Processor) processFrameSetting(f *FrameBase) {
	fs, err := ParseFrameSetting(f)
	if err != nil {
		slog.Error("ParseFrameSetting:%v", err)
		return
	}
	if fs.Ack {
		return
	}

	hc := p.ConnRepository[*f.DirectConn]
	for _, item := range fs.settings {
		if item.ID == http2SettingHeaderTableSize {
			slog.Warn("adjust http2SettingHeaderTableSize:%v", item.Val)
			hc.MaxDynamicTableSize = item.Val
			hc.HeaderDecoder.SetMaxDynamicTableSize(item.Val)
		}
	}
}

func (p *Processor) processFrameContinuation(f *FrameBase) {
	fc, err := ParseFrameContinuation(f)
	if err != nil {
		slog.Error("ParseFrameContinuation:%v", err)
		return
	}

	hc, ok := p.ConnRepository[*f.DirectConn]
	if !ok {
		slog.Error("connection[%v] doesn't exist", f.DirectConn.String())
		return
	}
	// 设置stream的状态
	index := fc.fb.StreamID % StreamArraySize
	stream := hc.Streams[index]
	stream.StreamID = f.StreamID
	stream.EndHeader = fc.EndHeader
	slog.Debug("Connection:%v, stream:%v, EndHeader:%v, EndStream:%v",
		hc.DirectConn.String(), stream.StreamID, stream.EndHeader, stream.EndStream)

	hdec := hc.HeaderDecoder
	hdec.SetMaxStringLength(int(hc.MaxHeaderStringLen))
	fields, err := hdec.DecodeFull(fc.HeaderBlockFragment)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	for _, field := range fields {
		stream.Headers[field.Name] = field.Value
		slog.Debug(field.String())
	}
}

func (p *Processor) processFrameGoAway(f *FrameBase) {
	_, ok := p.ConnRepository[*f.DirectConn]
	if !ok {
		slog.Error("connection[%v] doesn't exist", f.DirectConn.String())
		return
	}
	// remove http2Conn
	delete(p.ConnRepository, *f.DirectConn)
}

func (p *Processor) processFrameRSTStream(f *FrameBase) {
	hc, ok := p.ConnRepository[*f.DirectConn]
	if !ok {
		slog.Error("connection[%v] doesn't exist", f.DirectConn.String())
		return
	}
	// 设置stream的状态
	index := f.StreamID % StreamArraySize
	stream := hc.Streams[index]
	stream.Reset()
}

type PBMessageFinder struct {
	cacheMu   sync.RWMutex
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

func (f *PBMessageFinder) FindMethodInputWithCache(svcAndMethod string) proto.Message {
	slog.Debug("FindMethodInputWithCache, svcAndMethod:%v", svcAndMethod)

	f.cacheMu.RLock()
	m, ok := f.symbolMsg[svcAndMethod]
	f.cacheMu.RUnlock()
	if ok {
		return m
	}

	msg := f.FindMethodInput(svcAndMethod)

	f.cacheMu.Lock()
	f.symbolMsg[svcAndMethod] = msg
	f.cacheMu.Unlock()
	return msg
}

func (f *PBMessageFinder) FindMethodInput(svcAndMethod string) proto.Message {
	slog.Debug("FindMethodInput, svcAndMethod:%v", svcAndMethod)

	// can't find in cache
	var cc *grpc.ClientConn
	network := "tcp"
	ctx := context.Background()
	cc, err := grpcurl.BlockingDial(ctx, network, f.addr, nil)
	if err != nil {
		slog.Fatal("PBMessageFinder.FindMethodInput, addr:%v, error:%v,enable grpc reflection service？",
			f.addr, err)
	}
	refClient := grpcreflect.NewClientV1Alpha(ctx, reflectpb.NewServerReflectionClient(cc))
	descSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)
	svc, method := parseSymbol(svcAndMethod)
	slog.Debug("parseSymbol, svc:%v, method:%v", svc, method)
	dsc, err := descSource.FindSymbol(svc)
	if err != nil {
		slog.Fatal("descSource.FindSymbol, service:%v, error:%v", svc, err)
	}
	sd, ok := dsc.(*desc.ServiceDescriptor)
	if !ok {
		slog.Fatal("FindMethodInput, error:%v", err)
	}
	mtd := sd.FindMethodByName(method)
	inputType := mtd.GetInputType()
	// get FileDescriptor
	fileDesc := inputType.GetFile()
	files := &descriptorpb.FileDescriptorSet{}
	files.File = append(files.File, fileDesc.AsFileDescriptorProto())
	for _, dependentItem := range fileDesc.GetDependencies() {
		files.File = append(files.File, dependentItem.AsFileDescriptorProto())
	}

	prFiles, err := protodesc.NewFiles(files)
	if err != nil {
		slog.Fatal("protodesc.NewFiles, svcAndMethod:%v, error:%v", svcAndMethod, err)
	}
	pfd, err := prFiles.FindDescriptorByName(protoreflect.FullName(inputType.GetFullyQualifiedName()))
	if err != nil {
		slog.Fatal("prFiles.FindDescriptorByName, svcAndMethod:%v, error:%v", svcAndMethod, err)
	}

	pfmd, ok := pfd.(protoreflect.MessageDescriptor)
	if !ok {
		slog.Fatal("pfd.(protoreflect.MessageDescriptor), svcAndMethod:%v, type:%T", svcAndMethod, pfd)
	}
	return dynamicpb.NewMessage(pfmd)

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
