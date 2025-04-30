package util

import (
	"bytes"
	"sync"
)

type GoroutineSafeBuffer struct {
	mu  sync.Mutex
	buf *bytes.Buffer
}

func NewGoroutineSafeBuffer() *GoroutineSafeBuffer {
	var b GoroutineSafeBuffer
	b.buf = bytes.NewBuffer([]byte{})
	return &b
}

func (g *GoroutineSafeBuffer) Write(p []byte) (n int, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.buf.Write(p)
}

func (g *GoroutineSafeBuffer) Read(p []byte) (n int, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.buf.Read(p)
}

func (g *GoroutineSafeBuffer) String() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.buf.String()
}

func (g *GoroutineSafeBuffer) Bytes() []byte {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]byte(nil), g.buf.Bytes()...)
}

func (g *GoroutineSafeBuffer) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.buf.Reset()
}

func (g *GoroutineSafeBuffer) Len() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.buf.Len()
}
