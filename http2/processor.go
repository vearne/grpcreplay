package http2

import (
	fsm "github.com/smallnest/gofsm"
	"github.com/vearne/grpcreplay/protocol"
	slog "github.com/vearne/simplelog"
	"math"
)

type Processor struct {
	ConnStates      map[DirectConn]*TCPConnectionState
	ConnRepository  map[DirectConn]*Http2Conn
	InputChan       chan *NetPkg
	OutputChan      chan *protocol.Message
	Finder          PBFinder
	RecordResponse  bool
	TCPStateMachine *fsm.StateMachine
}

func NewProcessor(input chan *NetPkg, recordResponse bool, finder PBFinder) *Processor {
	var p Processor
	p.ConnStates = make(map[DirectConn]*TCPConnectionState, 100)
	p.ConnRepository = make(map[DirectConn]*Http2Conn, 100)
	p.InputChan = input
	p.OutputChan = make(chan *protocol.Message, 100)
	p.Finder = finder
	p.RecordResponse = recordResponse
	p.TCPStateMachine = InitTCPFSM(&TCPEventProcessor{})
	slog.Info("create new Processor")
	return &p
}

func (p *Processor) ProcessIncomingTCPPkg(pkg *NetPkg) {
	dc := pkg.DirectConn()
	payload := pkg.TCP.Payload

	if _, ok := p.ConnRepository[dc]; !ok {
		return
	}

	hc := p.ConnRepository[dc]
	payloadSize := uint32(len(payload))

	// connection preface
	if IsConnPreface(payload) {
		hc.Input.TCPBuffer.SetExpectedSeq((pkg.TCP.Seq + payloadSize) % math.MaxUint32)
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
	slog.Debug("[AddTCP]Connection:%v, seq:%v, length:%v", dc.String(), pkg.TCP.Seq, len(payload))
	hc.Output.TCPBuffer.AddTCP(pkg.TCP)
}

func (p *Processor) ProcessTCPPkg() {
	// need to handle both inbound and outbound traffic
	for pkg := range p.InputChan {
		payload := pkg.TCP.Payload
		dc := pkg.DirectConn()
		slog.Debug("Connection:%v, seq:%v, length:%v", dc.String(), pkg.TCP.Seq, len(payload))

		if pkg.Direction == DirOutcoming {
			dc = dc.Reverse()
		}
		ts, exist := p.ConnStates[dc]
		if !exist {
			p.ConnStates[dc] = NewTCPConnection(dc)
			ts = p.ConnStates[dc]
		}
		// try to handling connection status
		err := p.handleConnectionState(ts, pkg)
		if err != nil {
			slog.Warn("TCPStateMachine.Trigger, %v", err)
		}

		// data
		if ts.State == StateEstablished && len(payload) > 0 {
			if pkg.Direction == DirIncoming {
				p.ProcessIncomingTCPPkg(pkg)
			} else if p.RecordResponse && pkg.Direction == DirOutcoming {
				p.ProcessOutComingTCPPkg(pkg)
			}
		}

	}
}

func (p *Processor) handleConnectionState(ts *TCPConnectionState, pkg *NetPkg) error {
	if len(pkg.TCP.Payload) > 0 {
		return nil
	}
	if pkg.Direction == DirIncoming && pkg.TCP.SYN {
		return p.TCPStateMachine.Trigger(ts.State, EventReceiveSYN, ts, pkg, p)
	} else if pkg.Direction == DirOutcoming && pkg.TCP.SYN && pkg.TCP.ACK {
		return p.TCPStateMachine.Trigger(ts.State, EventSendSYNACK, ts, pkg, p)
	} else if pkg.Direction == DirIncoming && pkg.TCP.ACK {
		return p.TCPStateMachine.Trigger(ts.State, EventReceiveACK, ts, pkg, p)
	}
	return nil
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
