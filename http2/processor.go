package http2

import (
	"github.com/vearne/grpcreplay/protocol"
	slog "github.com/vearne/simplelog"
)

type Processor struct {
	ProcessBuff map[DirectConn]*Http2Conn
	InputChan   chan *NetPkg
	OutputChan  chan *protocol.Message
}

func NewProcessor(input chan *NetPkg) *Processor {
	var p Processor
	p.ProcessBuff = make(map[DirectConn]*Http2Conn, 100)
	p.InputChan = input
	p.OutputChan = make(chan *protocol.Message, 100)
	slog.Info("create new Processer")
	return &p
}

func (p *Processor) ProcessTCPPkg() {
	var fh *FHeader
	var err error
	for pkg := range p.InputChan {
		payload := pkg.TCP.Payload

		// connection preface
		if IsConnPreface(payload) {
			continue
		}

		for len(payload) >= HeaderSize {
			fh, err = processFrameBase(payload)
			if err != nil {
				slog.Error("ProcessTCPPkg error:%v", err)
				continue
			}
			slog.Debug("FrameType:%v, length:%v, streamID:%v", GetFrameType(fh.Type),
				fh.Length, fh.StreamID)

			// Separate processing according to frame type
			p.ProcessFrame(fh)

			if len(payload) >= int(HeaderSize+fh.Length) {
				payload = payload[HeaderSize+fh.Length:]
			} else {
				dc := pkg.DirectConn()
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

func (p *Processor) ProcessFrame(f *FHeader) {
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
	default:
		// ignore the frame
		slog.Debug("ignore Frame:%v", GetFrameType(f.Type))
	}
}

func (p *Processor) processFrameData(f *FHeader) {

}

func (p *Processor) processFrameHeader(f *FHeader) {

}

func (p *Processor) processFrameContinuation(f *FHeader) {

}

func (p *Processor) processFrameGoAway(f *FHeader) {

}

func (p *Processor) processFrameRSTStream(f *FHeader) {

}
