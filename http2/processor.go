package http2

import (
	"github.com/vearne/grpcreplay/protocol"
	slog "github.com/vearne/simplelog"
)

type Processor struct {
	ConnRepository map[DirectConn]*Http2Conn
	InputChan      chan *NetPkg
	OutputChan     chan *protocol.Message
}

func NewProcessor(input chan *NetPkg) *Processor {
	var p Processor
	p.ConnRepository = make(map[DirectConn]*Http2Conn, 100)
	p.InputChan = input
	p.OutputChan = make(chan *protocol.Message, 100)
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
	return len(payload) == ConnectionPrefaceSize &&
		(string(payload) == PrefaceEarly || string(payload) == PrefaceSTD)
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
}

func (p *Processor) processFrameHeader(f *FrameBase) {
	fh, err := ParseFrameHeader(f)
	if err != nil {
		slog.Error("ProcessFrameHeader:%v", err)
		return
	}

	hc := p.ConnRepository[*f.DirectConn]
	// 设置stream的状态
	index := fh.fb.StreamID % StreamArraySize
	stream := hc.Streams[index]
	stream.StreamID = f.StreamID
	stream.EndStream = fh.EndStream
	stream.EndHeader = fh.EndHeader
	slog.Info("Connection:%v, stream:%v, EndHeader:%v, EndStream:%v, MaxDynamicTableSize:%v",
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
		slog.Debug(field.String())
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

	hc := p.ConnRepository[*f.DirectConn]
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

}

func (p *Processor) processFrameRSTStream(f *FrameBase) {

}
