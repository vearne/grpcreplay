package biz

import "github.com/vearne/grpcreplay/model"

// PluginReader is an interface for input plugins
type PluginReader interface {
	PluginRead() (msg *model.Message, err error)
}

// PluginWriter is an interface for output plugins
type PluginWriter interface {
	PluginWrite(msg *model.Message) (n int, err error)
}
