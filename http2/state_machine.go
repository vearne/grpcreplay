package http2

import (
	"github.com/smallnest/gofsm"
	"log/slog"
)

const (
	StateListen = "LISTEN"
	// receive SYN
	StateSynReceived1 = "SYN-RECEIVED-1"
	// receive SYN and send SYN+ACK
	StateSynReceived2 = "SYN-RECEIVED-2"
	// receive ACK
	StateEstablished = "ESTABLISHED"
)
const (
	EventReceiveSYN = "RECEIVE_SYN"
	EventReceiveACK = "RECEIVE_ACK"
	EventSendSYNACK = "SEND_SYN_ACK"
)

type TCPConnectionState struct {
	dc     DirectConn
	State  string
	States []string
}

func NewTCPConnection(dc DirectConn) *TCPConnectionState {
	var t TCPConnectionState
	t.dc = dc
	t.State = StateListen
	t.States = []string{StateListen}
	return &t
}

type TCPEventProcessor struct{}

func (p *TCPEventProcessor) Action(action string, fromState string, toState string, args []interface{}) error {
	ts := args[0].(*TCPConnectionState)
	switch action {
	case "change-state":
		slog.Debug("change-state, DirectConn:%v, fromState:[%v] -> toState:[%v]\n", ts.dc.String(), fromState, toState)
	case "do-nothing":
		slog.Debug("do-nothing, DirectConn:%v, current state:%v\n", ts.dc.String(), toState)
	case "establish-connection":
		slog.Debug("establish-connection, DirectConn:%v", ts.dc.String())
	default:
		slog.Debug("unknow action: %v\n, DirectConn:%v", action, ts.dc.String())
	}
	return nil
}

func (p *TCPEventProcessor) OnActionFailure(action string, fromState string, toState string, args []interface{}, err error) {

}

func (p *TCPEventProcessor) OnExit(fromState string, args []interface{}) {
}

func (p *TCPEventProcessor) OnEnter(toState string, args []interface{}) {
	ts := args[0].(*TCPConnectionState)
	ts.State = toState
	ts.States = append(ts.States, toState)
	slog.Debug("OnEnter, DirectConn:%v, connection state -> %v\n", ts.dc.String(), toState)
	// args []interface{}
	// ts *TCPConnectionState, pkg *NetPkg, p *Processor
	if ts.State == StateEstablished {
		p := args[2].(*Processor)
		p.ConnRepository[ts.dc] = NewHttp2Conn(ts.dc, http2initialHeaderTableSize, p)
		slog.Info("TCPEventProcessor connection [ESTABLISHED], DirectConn:%v", ts.dc.String())
	}
}

func InitTCPFSM(processor fsm.EventProcessor) *fsm.StateMachine {
	delegate := &fsm.DefaultDelegate{P: processor}

	// from the server's perspective
	transitions := []fsm.Transition{
		// Split SYN-RECEIVED into two states: SYN-RECEIVED-1 and SYN-RECEIVED-2
		{From: StateListen, Event: EventReceiveSYN, To: StateSynReceived1, Action: "change-state"},
		{From: StateSynReceived1, Event: EventReceiveSYN, To: StateSynReceived1, Action: "do-nothing"},
		{From: StateSynReceived1, Event: EventSendSYNACK, To: StateSynReceived2, Action: "change-state"},
		{From: StateSynReceived2, Event: EventSendSYNACK, To: StateSynReceived2, Action: "do-nothing"},
		{From: StateSynReceived2, Event: EventReceiveACK, To: StateEstablished, Action: "establish-connection"},
		{From: StateEstablished, Event: EventReceiveACK, To: StateEstablished, Action: "do-nothing"},
	}

	return fsm.NewStateMachine(delegate, transitions...)
}
