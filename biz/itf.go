package biz

import (
	"github.com/vearne/grpcreplay/protocol"
	"io"
)

// PluginReader is an interface for input plugins
type PluginReader interface {
	io.Closer
	Read() (msg *protocol.Message, err error)
}

// PluginWriter is an interface for output plugins
type PluginWriter interface {
	io.Closer
	Write(msg *protocol.Message) (err error)
}
