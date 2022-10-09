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

const (
	SETTINGS_HEADER_TABLE_SIZE      = 0x1
	SETTINGS_ENABLE_PUSH            = 0x2
	SETTINGS_MAX_CONCURRENT_STREAMS = 0x3
	SETTINGS_INITIAL_WINDOW_SIZE    = 0x4
	SETTINGS_MAX_FRAME_SIZE         = 0x5
	SETTINGS_MAX_HEADER_LIST_SIZE   = 0x6
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

func ProcessFrameBase(b []byte) (*FHeader, error) {
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

func ProcessFrameSetting(f *FHeader) (*FrameSetting, error) {
	var fs FrameSetting
	var err error
	var identifier uint16
	var value uint32
	fs.Ack = f.Flags&0x1 != 0
	reader := bytes.NewReader(f.Payload)
	reader.Size()
	for reader.Len() > 0 {
		err = binary.Read(reader, binary.BigEndian, &identifier)
		if err != nil {
			return nil, err
		}
		err = binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, err
		}
		switch identifier {
		case SETTINGS_HEADER_TABLE_SIZE:
			fs.HeaderTableSize = value
		case SETTINGS_ENABLE_PUSH:
			fs.EnablePush = value != 0
		case SETTINGS_MAX_CONCURRENT_STREAMS:
			fs.MaxConcurrentStreams = value
		case SETTINGS_INITIAL_WINDOW_SIZE:
			fs.InitialWindowSize = value
		case SETTINGS_MAX_FRAME_SIZE:
			fs.MaxFrameSize = value
		case SETTINGS_MAX_HEADER_LIST_SIZE:
			fs.MaxHeaderListSize = value
		}
	}
	return &fs, nil
}

func ProcessFrameData(f *FHeader) (*FrameData, error) {
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

type FrameSetting struct {
	fh  FHeader
	Ack bool
	// Frame Payload
	HeaderTableSize      uint32
	EnablePush           bool
	MaxConcurrentStreams uint32
	InitialWindowSize    uint32
	MaxFrameSize         uint32
	MaxHeaderListSize    uint32
}

type FrameData struct {
	fh        FHeader
	EndStream bool
	Padded    bool
	// Frame Payload
	PadLength uint8
	Data      []byte
}

type FrameHeader struct {
	fh        FHeader
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
	fh FHeader
}
