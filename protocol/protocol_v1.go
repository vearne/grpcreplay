package protocol

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
)

type ProtocolV1 struct {
}

func (p *ProtocolV1) Encode(msg *Message) (bt []byte, err error) {
	buff := bytes.NewBuffer(make([]byte, 0))
	//{version} {uuid} {start-timestamp}
	id, _ := uuid.NewUUID()
	buff.WriteString(fmt.Sprintf("%d %s %d", 1))
}

func (p *ProtocolV1) Decode(bt []byte) (msg *Message, err error) {

}
