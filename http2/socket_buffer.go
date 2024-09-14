package http2

import (
	"bytes"
	"github.com/google/gopacket/layers"
	"github.com/huandu/skiplist"
	slog "github.com/vearne/simplelog"
	"math"
	"sync"
	"sync/atomic"
)

type SocketBuffer struct {
	lock sync.Mutex
	//The number of bytes of data currently cached
	size              uint32
	actualCanReadSize atomic.Uint32
	List              *skiplist.SkipList
	expectedSeq       int64
	leftPointer       int64

	//There is at most one reader to read
	dataChannel chan []byte
}

func NewSocketBuffer() *SocketBuffer {
	var sb SocketBuffer
	sb.List = skiplist.New(skiplist.Uint32)
	sb.size = 0
	sb.actualCanReadSize.Store(0)
	sb.expectedSeq = -1
	sb.leftPointer = -1
	sb.dataChannel = make(chan []byte, 10)
	return &sb
}

// may block
func (sb *SocketBuffer) Read(p []byte) (n int, err error) {
	slog.Debug("[start]SocketBuffer.Read")
	var data []byte
	if sb.actualCanReadSize.Load() > 0 {
		slog.Debug("SocketBuffer.Read, satisfy the conditions, size:%v, actualCanReadSize:%v, expectedSeq:%v",
			sb.size, sb.actualCanReadSize.Load(), sb.expectedSeq)
		data = sb.getData()
	} else {
		data = <-sb.dataChannel
	}
	n = copy(p, data)
	slog.Debug("[end]SocketBuffer.Read, got:%v bytes", n)
	return n, nil
}

func (sb *SocketBuffer) AddTCP(tcpPkg *layers.TCP) {
	sb.addTCP(tcpPkg)

	if sb.actualCanReadSize.Load() > 0 {
		slog.Debug("SocketBuffer.AddTCP, satisfy the conditions, size:%v, actualCanReadSize:%v, expectedSeq:%v",
			sb.size, sb.actualCanReadSize.Load(), sb.expectedSeq)
		data := sb.getData()
		slog.Debug("push to channel: %v bytes", len(data))
		sb.dataChannel <- data
	}
}

func (sb *SocketBuffer) addTCP(tcpPkg *layers.TCP) {
	slog.Debug("[start]SocketBuffer.addTCP, size:%v, actualCanReadSize:%v, expectedSeq:%v",
		sb.size, sb.actualCanReadSize.Load(), sb.expectedSeq)

	sb.lock.Lock()
	defer sb.lock.Unlock()

	// duplicate package
	if int64(tcpPkg.Seq) < sb.leftPointer || sb.List.Get(tcpPkg.Seq) != nil {
		slog.Debug("[end]SocketBuffer.addTCP-duplicate package, size:%v, actualCanReadSize:%v, expectedSeq:%v",
			sb.size, sb.actualCanReadSize.Load(), sb.expectedSeq)
		return
	}

	ele := sb.List.Set(tcpPkg.Seq, tcpPkg)
	sb.size += uint32(len(tcpPkg.Payload))

	for ele != nil && sb.expectedSeq == int64(tcpPkg.Seq) {
		// expect next sequence number
		sb.expectedSeq = int64((tcpPkg.Seq + uint32(len(tcpPkg.Payload))) % math.MaxUint32)
		sb.actualCanReadSize.Store(sb.actualCanReadSize.Load() + uint32(len(tcpPkg.Payload)))

		ele = ele.Next()
		if ele != nil {
			tcpPkg = ele.Value.(*layers.TCP)
		}
	}
	slog.Debug("[end]SocketBuffer.addTCP, size:%v, actualCanReadSize:%v, expectedSeq:%v",
		sb.size, sb.actualCanReadSize.Load(), sb.expectedSeq)
}

func (sb *SocketBuffer) getData() []byte {
	slog.Debug("[start]SocketBuffer.getData, size:%v, actualCanReadSize:%v, expectedSeq:%v",
		sb.size, sb.actualCanReadSize.Load(), sb.expectedSeq)

	sb.lock.Lock()
	defer sb.lock.Unlock()

	var tcpPkg *layers.TCP
	buf := bytes.NewBuffer([]byte{})
	ele := sb.List.Front()
	if ele != nil {
		tcpPkg = ele.Value.(*layers.TCP)
	}

	needRemoveList := make([]*skiplist.Element, 0)
	for ele != nil && int64(tcpPkg.Seq) <= sb.expectedSeq {
		sb.actualCanReadSize.Store(sb.actualCanReadSize.Load() - uint32(len(tcpPkg.Payload)))
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
		sb.size, sb.actualCanReadSize.Load(), sb.expectedSeq, buf.Len())
	return buf.Bytes()
}
