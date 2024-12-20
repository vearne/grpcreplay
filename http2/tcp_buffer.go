package http2

import (
	"github.com/google/gopacket/layers"
	"github.com/huandu/skiplist"
	slog "github.com/vearne/simplelog"
	"math"
	"net"
	"sync/atomic"
)

const MaxWindowSize = 65536

type TCPBuffer struct {
	//The number of bytes of data currently cached
	size              atomic.Int64
	actualCanReadSize atomic.Int64
	List              *skiplist.SkipList
	expectedSeq       uint32
	//There is at most one reader to read
	dataChannel chan []byte
	closeChan   chan struct{}
}

func NewTCPBuffer() *TCPBuffer {
	var sb TCPBuffer
	sb.List = skiplist.New(skiplist.Uint32)
	sb.size.Store(0)
	sb.actualCanReadSize.Store(0)
	sb.expectedSeq = 0
	sb.dataChannel = make(chan []byte, 100)
	sb.closeChan = make(chan struct{})
	return &sb
}

func (sb *TCPBuffer) Close() {
	close(sb.closeChan)
}

// may block
func (sb *TCPBuffer) Read(p []byte) (n int, err error) {
	var data []byte
	select {
	case <-sb.closeChan:
		err = net.ErrClosed
	case data = <-sb.dataChannel:
		n = copy(p, data)
		dataSize := int64(len(data))
		sb.size.Add(dataSize * -1)
		sb.actualCanReadSize.Add(dataSize * -1)
	}
	slog.Debug("SocketBuffer.Read, got:%v bytes", n)
	return n, err
}

func (sb *TCPBuffer) AddTCP(tcpPkg *layers.TCP) {
	slog.Debug("[start]SocketBuffer.addTCP, size:%v, actualCanReadSize:%v, expectedSeq:%v",
		sb.size.Load(), sb.actualCanReadSize.Load(), sb.expectedSeq)

	// Discard packets outside the sliding window
	if !validPackage(sb.expectedSeq, MaxWindowSize, tcpPkg.Seq) {
		slog.Debug("[end]SocketBuffer.addTCP-discard packets outside the sliding window, "+
			"size:%v, actualCanReadSize:%v, expectedSeq:%v",
			sb.size.Load(), sb.actualCanReadSize.Load(), sb.expectedSeq)
		return
	}

	// duplicate package
	if sb.List.Get(tcpPkg.Seq) != nil {
		slog.Debug("[end]SocketBuffer.addTCP-duplicate package, size:%v, actualCanReadSize:%v, expectedSeq:%v",
			sb.size.Load(), sb.actualCanReadSize.Load(), sb.expectedSeq)
		return
	}

	ele := sb.List.Set(tcpPkg.Seq, tcpPkg)
	sb.size.Add(int64(len(tcpPkg.Payload)))
	needRemoveList := make([]*skiplist.Element, 0)

	for ele != nil && sb.expectedSeq == tcpPkg.Seq {
		// expect next sequence number
		// sequence numbers may wrap around
		payloadSize := uint32(len(tcpPkg.Payload))
		sb.actualCanReadSize.Add(int64(payloadSize))
		sb.expectedSeq = (tcpPkg.Seq + payloadSize) % math.MaxUint32

		// push to channel
		sb.dataChannel <- tcpPkg.Payload
		needRemoveList = append(needRemoveList, ele)

		ele = sb.List.Get(sb.expectedSeq)
		if ele != nil {
			tcpPkg = ele.Value.(*layers.TCP)
		}
	}

	// remove
	for _, element := range needRemoveList {
		sb.List.RemoveElement(element)
	}

	slog.Debug("[end]SocketBuffer.addTCP, size:%v, actualCanReadSize:%v, expectedSeq:%v",
		sb.size.Load(), sb.actualCanReadSize.Load(), sb.expectedSeq)
}

func validPackage(expectedSeq uint32, maxWindowSize uint32, pkgSeq uint32) bool {
	rightBorder := (expectedSeq + maxWindowSize) % math.MaxUint32
	// case 1: sequence wrap around
	if rightBorder < expectedSeq {
		if (pkgSeq <= rightBorder) || (pkgSeq >= expectedSeq) {
			return true
		} else {
			return false
		}
	} else { // case 2
		if pkgSeq >= expectedSeq && pkgSeq <= expectedSeq+maxWindowSize {
			return true
		} else {
			return false
		}
	}
}
