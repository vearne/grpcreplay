package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/vearne/grpcreplay/consts"
	"strconv"
	"strings"
)

type ProtocolV1 struct {
}

func (p *ProtocolV1) Encode(msg *Message) (bt []byte, err error) {
	buff := bytes.NewBuffer(make([]byte, 0))
	//{version} {uuid} {start-timestamp}
	buff.WriteString(fmt.Sprintf("%d %s %d", msg.Meta.Version, msg.Meta.UUID, msg.Meta.Timestamp))
	data, err := json.Marshal(msg.Data)
	if err != nil {
		return nil, err
	}
	buff.Write([]byte{'\n'})
	buff.Write(data)
	return buff.Bytes(), nil
}

func (p *ProtocolV1) Decode(bt []byte) (*Message, error) {
	var err error
	index := bytes.IndexByte(bt, '\n')
	line1 := string(bt[0:index])
	strList := strings.Split(line1, " ")
	if len(strList) != 3 {
		return nil, consts.ErrProtocal
	}
	var msg Message
	msg.Meta.Version, err = strconv.Atoi(strList[0])
	if err != nil {
		return nil, err
	}

	msg.Meta.UUID = strList[1]
	msg.Meta.Timestamp, err = strconv.Atoi(strList[2])
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bt[index+1:], &msg.Data)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
