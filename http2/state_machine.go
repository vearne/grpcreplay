package http2

import (
	"github.com/smallnest/gofsm"
	slog "github.com/vearne/simplelog"
)

const (
	//#################### Establish Connection #################

	StateListen = "LISTEN"
	// receive SYN
	StateSynReceived1 = "SYN-RECEIVED-1"
	// receive SYN and send SYN+ACK
	StateSynReceived2 = "SYN-RECEIVED-2"
	// receive ACK
	StateEstablished = "ESTABLISHED"

	//#################### Close Connection ####################
	// receive FIN
	StateCloseWait = "CLOSE_WAIT"
	// send ACk,FIN
	StateLastAck = "LAST_ACK"
	// send FIN
	StateClosed = "CLOSED"
)
const (
	EventReceiveSYN = "RECEIVE_SYN"
	EventReceiveACK = "RECEIVE_ACK"
	EventSendSYNACK = "SEND_SYN_ACK"
	EventReceiveFIN = "RECEIVE_FIN"
	EventSendACK    = "SEND_ACK"
	EventSendFIN    = "SEND_FIN"
	EventReceiveRST = "RECEIVE_RST"
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
		slog.Info("change-state, DirectConn:%v, fromState:[%v] -> toState:[%v]",
			ts.dc.String(), fromState, toState)
	case "do-nothing":
		slog.Debug("do-nothing, DirectConn:%v, current state:%v", ts.dc.String(), toState)
	default:
		slog.Debug("unknow action: %v, DirectConn:%v", action, ts.dc.String())
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
	slog.Debug("OnEnter, DirectConn:%v, connection state -> %v", ts.dc.String(), toState)
	// args []interface{}
	// ts *TCPConnectionState, pkg *NetPkg, p *Processor
	switch ts.State {
	case StateEstablished:
		p := args[2].(*Processor)
		hc := NewHttp2Conn(ts.dc, http2initialHeaderTableSize, p)
		p.ConnRepository[ts.dc] = hc
		// set sequence
		/*
			    client --> server
				pkg.TCP.ACK
				pkg.TCP.Seq
		*/
		pkg := args[1].(*NetPkg)
		hc.Input.TCPBuffer.SetExpectedSeq(pkg.TCP.Seq + 1)
		hc.Output.TCPBuffer.SetExpectedSeq(pkg.TCP.Ack)

		slog.Info("TCPEventProcessor connection [ESTABLISHED], DirectConn:%v", ts.dc.String())
	case StateClosed:
		p := args[2].(*Processor)
		delete(p.ConnStates, ts.dc)
		delete(p.ConnRepository, ts.dc)
		slog.Info("TCPEventProcessor connection [CLOSED], DirectConn:%v", ts.dc.String())
	}
}

func InitTCPFSM(processor fsm.EventProcessor) *fsm.StateMachine {
	delegate := &fsm.DefaultDelegate{P: processor}
	// https://juejin.cn/post/6844904070000410631
	// from the server's perspective
	transitions := []fsm.Transition{
		// Split SYN-RECEIVED into two states: SYN-RECEIVED-1 and SYN-RECEIVED-2
		// 1.
		{From: StateListen, Event: EventReceiveSYN, To: StateSynReceived1, Action: "change-state"},
		{From: StateSynReceived1, Event: EventSendSYNACK, To: StateSynReceived2, Action: "change-state"},
		{From: StateSynReceived2, Event: EventReceiveACK, To: StateEstablished, Action: "change-state"},

		{From: StateEstablished, Event: EventReceiveACK, To: StateEstablished, Action: "do-nothing"},
		{From: StateEstablished, Event: EventSendACK, To: StateEstablished, Action: "do-nothing"},
		{From: StateSynReceived1, Event: EventReceiveSYN, To: StateSynReceived1, Action: "do-nothing"},
		{From: StateSynReceived2, Event: EventSendSYNACK, To: StateSynReceived2, Action: "do-nothing"},
		{From: StateListen, Event: EventSendACK, To: StateListen, Action: "do-nothing"},

		// 2.
		{From: StateEstablished, Event: EventReceiveFIN, To: StateCloseWait, Action: "change-state"},
		{From: StateCloseWait, Event: EventSendACK, To: StateCloseWait, Action: "do-nothing"},
		{From: StateCloseWait, Event: EventSendFIN, To: StateLastAck, Action: "change-state"},
		{From: StateLastAck, Event: EventReceiveACK, To: StateClosed, Action: "change-state"},

		// 3.
		{From: StateEstablished, Event: EventReceiveRST, To: StateClosed, Action: "change-state"},
		{From: StateListen, Event: EventReceiveRST, To: StateClosed, Action: "change-state"},
	}

	return fsm.NewStateMachine(delegate, transitions...)
}
