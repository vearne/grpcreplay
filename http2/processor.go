package http2

import (
	"github.com/vearne/grpcreplay/protocol"
	slog "github.com/vearne/simplelog"
	"math"
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
		return
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
