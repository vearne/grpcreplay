package http2

import (
	"log"
	"testing"
)

type TestEventProcessor struct{}

func (p *TestEventProcessor) Action(action string, fromState string, toState string, args []interface{}) error {
	switch action {
	case "change-state":
		log.Printf("change-state, fromState:[%v] -> toState:[%v]\n", fromState, toState)
	case "do-nothing":
		log.Printf("do-nothing, current state:%v\n", toState)
	case "establish-connection":
		log.Printf("establish-connection")
	default:
		log.Printf("unknow action: %v\n", action)
	}
	return nil
}

func (p *TestEventProcessor) OnActionFailure(action string, fromState string, toState string, args []interface{}, err error) {

}

func (p *TestEventProcessor) OnExit(fromState string, args []interface{}) {
}

func (p *TestEventProcessor) OnEnter(toState string, args []interface{}) {
	log.Printf("OnEnter, connection state -> %v\n", toState)
	ts := args[0].(*TCPConnectionState)
	ts.State = toState
	ts.States = append(ts.States, toState)
}

func TestTCPFSM(t *testing.T) {
	ts := &TCPConnectionState{
		State:  StateListen,
		States: []string{StateListen},
	}
	tcpFSM := InitTCPFSM(&TestEventProcessor{})

	var err error
	err = tcpFSM.Trigger(ts.State, EventReceiveSYN, ts)
	if err != nil {
		t.Errorf("trigger err: %v", err)
	}

	err = tcpFSM.Trigger(ts.State, EventSendSYNACK, ts)
	if err != nil {
		t.Errorf("trigger err: %v", err)
	}

	err = tcpFSM.Trigger(ts.State, EventReceiveACK, ts)
	if err != nil {
		t.Errorf("trigger err: %v", err)
	}

	t.Logf("current state:%v", ts.State)
	if ts.State != StateEstablished {
		t.Errorf("expect:%v, actual:%v", StateEstablished, ts.State)
	}
}
