package http2

import (
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
	slog "github.com/vearne/simplelog"
	"io"
	"testing"
)

func TestSocketBufferSequence1(t *testing.T) {
	slog.SetLevel(slog.DebugLevel)
	buffer := NewTCPBuffer()
	buffer.expectedSeq = 1000

	var tcpPkgA layers.TCP
	tcpPkgA.Seq = 1000
	tcpPkgA.Payload = []byte("aaaaaaaaaa")

	var tcpPkgB layers.TCP
	tcpPkgB.Seq = 1010
	tcpPkgB.Payload = []byte("bbbbbbbbbb")

	var tcpPkgC layers.TCP
	tcpPkgC.Seq = 1020
	tcpPkgC.Payload = []byte("cccccccccc")

	buffer.AddTCP(&tcpPkgA)
	buffer.AddTCP(&tcpPkgC)
	buffer.AddTCP(&tcpPkgB)
	//buffer.AddTCP(&tcpPkgA)

	buf := make([]byte, 1024)
	n, err := io.ReadAtLeast(buffer, buf, 30)
	// assert equality
	assert.Equal(t, 30, n, "read data")
	assert.Equal(t, "aaaaaaaaaabbbbbbbbbbcccccccccc", string(buf[0:n]), "read data")
	// assert for nil (good for errors)
	assert.Nil(t, err)
}

func TestSocketBufferSequence2(t *testing.T) {
	slog.SetLevel(slog.DebugLevel)
	buffer := NewTCPBuffer()
	buffer.expectedSeq = 1000

	var tcpPkgA layers.TCP
	tcpPkgA.Seq = 1000
	tcpPkgA.Payload = []byte("aaaaaaaaaa")

	var tcpPkgB layers.TCP
	tcpPkgB.Seq = 1010
	tcpPkgB.Payload = []byte("bbbbbbbbbb")

	var tcpPkgC layers.TCP
	tcpPkgC.Seq = 1020
	tcpPkgC.Payload = []byte("cccccccccc")

	var tcpPkgD layers.TCP
	tcpPkgD.Seq = 1030
	tcpPkgD.Payload = []byte("dddddddddd")

	buffer.AddTCP(&tcpPkgA)
	buffer.AddTCP(&tcpPkgD)
	buffer.AddTCP(&tcpPkgC)
	buffer.AddTCP(&tcpPkgB)

	buf := make([]byte, 1024)
	n, err := io.ReadAtLeast(buffer, buf, 40)
	// assert equality
	assert.Equal(t, 40, n, "read data")
	assert.Equal(t, "aaaaaaaaaabbbbbbbbbbccccccccccdddddddddd", string(buf[0:n]), "read data")
	// assert for nil (good for errors)
	assert.Nil(t, err)
}

func TestSocketBufferSequence3(t *testing.T) {
	slog.SetLevel(slog.DebugLevel)
	buffer := NewTCPBuffer()
	buffer.expectedSeq = 1000

	var tcpPkgA layers.TCP
	tcpPkgA.Seq = 1000
	tcpPkgA.Payload = []byte("aaaaaaaaaa")

	var tcpPkgB layers.TCP
	tcpPkgB.Seq = 1010
	tcpPkgB.Payload = []byte("bbbbbbbbbb")

	var tcpPkgC layers.TCP
	tcpPkgC.Seq = 1020
	tcpPkgC.Payload = []byte("cccccccccc")

	var tcpPkgD layers.TCP
	tcpPkgD.Seq = 1030
	tcpPkgD.Payload = []byte("dddddddddd")

	buffer.AddTCP(&tcpPkgC)
	buffer.AddTCP(&tcpPkgB)
	buffer.AddTCP(&tcpPkgA)
	buffer.AddTCP(&tcpPkgD)

	buf := make([]byte, 1024)
	n, err := io.ReadAtLeast(buffer, buf, 40)
	// assert equality
	assert.Equal(t, 40, n, "read data")
	assert.Equal(t, "aaaaaaaaaabbbbbbbbbbccccccccccdddddddddd", string(buf[0:n]), "read data")
	// assert for nil (good for errors)
	assert.Nil(t, err)
}

func TestSocketBufferWrapAround1(t *testing.T) {
	slog.SetLevel(slog.DebugLevel)
	buffer := NewTCPBuffer()
	buffer.expectedSeq = 4294967290

	var tcpPkgA layers.TCP
	tcpPkgA.Seq = 4294967290
	tcpPkgA.Payload = []byte("aaaaaaaaaa")

	var tcpPkgB layers.TCP
	tcpPkgB.Seq = 4
	tcpPkgB.Payload = []byte("bbbbbbbbbb")

	var tcpPkgC layers.TCP
	tcpPkgC.Seq = 14
	tcpPkgC.Payload = []byte("cccccccccc")

	buffer.AddTCP(&tcpPkgA)
	buffer.AddTCP(&tcpPkgC)
	buffer.AddTCP(&tcpPkgB)
	buffer.AddTCP(&tcpPkgA)

	buf := make([]byte, 1024)
	n, err := io.ReadAtLeast(buffer, buf, 30)
	// assert equality
	assert.Equal(t, 30, n, "read data")
	assert.Equal(t, "aaaaaaaaaabbbbbbbbbbcccccccccc", string(buf[0:n]), "read data")
	// assert for nil (good for errors)
	assert.Nil(t, err)
}

func TestSocketBufferWrapAround2(t *testing.T) {
	slog.SetLevel(slog.DebugLevel)
	buffer := NewTCPBuffer()
	buffer.expectedSeq = 4294967290

	var tcpPkgA layers.TCP
	tcpPkgA.Seq = 4294967290
	tcpPkgA.Payload = []byte("aaaaaaaaaa")

	var tcpPkgB layers.TCP
	tcpPkgB.Seq = 4
	tcpPkgB.Payload = []byte("bbbbbbbbbb")

	var tcpPkgC layers.TCP
	tcpPkgC.Seq = 14
	tcpPkgC.Payload = []byte("cccccccccc")

	buffer.AddTCP(&tcpPkgB)
	buffer.AddTCP(&tcpPkgA)
	buffer.AddTCP(&tcpPkgC)
	buffer.AddTCP(&tcpPkgA)

	buf := make([]byte, 1024)
	n, err := io.ReadAtLeast(buffer, buf, 30)
	// assert equality
	assert.Equal(t, 30, n, "read data")
	assert.Equal(t, "aaaaaaaaaabbbbbbbbbbcccccccccc", string(buf[0:n]), "read data")
	// assert for nil (good for errors)
	assert.Nil(t, err)
}

func TestSocketBufferWrapAround3(t *testing.T) {
	slog.SetLevel(slog.DebugLevel)
	buffer := NewTCPBuffer()
	buffer.expectedSeq = 4294967290

	var tcpPkgA layers.TCP
	tcpPkgA.Seq = 4294967290
	tcpPkgA.Payload = []byte("aaaaaaaaaa")

	var tcpPkgB layers.TCP
	tcpPkgB.Seq = 4
	tcpPkgB.Payload = []byte("bbbbbbbbbb")

	var tcpPkgC layers.TCP
	tcpPkgC.Seq = 14
	tcpPkgC.Payload = []byte("cccccccccc")

	buffer.AddTCP(&tcpPkgA)
	buffer.AddTCP(&tcpPkgB)
	buffer.AddTCP(&tcpPkgC)

	buf := make([]byte, 1024)
	n, err := io.ReadAtLeast(buffer, buf, 30)
	// assert equality
	assert.Equal(t, 30, n, "read data")
	assert.Equal(t, "aaaaaaaaaabbbbbbbbbbcccccccccc", string(buf[0:n]), "read data")
	// assert for nil (good for errors)
	assert.Nil(t, err)
}
