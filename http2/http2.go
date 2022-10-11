package http2

import (
	"bytes"
	"encoding/binary"
	"fmt"
	slog "github.com/vearne/simplelog"
	"golang.org/x/net/http2/hpack"
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
	DirectConn          DirectConn
	MaxDynamicTableSize uint32
	MaxHeaderStringLen  uint32
	HeaderDecoder       *hpack.Decoder
	Streams             [StreamArraySize]*Stream
}

func NewHttp2Conn(conn DirectConn, maxDynamicTableSize uint32) *Http2Conn {
	var hc Http2Conn
	hc.DirectConn = conn
	hc.MaxDynamicTableSize = maxDynamicTableSize

	slog.Info("create Http2Conn, MaxDynamicTableSize:%v", maxDynamicTableSize)
	hc.HeaderDecoder = hpack.NewDecoder(maxDynamicTableSize, nil)
	for i := 0; i < StreamArraySize; i++ {
		hc.Streams[i] = NewStream()
	}
	return &hc
}

type Stream struct {
	StreamID  uint32
	EndHeader bool
	EndStream bool
	Headers   map[string]string `json:"headers"`
	Method    string            `json:"method"`
	// json序列化后的结果
	Request []byte `json:"request"`
}

func NewStream() *Stream {
	var s Stream
	s.Headers = make(map[string]string)
	s.EndStream = false
	s.EndHeader = false
	return &s
}

func (s *Stream) Reset() {
	s.StreamID = 0
	s.EndStream = false
	s.EndHeader = false
	s.Headers = make(map[string]string)
}

// 帧头部
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
	fb.Payload = b[HeaderSize : HeaderSize+fb.Length]
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
	// 参数都是可选
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
	fh.EndStream = f.Flags&0x1 != 0
	fh.Padded = f.Flags&0x8 != 0

	reader := bytes.NewReader(f.Payload)
	err = binary.Read(reader, binary.BigEndian, &fh.PadLength)
	if err != nil {
		return nil, err
	}
	fh.Data = f.Payload[1 : len(fh.Data)-int(fh.PadLength)]
	return &fh, nil
}

func ParseFrameContinuation(f *FrameBase) (*FrameContinuation, error) {
	var fc FrameContinuation
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
