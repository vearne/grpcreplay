package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/vearne/grpcreplay/consts"
	"strconv"
	"strings"
)

const CodecSimpleName = "simple"

func init() {
	RegisterCodec(CodecSimple{})
}

type CodecSimple struct{}

func (c CodecSimple) Marshal(msg *Message) ([]byte, error) {
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

func (c CodecSimple) Unmarshal(data []byte, msg *Message) error {
	var err error
	index := bytes.IndexByte(data, '\n')
	line1 := string(data[0:index])
	strList := strings.Split(line1, " ")
	if len(strList) != 3 {
		return consts.ErrProtocal
	}
	msg.Meta.Version, err = strconv.Atoi(strList[0])
	if err != nil {
		return err
	}

	msg.Meta.UUID = strList[1]
	tmp, err := strconv.Atoi(strList[2])
	msg.Meta.Timestamp = int64(tmp)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data[index+1:], &msg.Data)
	if err != nil {
		return err
	}
	return nil
}

func (c CodecSimple) Name() string {
	return CodecSimpleName
}
