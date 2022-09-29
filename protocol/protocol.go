package protocol

type Protocol interface {
	Encode(msg *Message) (bt []byte, err error)
	Decode(bt []byte) (msg *Message, err error)
}

// Message represents data across plugins
type Message struct {
	Meta struct {
		Version   int    `json:"version"`
		UUID      string `json:"uuid"`
		StartTime int64  `json:"startTime"`
	}
	Data struct {
		Headers map[string][]string `json:"headers"`
		Method  string              `json:"method"`
		Request string              `json:"request"`
	}
}
