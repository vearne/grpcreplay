package http2

import (
	"bytes"
	"github.com/google/gopacket/layers"
	"github.com/huandu/skiplist"
	slog "github.com/vearne/simplelog"
	"math"
	"net"
)

type TCPBuffer struct {
	//The number of bytes of data currently cached
	size              uint32
	actualCanReadSize uint32
	List              *skiplist.SkipList
	expectedSeq       int64
	// The sliding window contains the leftPointer
	leftPointer int64

	//There is at most one reader to read
	dataChannel chan []byte
	closeChan   chan struct{}
}

func NewTCPBuffer() *TCPBuffer {
	var sb TCPBuffer
	sb.List = skiplist.New(skiplist.Uint32)
	sb.size = 0
	sb.actualCanReadSize = 0
	sb.expectedSeq = -1
	sb.leftPointer = -1
	sb.dataChannel = make(chan []byte, 10)
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
	}
	slog.Debug("SocketBuffer.Read, got:%v bytes", n)
	return n, err
}

func (sb *TCPBuffer) AddTCP(tcpPkg *layers.TCP) {
	sb.addTCP(tcpPkg)

	if sb.actualCanReadSize > 0 {
		slog.Debug("SocketBuffer.AddTCP, satisfy the conditions, size:%v, actualCanReadSize:%v, expectedSeq:%v",
			sb.size, sb.actualCanReadSize, sb.expectedSeq)
		data := sb.getData()
		slog.Debug("push to channel: %v bytes", len(data))
		sb.dataChannel <- data
	}
}

func (sb *TCPBuffer) addTCP(tcpPkg *layers.TCP) {
	slog.Debug("[start]SocketBuffer.addTCP, size:%v, actualCanReadSize:%v, expectedSeq:%v",
		sb.size, sb.actualCanReadSize, sb.expectedSeq)

	// duplicate package
	if int64(tcpPkg.Seq) < sb.leftPointer || sb.List.Get(tcpPkg.Seq) != nil {
		slog.Debug("[end]SocketBuffer.addTCP-duplicate package, size:%v, actualCanReadSize:%v, expectedSeq:%v",
			sb.size, sb.actualCanReadSize, sb.expectedSeq)
		return
	}

	ele := sb.List.Set(tcpPkg.Seq, tcpPkg)
	sb.size += uint32(len(tcpPkg.Payload))

	for ele != nil && sb.expectedSeq == int64(tcpPkg.Seq) {
		// expect next sequence number
		sb.expectedSeq = int64((tcpPkg.Seq + uint32(len(tcpPkg.Payload))) % math.MaxUint32)
		sb.actualCanReadSize += uint32(len(tcpPkg.Payload))

		ele = ele.Next()
		if ele != nil {
			tcpPkg = ele.Value.(*layers.TCP)
		}
	}
	slog.Debug("[end]SocketBuffer.addTCP, size:%v, actualCanReadSize:%v, expectedSeq:%v",
		sb.size, sb.actualCanReadSize, sb.expectedSeq)
}

func (sb *TCPBuffer) getData() []byte {
	slog.Debug("[start]SocketBuffer.getData, size:%v, actualCanReadSize:%v, expectedSeq:%v",
		sb.size, sb.actualCanReadSize, sb.expectedSeq)

	var tcpPkg *layers.TCP
	buf := bytes.NewBuffer([]byte{})
	ele := sb.List.Front()
	if ele != nil {
		tcpPkg = ele.Value.(*layers.TCP)
	}

	needRemoveList := make([]*skiplist.Element, 0)
	for ele != nil && int64(tcpPkg.Seq) <= sb.expectedSeq {
		sb.actualCanReadSize -= uint32(len(tcpPkg.Payload))
		sb.size -= uint32(len(tcpPkg.Payload))
		sb.leftPointer += int64(len(tcpPkg.Payload))

		buf.Write(tcpPkg.Payload)
		needRemoveList = append(needRemoveList, ele)

		ele = ele.Next()
		if ele != nil {
			tcpPkg = ele.Value.(*layers.TCP)
		}
	}

	// remove
	for _, element := range needRemoveList {
		sb.List.RemoveElement(element)
	}

	slog.Debug("[end]SocketBuffer.getData, size:%v, actualCanReadSize:%v, expectedSeq:%v, data: %v bytes",
		sb.size, sb.actualCanReadSize, sb.expectedSeq, buf.Len())
	return buf.Bytes()
}
