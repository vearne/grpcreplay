package model

// Message represents data across plugins
type Message struct {
	Meta []byte // metadata
	Data []byte // actual data
}
