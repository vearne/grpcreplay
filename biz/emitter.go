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
	plugins     *InOutPlugins
	filterChain filter.Filter
	limiter     Limiter
}

// NewEmitter creates and initializes new Emitter object.
func NewEmitter(f filter.Filter, lim Limiter) *Emitter {
	var e Emitter
	e.filterChain = f
	e.limiter = lim
	return &e
}

// Start initialize loop for sending data from inputs to outputs
func (e *Emitter) Start(plugins *InOutPlugins) {
	e.plugins = plugins
	for _, in := range plugins.Inputs {
		e.Add(1)
		go func(in PluginReader) {
			defer e.Done()
			if err := e.CopyMulty(in, plugins.Outputs...); err != nil {
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
func (e *Emitter) CopyMulty(src PluginReader, writers ...PluginWriter) error {
	for {
		msg, err := src.Read()
		if err != nil {
			slog.Error("src.Read:%v", err)
			continue
		}
		msg, ok := e.filterChain.Filter(msg)
		if !ok {
			continue
		}

		if e.limiter != nil && !e.limiter.Allow() {
			continue
		}

		for _, dst := range writers {
			if err = dst.Write(msg); err != nil {
				slog.Error("dst.Write:%v", err)
			}
		}
	}
}
