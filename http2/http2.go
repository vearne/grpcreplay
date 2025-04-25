package http2

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/vearne/grpcreplay/protocol"
	slog "github.com/vearne/simplelog"
	"golang.org/x/net/http2/hpack"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"io"
	"strings"
	"time"
)

const (
	PseudoHeaderPath = ":path"
)

const (
	// http://http2.github.io/http2-spec/#SettingValues
	http2initialHeaderTableSize = 4096
)

const (
	PrefaceEarly = "FOO * HTTP/2.0\r\n\r\nBA\r\n\r\n"
	PrefaceSTD   = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
)

const (
	FrameTypeData         = 0x0
	FrameTypeHeader       = 0x1
	FrameTypePriority     = 0x2
	FrameTypeRSTStream    = 0x3
	FrameTypeSetting      = 0x4
	FrameTypePushPromise  = 0x5
	FrameTypePing         = 0x6
	FrameTypeGoAway       = 0x7
	FrameTypeWindowUpdate = 0x8
	FrameTypeContinuation = 0x9
)

var FrameTypeStr map[uint8]string

func init() {
	FrameTypeStr = make(map[uint8]string)
	FrameTypeStr[FrameTypeData] = "FrameData"
	FrameTypeStr[FrameTypeHeader] = "FrameHeader"
	FrameTypeStr[FrameTypePriority] = "FramePriority"
	FrameTypeStr[FrameTypeRSTStream] = "FrameRSTStream"
	FrameTypeStr[FrameTypeSetting] = "FrameSetting"
	FrameTypeStr[FrameTypePushPromise] = "FramePushPromise"
	FrameTypeStr[FrameTypePing] = "FramePing"
	FrameTypeStr[FrameTypeGoAway] = "FrameGoAway"
	FrameTypeStr[FrameTypeWindowUpdate] = "FrameWindowUpdate"
	FrameTypeStr[FrameTypeContinuation] = "FrameContinuation"
}

func GetFrameType(t uint8) string {
	name, ok := FrameTypeStr[t]
	if ok {
		return name
	}
	return "UNKNOW"
}

// A SettingID is an HTTP/2 setting as defined in
// http://http2.github.io/http2-spec/#iana-settings
type http2SettingID uint16

const (
	http2SettingHeaderTableSize      http2SettingID = 0x1
	http2SettingEnablePush           http2SettingID = 0x2
	http2SettingMaxConcurrentStreams http2SettingID = 0x3
	http2SettingInitialWindowSize    http2SettingID = 0x4
	http2SettingMaxFrameSize         http2SettingID = 0x5
	http2SettingMaxHeaderListSize    http2SettingID = 0x6
)

var http2settingName = map[http2SettingID]string{
	http2SettingHeaderTableSize:      "HEADER_TABLE_SIZE",
	http2SettingEnablePush:           "ENABLE_PUSH",
	http2SettingMaxConcurrentStreams: "MAX_CONCURRENT_STREAMS",
	http2SettingInitialWindowSize:    "INITIAL_WINDOW_SIZE",
	http2SettingMaxFrameSize:         "MAX_FRAME_SIZE",
	http2SettingMaxHeaderListSize:    "MAX_HEADER_LIST_SIZE",
}

func (s http2SettingID) String() string {
	if v, ok := http2settingName[s]; ok {
		return v
	}
	return fmt.Sprintf("UNKNOWN_SETTING_%d", uint16(s))
}

// Setting is a setting parameter: which setting it is, and its value.
type http2Setting struct {
	// ID is which setting is being set.
	// See http://http2.github.io/http2-spec/#SettingValues
	ID http2SettingID

	// Val is the value.
	Val uint32
}

func (s http2Setting) String() string {
	return fmt.Sprintf("[%v = %d]", s.ID, s.Val)
}

// http2 connection context
type Http2Conn struct {
	DirectConn DirectConn

	Streams [StreamArraySize]*Stream
	// ###### for input ######
	Input *MessageParser
	// ###### for output ######
	Output *MessageParser

	Processor          *Processor
	RecordResponse     bool
	MaxHeaderStringLen uint32
}

type MessageParser struct {
	TCPBuffer           *TCPBuffer
	MaxDynamicTableSize uint32
	HeaderDecoder       *hpack.Decoder
}

func NewMessageParser(maxDynamicTableSize uint32) *MessageParser {
	var p MessageParser
	p.MaxDynamicTableSize = maxDynamicTableSize
	p.HeaderDecoder = hpack.NewDecoder(maxDynamicTableSize, nil)
	p.TCPBuffer = NewTCPBuffer()
	return &p
}

func NewHttp2Conn(conn DirectConn, maxDynamicTableSize uint32, p *Processor) *Http2Conn {
	var hc Http2Conn
	hc.DirectConn = conn
	hc.Input = NewMessageParser(maxDynamicTableSize)
	hc.Output = NewMessageParser(maxDynamicTableSize)

	slog.Info("create Http2Conn, MaxDynamicTableSize:%v", maxDynamicTableSize)
	for i := 0; i < StreamArraySize; i++ {
		hc.Streams[i] = NewStream(p.RecordResponse)
	}

	hc.Processor = p
	hc.RecordResponse = p.RecordResponse
	hc.MaxHeaderStringLen = 16 << 20

	go hc.DealInput()
	if hc.RecordResponse {
		go hc.DealOutput()
	}
	return &hc
}

func (hc *Http2Conn) DealOutput() {
	dc := hc.DirectConn.Reverse()
	slog.Debug("[start]Http2Conn.DealOutput, Connection:%v", dc.String())

	var err error
	var fb *FrameBase

	for {
		slog.Debug("Http2Conn.DealOutput, Connection:%v", dc.String())
		buf := make([]byte, HeaderSize)
		_, err = io.ReadFull(hc.Output.TCPBuffer, buf)
		if err != nil {
			slog.Warn("Http2Conn.DealOutput, ReadFull:%v", err)
			break
		}

		slog.Debug("Http2Conn.DealOutput, ParseFrameBase, Connection:%v", dc.String())
		fb, err = ParseFrameBase(buf, hc.DirectConn.Reverse(), false)
		if err != nil {
			slog.Error("ProcessTCPPkg error:%v", err)
			break
		}
		slog.Debug("Connection:%v,  FrameType:%v,  streamID:%v, len(payload):%v",
			dc.String(), GetFrameType(fb.Type), fb.StreamID, fb.Length)

		// Separate processing according to frame type
		buf = make([]byte, fb.Length)
		if fb.Length > 0 {
			_, err = io.ReadFull(hc.Output.TCPBuffer, buf)
			if err != nil {
				slog.Warn("Http2Conn.DealOutput, ReadFull:%v", err)
				break
			}
		}
		fb.Payload = buf
		hc.ProcessFrame(fb)
	}
}

func (hc *Http2Conn) DealInput() {
	slog.Debug("[start]Http2Conn.DealInput, Connection:%v", hc.DirectConn.String())

	var err error
	var fb *FrameBase
	for {
		slog.Debug("Http2Conn.DealInput, Connection:%v", hc.DirectConn.String())
		buf := make([]byte, HeaderSize)
		_, err = io.ReadFull(hc.Input.TCPBuffer, buf)
		if err != nil {
			slog.Warn("Http2Conn.DealInput, ReadFull:%v", err)
			break
		}

		slog.Debug("Http2Conn.DealInput, ParseFrameBase, Connection:%v", hc.DirectConn.String())
		fb, err = ParseFrameBase(buf, hc.DirectConn, true)
		if err != nil {
			slog.Error("ProcessTCPPkg error:%v", err)
			break
		}
		slog.Debug("Connection:%v,  FrameType:%v,  streamID:%v, len(payload):%v",
			hc.DirectConn.String(), GetFrameType(fb.Type), fb.StreamID, fb.Length)

		// Separate processing according to frame type
		buf = make([]byte, fb.Length)
		if fb.Length > 0 {
			_, err = io.ReadFull(hc.Input.TCPBuffer, buf)
			if err != nil {
				slog.Warn("Http2Conn.deal, ReadFull:%v", err)
				break
			}
		}
		fb.Payload = buf
		hc.ProcessFrame(fb)
	}

	slog.Debug("[end]Http2Conn.deal, Connection:%v", hc.DirectConn.String())
}

func (hc *Http2Conn) ProcessFrame(f *FrameBase) {
	slog.Debug("[ProcessFrame]: %v", GetFrameType(f.Type))
	switch f.Type {
	case FrameTypeData:
		// parse data
		hc.processFrameData(f)
	case FrameTypeHeader:
		// parse header
		hc.processFrameHeader(f)
	case FrameTypeContinuation:
		hc.processFrameContinuation(f)
	case FrameTypeRSTStream:
		// close stream
		hc.processFrameRSTStream(f)
	case FrameTypeGoAway:
		// close connection
		hc.processFrameGoAway(f)
	case FrameTypeSetting:
		hc.processFrameSetting(f)
	default:
		// ignore the frame
		slog.Debug("ignore Frame:%v", GetFrameType(f.Type))
	}
}

func (hc *Http2Conn) processFrameData(f *FrameBase) {
	// Set the state of the stream
	stream := hc.Streams[f.StreamID%StreamArraySize]
	stream.StreamID = f.StreamID

	//Is it input or output?
	if f.InputFlag {
		hc._processFrameData(f, stream.Request)
	} else {
		hc._processFrameData(f, stream.Response)
	}

	if (!hc.RecordResponse && stream.Request.EndStream) || (hc.RecordResponse && stream.Response.EndStream) {
		pMsg, pErr := stream.toMsg(hc.Processor.Finder)
		if pErr == nil {
			hc.Processor.OutputChan <- pMsg
		} else {
			slog.Warn("stream.toMsg, error:%v", pErr)
		}
		stream.Reset()
	}
}

func (hc *Http2Conn) _processFrameData(f *FrameBase, item *HTTPItem) {
	fd, err := ParseFrameData(f)
	if err != nil {
		slog.Error("ParseFrameData:%v", err)
		return
	}

	item.EndStream = fd.EndStream
	slog.Debug("processFrameData, Padded:%v, PadLength:%v, EndStream:%v, len(fd.Data):%v",
		fd.Padded, fd.PadLength, fd.EndStream, len(fd.Data))

	var gzipReader *gzip.Reader

	// Convert protobuf to JSON string
	if len(fd.Data) > 0 {
		msg, _ := fd.ParseGRPCMessage()
		// Compression is turned on
		if msg.PayloadFormat == compressionMade {
			slog.Debug("msg.PayloadFormat == compressionMade")
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

		slog.Debug("len(msg.EncodedMessage):%v", len(msg.EncodedMessage))
		_, err = item.DataBuf.Write(msg.EncodedMessage)
		if err != nil {
			slog.Error("processFrameData, gunzip error:%v", err)
		}
	}
}

func (hc *Http2Conn) processFrameHeader(f *FrameBase) {
	// Set the state of the stream
	index := f.StreamID % StreamArraySize
	stream := hc.Streams[index]
	stream.StreamID = f.StreamID

	//Is it input or output?
	if f.InputFlag {
		hc._processFrameHeader(f, hc.Input, stream.Request)
	} else {
		hc._processFrameHeader(f, hc.Output, stream.Response)
	}

	if (!hc.RecordResponse && stream.Request.EndStream) || (hc.RecordResponse && stream.Response.EndStream) {
		pMsg, pErr := stream.toMsg(hc.Processor.Finder)
		if pErr == nil {
			hc.Processor.OutputChan <- pMsg
		}
		stream.Reset()
	}
}

func (hc *Http2Conn) _processFrameHeader(f *FrameBase, parser *MessageParser,
	item *HTTPItem) {
	fh, err := ParseFrameHeader(f)
	if err != nil {
		slog.Error("ProcessFrameHeader:%v", err)
		return
	}

	item.EndHeader = fh.EndHeader
	item.EndStream = fh.EndStream

	slog.Debug("Connection:%v, stream:%v, EndHeader:%v, EndStream:%v",
		f.DirectConn.String(), f.StreamID, fh.EndHeader, fh.EndStream)

	hdec := parser.HeaderDecoder
	//hdec.SetMaxStringLength(int(hc.MaxHeaderStringLen))
	fields, err := hdec.DecodeFull(fh.HeaderBlockFragment)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	for _, field := range fields {
		item.Headers[field.Name] = field.Value
		if field.Name == PseudoHeaderPath {
			item.Method = field.Value
		}
		slog.Debug(field.String())
	}
}

func (hc *Http2Conn) processFrameContinuation(f *FrameBase) {
	// Set the state of the stream
	stream := hc.Streams[f.StreamID%StreamArraySize]
	stream.StreamID = f.StreamID

	//Is it input or output?
	if f.InputFlag {
		hc._processFrameContinuation(f, hc.Input, stream.Request)
	} else {
		hc._processFrameContinuation(f, hc.Output, stream.Response)
	}
}

func (hc *Http2Conn) _processFrameContinuation(f *FrameBase,
	parser *MessageParser, item *HTTPItem) {
	fc, err := ParseFrameContinuation(f)
	if err != nil {
		slog.Error("ParseFrameContinuation:%v", err)
		return
	}

	item.EndHeader = fc.EndHeader

	slog.Debug("Connection:%v, stream:%v, EndHeader:%v, EndStream:%v",
		hc.DirectConn.String(), f.StreamID, item.EndHeader, item.EndStream)

	hdec := parser.HeaderDecoder
	fields, err := hdec.DecodeFull(fc.HeaderBlockFragment)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	for _, field := range fields {
		item.Headers[field.Name] = field.Value
		slog.Debug(field.String())
	}
}

func (hc *Http2Conn) processFrameGoAway(f *FrameBase) {
	// remove http2Conn
	delete(hc.Processor.ConnRepository, hc.DirectConn)
}

func (hc *Http2Conn) processFrameRSTStream(f *FrameBase) {
	// Set the state of the stream
	index := f.StreamID % StreamArraySize
	stream := hc.Streams[index]
	stream.Reset()
}

func (hc *Http2Conn) processFrameSetting(f *FrameBase) {
	// Set the state of the stream
	stream := hc.Streams[f.StreamID%StreamArraySize]
	stream.StreamID = f.StreamID

	//Is it input or output?
	if f.InputFlag {
		hc._processFrameSetting(f, hc.Input)
	} else {
		hc._processFrameSetting(f, hc.Output)
	}
}

func (hc *Http2Conn) _processFrameSetting(f *FrameBase, parser *MessageParser) {
	fs, err := ParseFrameSetting(f)
	if err != nil {
		slog.Error("ParseFrameSetting:%v", err)
		return
	}
	if fs.Ack {
		return
	}

	for _, item := range fs.settings {
		if item.ID == http2SettingHeaderTableSize {
			slog.Warn("adjust http2SettingHeaderTableSize:%v", item.Val)
			parser.MaxDynamicTableSize = item.Val
			parser.HeaderDecoder.SetMaxDynamicTableSize(item.Val)
		}
	}
}

type Stream struct {
	StreamID       uint32
	RecordResponse bool
	Request        *HTTPItem
	Response       *HTTPItem
}

type HTTPItem struct {
	EndHeader bool
	EndStream bool

	Headers map[string]string `json:"headers"`
	Method  string            `json:"method"`

	DataBuf *bytes.Buffer `json:"-"`
}

func (item *HTTPItem) Reset() {
	item.EndStream = false
	item.EndHeader = false
	item.Headers = make(map[string]string)
	//item.Body = make([]byte, 0)
	item.DataBuf.Reset()
}

func NewHTTPItem() *HTTPItem {
	var item HTTPItem
	item.Headers = make(map[string]string)
	item.EndStream = false
	item.EndHeader = false
	item.DataBuf = bytes.NewBuffer([]byte{})
	return &item
}

func NewStream(recordResponse bool) *Stream {
	var s Stream
	s.RecordResponse = recordResponse
	s.Request = NewHTTPItem()
	if recordResponse {
		s.Response = NewHTTPItem()
	}
	return &s
}

func (s *Stream) toMsg(finder PBFinder) (*protocol.Message, error) {
	method := strings.TrimSpace(s.Request.Method)
	if len(method) <= 0 {
		slog.Error("method is empty, this is illegal")
		return nil, errors.New("method is empty")
	}
	if strings.Contains(method, "grpc.reflection") {
		return nil, errors.New("method is grpc.reflection")
	}

	var msg protocol.Message
	var err error
	var dataType *MethodInputOutput
	id := uuid.Must(uuid.NewUUID())
	msg.Meta.Version = 2
	msg.Meta.UUID = id.String()
	msg.Meta.Timestamp = time.Now().UnixNano()
	msg.Meta.ContainResponse = s.RecordResponse

	msg.Method = strings.TrimSpace(s.Request.Method)
	codecType := getCodecType(s.Request.Headers)
	// 1. ###### request ######
	msg.Request = &protocol.MsgItem{}
	msg.Request.Headers = s.Request.Headers

	if codecType == CodecProtobuf {
		// Note: Temporarily only handle the case where the encoding method is Protobuf
		dataType, err = finder.Get(msg.Method)
		if err != nil {
			slog.Error("finder.Get, method:%v, error:%v", method, err)
			return nil, err
		}
		msg.Request.Body, err = changeToJsonStr(dataType.InType, s.Request.DataBuf.Bytes())
		if err != nil {
			slog.Error("changeToJsonStr, method:%v, error:%v", method, err)
			return nil, err
		}
	} else {
		msg.Request.Body = s.Request.DataBuf.String()
	}
	// 2. ###### response ######
	if s.RecordResponse {
		msg.Response = &protocol.MsgItem{}
		msg.Response.Headers = s.Response.Headers

		if codecType == CodecProtobuf {
			msg.Response.Body, err = changeToJsonStr(dataType.OutType, s.Response.DataBuf.Bytes())
			if err != nil {
				slog.Error("changeToJsonStr, method:%v, error:%v", method, err)
				return nil, err
			}
		} else {
			msg.Response.Body = s.Response.DataBuf.String()
		}
	}

	return &msg, nil
}

func changeToJsonStr(pbMsg proto.Message, data []byte) (string, error) {
	err := proto.Unmarshal(data, pbMsg)
	if err != nil {
		return "", err
	}

	result, err := protojson.Marshal(pbMsg)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func (s *Stream) Reset() {
	s.StreamID = 0
	s.Request.Reset()
	if s.Response != nil {
		s.Response.Reset()
	}
}

// Frame Header
type FrameBase struct {
	DirectConn DirectConn
	// input or output ?
	InputFlag bool
	StreamID  uint32
	Type      uint8
	Flags     uint8
	Length    uint32
	Payload   []byte
}

func ParseFrameBase(b []byte, dc DirectConn, inputFlag bool) (*FrameBase, error) {
	reader := bytes.NewReader(b)
	var fb FrameBase
	var tmp uint8
	var err error
	// Length(24)
	for i := 0; i < LengthSize; i++ {
		err = binary.Read(reader, binary.BigEndian, &tmp)
		if err != nil {
			return nil, err
		}
		fb.Length = fb.Length*256 + uint32(tmp)
	}
	// Type(8)
	err = binary.Read(reader, binary.BigEndian, &fb.Type)
	if err != nil {
		return nil, err
	}
	// Flags(8)
	err = binary.Read(reader, binary.BigEndian, &fb.Flags)
	if err != nil {
		return nil, err
	}
	// Stream Identifier(31)
	err = binary.Read(reader, binary.BigEndian, &fb.StreamID)
	if err != nil {
		return nil, err
	}

	if fb.Length+HeaderSize <= uint32(len(b)) {
		fb.Payload = b[HeaderSize : HeaderSize+fb.Length]
	} else {
		fb.Payload = b[HeaderSize:]
	}
	// Mark the direction
	fb.DirectConn = dc
	fb.InputFlag = inputFlag
	return &fb, nil
}

func ParseFrameSetting(f *FrameBase) (*FrameSetting, error) {
	var fs FrameSetting
	fs.settings = make([]http2Setting, 0)

	var err error
	var identifier uint16
	var value uint32
	// basic info
	fs.fb = f

	fs.Ack = f.Flags&0x1 != 0
	reader := bytes.NewReader(f.Payload)
	// All parameters are optional
	for reader.Len() > 0 {
		err = binary.Read(reader, binary.BigEndian, &identifier)
		if err != nil {
			return nil, err
		}
		err = binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, err
		}
		switch http2SettingID(identifier) {
		case http2SettingHeaderTableSize:
			fs.settings = append(fs.settings, http2Setting{ID: http2SettingHeaderTableSize, Val: value})
		default:
			slog.Debug("ignore:%v", http2SettingID(identifier))
		}
	}
	return &fs, nil
}

func ParseFrameHeader(f *FrameBase) (*FrameHeader, error) {
	var fh FrameHeader
	// basic info
	fh.fb = f

	fh.EndStream = f.Flags&0x1 != 0
	fh.EndHeader = f.Flags&0x4 != 0
	fh.Padded = f.Flags&0x8 != 0
	fh.Priority = f.Flags&0x20 != 0

	// ----Frame Payload----
	start := 0
	// Pad Length(optional)
	if fh.Padded {
		start += 1

		reader := bytes.NewReader(f.Payload)
		//binary.BigEndian
		err := binary.Read(reader, binary.BigEndian, &fh.PadLength)
		if err != nil {
			return nil, err
		}
	}
	// E/Stream Dependency/Weight (optional)
	if fh.Priority {
		start += 5
	}

	// HeaderBlockFragment
	fh.HeaderBlockFragment = f.Payload[start : len(f.Payload)-int(fh.PadLength)]
	slog.Debug("Padded:%v, len(f.Payload):%v, PadLength:%v, len(HeaderBlockFragment):%v",
		fh.Padded, len(f.Payload), fh.PadLength, len(fh.HeaderBlockFragment))
	return &fh, nil
}

func ParseFrameData(f *FrameBase) (*FrameData, error) {
	var fh FrameData
	var err error

	// basic info
	fh.fb = f

	fh.EndStream = f.Flags&0x1 != 0
	fh.Padded = f.Flags&0x8 != 0

	start := 0
	// Pad Length(optional)
	if fh.Padded {
		start += 1

		reader := bytes.NewReader(f.Payload)
		//binary.BigEndian
		err = binary.Read(reader, binary.BigEndian, &fh.PadLength)
		if err != nil {
			return nil, err
		}
	}
	fh.Data = f.Payload[start : len(f.Payload)-int(fh.PadLength)]
	return &fh, nil
}

func ParseFrameContinuation(f *FrameBase) (*FrameContinuation, error) {
	var fc FrameContinuation
	// basic info
	fc.fb = f

	fc.EndHeader = f.Flags&0x4 != 0

	fc.HeaderBlockFragment = f.Payload
	return &fc, nil
}

type FrameSetting struct {
	fb  *FrameBase
	Ack bool
	// Frame Payload
	settings []http2Setting
}

type FrameData struct {
	fb        *FrameBase
	EndStream bool
	Padded    bool
	// Frame Payload
	PadLength uint8
	Data      []byte
}

func (fd *FrameData) ParseGRPCMessage() (*GRPCMessage, error) {
	var gm GRPCMessage
	gm.PayloadFormat = payloadFormat(fd.Data[0])
	gm.Length = binary.BigEndian.Uint32(fd.Data[1:])
	gm.EncodedMessage = fd.Data[5:]
	return &gm, nil
}

type GRPCMessage struct {
	// ------ complete gRPC Message------
	// https://github.com/grpc/grpc-go/blob/master/Documentation/encoding.md
	//	gRPC lets you use encoders other than Protobuf.
	//  gRPC is compatible with JSON, Thrift, Avro, Flatbuffers, Cap’n Proto, and even raw bytes!
	/*
				+--------------------+
				|  payloadFormat(8)  |
				+--------------------+------------------------------------------+
				|                          length(32)                           |
				+---------------------------------------------------------------+
				|                        encodedMessage(*)                  ... |
				+---------------------------------------------------------------+
			   payloadFormat: compressed or not?
		       encodedMessage: Protobuf,JSON,Thrift,etc.
	*/
	PayloadFormat  payloadFormat
	Length         uint32
	EncodedMessage []byte
}

type FrameHeader struct {
	fb        *FrameBase
	EndStream bool
	EndHeader bool
	Padded    bool
	// I don't care
	Priority bool
	// Frame Payload
	PadLength           uint8
	HeaderBlockFragment []byte
}

type FrameContinuation struct {
	fb                  *FrameBase
	EndHeader           bool
	HeaderBlockFragment []byte
}
