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
	// line 1
	//{version} {uuid} {start-timestamp} {containResponse}
	buff.WriteString(fmt.Sprintf("%d %s %d %d", msg.Meta.Version, msg.Meta.UUID,
		msg.Meta.Timestamp, bool2Int(msg.Meta.ContainResponse)))
	buff.Write([]byte{'\n'})
	// line 2
	// method
	buff.WriteString(msg.Method)
	buff.Write([]byte{'\n'})
	// line 3
	// request
	data, err := json.Marshal(msg.Request)
	if err != nil {
		return nil, err
	}
	buff.Write(data)
	buff.Write([]byte{'\n'})
	// line 4
	// response (optional)
	if msg.Meta.ContainResponse {
		data, err = json.Marshal(msg.Response)
		if err != nil {
			return nil, err
		}
		buff.Write(data)
		buff.Write([]byte{'\n'})
	}

	return buff.Bytes(), nil
}

func (c CodecSimple) Unmarshal(data []byte, msg *Message) error {
	var err error
	lines := bytes.Split(data, []byte{'\n'})
	// line 1
	line1 := string(lines[0])
	strList := strings.Split(line1, " ")
	if len(strList) != 4 {
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
	tmp, err = strconv.Atoi(strList[3])
	if err != nil {
		return err
	}
	msg.Meta.ContainResponse = int2bool(tmp)
	// line 2
	msg.Method = string(lines[1])
	// line 3
	msg.Request = &MsgItem{}
	err = json.Unmarshal(lines[2], &msg.Request)
	if err != nil {
		return err
	}
	// line 4
	if msg.Meta.ContainResponse && len(lines) == 4 {
		msg.Response = &MsgItem{}
		err = json.Unmarshal(lines[3], &msg.Response)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c CodecSimple) Name() string {
	return CodecSimpleName
}

func bool2Int(b bool) int {
	if b {
		return 1
	}
	return 0
}

func int2bool(v int) bool {
	return v > 0
}
