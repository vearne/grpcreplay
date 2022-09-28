package plugin

import (
	"context"
	"errors"
	"fmt"
	"github.com/vearne/grpcreplay/capture"
	"github.com/vearne/grpcreplay/model"
	"github.com/vearne/grpcreplay/tcp"
	slog "github.com/vearne/simplelog"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

// ErrorStopped is the error returned when the go routines reading the input is stopped.
var ErrorStopped = errors.New("reading stopped")

// RAWInputConfig represents configuration that can be applied on raw input
type RAWInputConfig = capture.PcapOptions

// RAWInput used for intercepting traffic for given address
type RAWInput struct {
	sync.Mutex
	config         RAWInputConfig
	messageStats   []tcp.Stats
	listener       *capture.Listener
	messageParser  *tcp.MessageParser
	cancelListener context.CancelFunc
	closed         bool

	quit  chan bool // Channel used only to indicate goroutine should shutdown
	host  string
	ports []uint16
}

// NewRAWInput constructor for RAWInput. Accepts raw input config as arguments.
func NewRAWInput(address string, config RAWInputConfig) (i *RAWInput) {
	slog.Debug("address:%q", address)
	i = new(RAWInput)
	i.config = config
	i.quit = make(chan bool)

	host, _ports, err := net.SplitHostPort(address)
	if err != nil {
		// If we are reading pcap file, no port needed
		if strings.HasSuffix(address, "pcap") {
			host = address
			_ports = "0"
			err = nil
		} else if strings.HasPrefix(address, "k8s://") {
			portIndex := strings.LastIndex(address, ":")
			host = address[:portIndex]
			_ports = address[portIndex+1:]
		} else {
			log.Fatalf("input-raw: error while parsing address: %s", err)
		}
	}

	if strings.HasSuffix(host, "pcap") {
		i.config.Engine = capture.EnginePcapFile
	}

	var ports []uint16
	if _ports != "" {
		portsStr := strings.Split(_ports, ",")

		for _, portStr := range portsStr {
			port, err := strconv.Atoi(strings.TrimSpace(portStr))
			if err != nil {
				log.Fatalf("parsing port error: %v", err)
			}
			ports = append(ports, uint16(port))

		}
	}

	i.host = host
	i.ports = ports

	i.listen()
	return
}

// PluginRead reads meassage from this plugin
func (i *RAWInput) PluginRead() (*model.Message, error) {
	var msgTCP *tcp.Message
	var msg model.Message
	slog.Debug("--1---")
	select {
	case <-i.quit:
		return nil, ErrorStopped
	case msgTCP = <-i.listener.Messages():
		msg.Data = msgTCP.Data()
	}

	slog.Debug("SrcAddr:%v, DstAddr:%v", msgTCP.SrcAddr, msgTCP.DstAddr)
	var gotAck bool = false
	var seq uint32 = 0
	if msgTCP.Direction == tcp.DirIncoming {
		pkgList := msgTCP.Packets()
		for _, pkg := range pkgList {
			if pkg.ACK {
				gotAck = true
				seq = pkg.Ack
			}
		}
	} else {
	}

	if gotAck {
		sendRST(seq, msgTCP.DstAddr, msgTCP.DstPort, msgTCP.SrcAddr, msgTCP.SrcPort)
	}
	return nil, nil
}

func (i *RAWInput) listen() {
	var err error
	i.listener, err = capture.NewListener(i.host, i.ports, i.config)
	if err != nil {
		log.Fatal(err)
	}

	err = i.listener.Activate()
	if err != nil {
		log.Fatal(err)
	}

	var ctx context.Context
	ctx, i.cancelListener = context.WithCancel(context.Background())
	errCh := i.listener.ListenBackground(ctx)
	<-i.listener.Reading

	slog.Debug("RAWInput.listen")
	go func() {
		<-errCh // the listener closed voluntarily
		i.Close()
	}()
}

func (i *RAWInput) String() string {
	return fmt.Sprintf("Intercepting traffic from: %s:%s",
		i.host, strings.Join(strings.Fields(fmt.Sprint(i.ports)), ","))
}

// GetStats returns the stats so far and reset the stats
func (i *RAWInput) GetStats() []tcp.Stats {
	i.Lock()
	defer func() {
		i.messageStats = []tcp.Stats{}
		i.Unlock()
	}()
	return i.messageStats
}

// Close closes the input raw listener
func (i *RAWInput) Close() error {
	i.Lock()
	defer i.Unlock()
	if i.closed {
		return nil
	}
	i.cancelListener()
	close(i.quit)
	i.closed = true
	return nil
}

func (i *RAWInput) addStats(mStats tcp.Stats) {
	i.Lock()
	if len(i.messageStats) >= 10000 {
		i.messageStats = []tcp.Stats{}
	}
	i.messageStats = append(i.messageStats, mStats)
	i.Unlock()
}
