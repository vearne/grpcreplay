package http2

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/vearne/grpcreplay/protocol"
	slog "github.com/vearne/simplelog"
	"golang.org/x/net/http2/hpack"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	PseudoHeaderPath = ":path"
)

const (
	// http://http2.github.io/http2-spec/#SettingValues
	http2initialHeaderTableSize = 4096
	ReadBufferSize              = 100 * 1024
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
	DirectConn          DirectConn
	MaxDynamicTableSize uint32
	MaxHeaderStringLen  uint32
	HeaderDecoder       *hpack.Decoder
	Streams             [StreamArraySize]*Stream
	TCPBuffer           *TCPBuffer
	Reader              *bufio.Reader
	Processor           *Processor
}

func NewHttp2Conn(conn DirectConn, maxDynamicTableSize uint32, p *Processor) *Http2Conn {
	var hc Http2Conn
	hc.DirectConn = conn
	hc.MaxDynamicTableSize = maxDynamicTableSize

	slog.Info("create Http2Conn, MaxDynamicTableSize:%v", maxDynamicTableSize)
	hc.HeaderDecoder = hpack.NewDecoder(maxDynamicTableSize, nil)
	for i := 0; i < StreamArraySize; i++ {
		hc.Streams[i] = NewStream()
	}
	hc.TCPBuffer = NewTCPBuffer()
	hc.Reader = bufio.NewReaderSize(hc.TCPBuffer, ReadBufferSize)
	hc.Processor = p

	go hc.deal()
	return &hc
}

func (hc *Http2Conn) deal() {
	slog.Debug("[start]Http2Conn.deal, Connection:%v", hc.DirectConn.String())

	var err error
	var fb *FrameBase
	for {
		slog.Debug("Http2Conn.deal, Connection:%v", hc.DirectConn.String())
		buf := make([]byte, HeaderSize)
		_, err = io.ReadFull(hc.Reader, buf)
		if err != nil {
			slog.Warn("Http2Conn.deal, ReadFull:%v", err)
			break
		}

		slog.Debug("Http2Conn.deal, ParseFrameBase, Connection:%v", hc.DirectConn.String())
		fb, err = ParseFrameBase(buf)
		if err != nil {
			slog.Error("ProcessTCPPkg error:%v", err)
			break
		}
		slog.Debug("Connection:%v,  FrameType:%v,  streamID:%v, len(payload):%v",
			hc.DirectConn.String(), GetFrameType(fb.Type), fb.StreamID, fb.Length)

		// Separate processing according to frame type
		buf = make([]byte, fb.Length)
		if fb.Length > 0 {
			_, err = io.ReadFull(hc.Reader, buf)
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
	fd, err := ParseFrameData(f)
	if err != nil {
		slog.Error("ParseFrameData:%v", err)
		return
	}

	slog.Debug("processFrameData, Padded:%v, PadLength:%v, EndStream:%v, len(fd.Data):%v",
		fd.Padded, fd.PadLength, fd.EndStream, len(fd.Data))

	// Set the state of the stream
	index := f.StreamID % StreamArraySize
	stream := hc.Streams[index]
	var gzipReader *gzip.Reader

	// Convert protobuf to JSON string
	if len(fd.Data) > 0 && !strings.Contains(stream.Method, "grpc.reflection") {
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
		_, err = stream.DataBuf.Write(msg.EncodedMessage)
		if err != nil {
			slog.Error("processFrameData, gunzip error:%v", err)
		}
	}

	if fd.EndStream {
		pMsg, pErr := stream.toMsg(hc.Processor.Finder)
		if pErr == nil {
			hc.Processor.OutputChan <- pMsg
		}
		stream.Reset()
	}
}

func (hc *Http2Conn) processFrameHeader(f *FrameBase) {
	fh, err := ParseFrameHeader(f)
	if err != nil {
		slog.Error("ProcessFrameHeader:%v", err)
		return
	}

	// Set the state of the stream
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
		pMsg, pErr := stream.toMsg(hc.Processor.Finder)
		if pErr == nil {
			hc.Processor.OutputChan <- pMsg
		}
		stream.Reset()
	}
}

func (hc *Http2Conn) processFrameContinuation(f *FrameBase) {
	fc, err := ParseFrameContinuation(f)
	if err != nil {
		slog.Error("ParseFrameContinuation:%v", err)
		return
	}

	// Set the state of the stream
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
			hc.MaxDynamicTableSize = item.Val
			hc.HeaderDecoder.SetMaxDynamicTableSize(item.Val)
		}
	}
}

type Stream struct {
	StreamID  uint32
	EndHeader bool
	EndStream bool
	Headers   map[string]string `json:"headers"`
	Method    string            `json:"method"`

	// The result after json serialization
	Request []byte        `json:"request"`
	DataBuf *bytes.Buffer `json:"-"`
}

func NewStream() *Stream {
	var s Stream
	s.Headers = make(map[string]string)
	s.EndStream = false
	s.EndHeader = false
	s.DataBuf = bytes.NewBuffer([]byte{})
	return &s
}

func (s *Stream) toMsg(finder *PBMessageFinder) (*protocol.Message, error) {
	var msg protocol.Message
	var err error
	id := uuid.Must(uuid.NewUUID())
	msg.Meta.Version = 1
	msg.Meta.UUID = id.String()
	msg.Meta.Timestamp = time.Now().UnixNano()

	msg.Data.Headers = s.Headers
	msg.Data.Method = s.Method

	codecType := getCodecType(s.Headers)
	if codecType == CodecProtobuf {
		s.Method = strings.TrimSpace(s.Method)
		if len(s.Method) <= 0 {
			slog.Error("method is empty, this is illegal")
			return nil, errors.New("method is empty")
		} else if !strings.Contains(s.Method, "grpc.reflection") {
			// Note: Temporarily only handle the case where the encoding method is Protobuf
			s.Request, err = finder.HandleRequestToJson(s.Method, s.DataBuf.Bytes())
			if err != nil {
				slog.Error("method:%v, HandleRequestToJson:%v", s.Method, err)
				return nil, err
			}
		}
	} else {
		s.Request = s.DataBuf.Bytes()
	}

	msg.Data.Request = string(s.Request)
	return &msg, nil
}

func (s *Stream) Reset() {
	s.StreamID = 0
	s.EndStream = false
	s.EndHeader = false
	s.Headers = make(map[string]string)
	s.Request = make([]byte, 0)
	s.DataBuf.Reset()
}

// Frame Header
type FrameBase struct {
	DirectConn *DirectConn
	StreamID   uint32
	Type       uint8
	Flags      uint8
	Length     uint32
	Payload    []byte
}

func ParseFrameBase(b []byte) (*FrameBase, error) {
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
