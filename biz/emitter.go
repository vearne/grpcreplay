package biz

import (
	"github.com/vearne/grpcreplay/filter"
	slog "github.com/vearne/simplelog"
	"io"
	"sync"
)

// Emitter represents an abject to manage plugins communication
type Emitter struct {
	sync.WaitGroup
	plugins *InOutPlugins
}

// NewEmitter creates and initializes new Emitter object.
func NewEmitter() *Emitter {
	return &Emitter{}
}

// Start initialize loop for sending data from inputs to outputs
func (e *Emitter) Start(plugins *InOutPlugins) {
	e.plugins = plugins
	for _, in := range plugins.Inputs {
		e.Add(1)
		go func(in PluginReader) {
			defer e.Done()
			if err := CopyMulty(in, plugins.Outputs...); err != nil {
				slog.Debug("[EMITTER] error during copy: %q", err)
			}
		}(in)
	}
}

// Close closes all the goroutine and waits for it to finish.
func (e *Emitter) Close() {
	for _, p := range e.plugins.All {
		if cp, ok := p.(io.Closer); ok {
			cp.Close()
		}
	}
	if len(e.plugins.All) > 0 {
		// wait for everything to stop
		e.Wait()
	}
	e.plugins.All = nil // avoid Close to make changes again
}

// CopyMulty copies from 1 reader to multiple writers
func CopyMulty(src PluginReader, writers ...PluginWriter) error {
	filterTool := filter.NewMethodExcludeFilter("grpc.reflection")
	for {
		msg, _ := src.Read()
		msg, ok := filterTool.Filter(msg)
		if ok {
			for _, dst := range writers {
				if err := dst.Write(msg); err != nil {
					slog.Error("dst.Write:%v", err)
				}
			}
		}
	}
}
