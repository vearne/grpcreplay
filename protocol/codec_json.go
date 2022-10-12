package protocol

import (
	"encoding/json"
)

const CodecJsonName = "json"

func init() {
	RegisterCodec(CodecJson{})
}

type CodecJson struct{}

func (c CodecJson) Marshal(v *Message) ([]byte, error) {
	return json.Marshal(v)
}

func (c CodecJson) Unmarshal(data []byte, v *Message) error {
	return json.Unmarshal(data, v)
}

func (c CodecJson) Name() string {
	return CodecJsonName
}
