package model

// Message represents data across plugins
type Message struct {
	Meta    map[string][]string `json:"meta"`
	Method  string              `json:"method"`
	Request string              `json:"request"`
}
