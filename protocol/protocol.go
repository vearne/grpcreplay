package protocol

type Protocol interface {
	Encode(msg *Message) (bt []byte, err error)
	Decode(bt []byte) (msg *Message, err error)
}

// Message represents data across plugins
type Message struct {
	Meta     Meta     `json:"meta"`
	Method   string   `json:"method"`
	Request  *MsgItem `json:"request"`
	Response *MsgItem `json:"response"`
}

type Meta struct {
	Version int    `json:"version"`
	UUID    string `json:"uuid"`
	// Nanosecond
	Timestamp       int64 `json:"timestamp"`
	ContainResponse bool  `json:"containResponse"`
}

type MsgItem struct {
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}
