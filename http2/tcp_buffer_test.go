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

func TestValidPackage(t *testing.T) {
	testCases := []struct {
		expectedSeq   uint32
		maxWindowSize uint32
		pkgSeq        uint32
		expected      bool
	}{
		// case 1
		{4294966995, 10000, 4294967095, true},
		{4294966995, 10000, 9500, true},
		{4294966995, 10000, 4294946995, false},
		// case 2
		{10000, 10000, 10200, true},
		{10000, 10000, 3000, false},
		{10000, 10000, 20300, false},
	}
	for _, testCase := range testCases {
		actual := validPackage(testCase.expectedSeq, testCase.maxWindowSize, testCase.pkgSeq)
		assert.Equal(t, testCase.expected, actual, "Not consistent with expectations")
	}
}
