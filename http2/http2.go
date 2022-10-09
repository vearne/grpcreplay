package http2

import (
	"bytes"
	"encoding/binary"
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

type Http2Conn struct {
	// 静态表
	// 动态表
	Streams [StreamArraySize]*Stream
}

type Stream struct {
	StreamID  uint32
	EndHeader bool
	EndStream bool
	Headers   map[string][]string `json:"headers"`
	Method    string              `json:"method"`
	Request   string              `json:"request"`
}

// 帧头部
type FHeader struct {
	StreamID uint32
	Type     uint8
	Flags    uint8
	Length   uint32
	Payload  []byte
}

func processFrameBase(b []byte) (*FHeader, error) {
	reader := bytes.NewReader(b)
	var fh FHeader
	var tmp uint8
	var err error
	// Length(24)
	for i := 0; i < LengthSize; i++ {
		err = binary.Read(reader, binary.BigEndian, &tmp)
		if err != nil {
			return nil, err
		}
		fh.Length = fh.Length*256 + uint32(tmp)
	}
	// Type(8)
	err = binary.Read(reader, binary.BigEndian, &fh.Type)
	if err != nil {
		return nil, err
	}
	// Flags(8)
	err = binary.Read(reader, binary.BigEndian, &fh.Flags)
	if err != nil {
		return nil, err
	}
	// Stream Identifier(31)
	err = binary.Read(reader, binary.BigEndian, &fh.StreamID)
	if err != nil {
		return nil, err
	}
	fh.Payload = b[HeaderSize : HeaderSize+fh.Length]
	return &fh, nil
}

type FrameData struct {
	fh FHeader
}

type FrameHeader struct {
	fh FHeader
}

type FrameContinuation struct {
	fh FHeader
}
