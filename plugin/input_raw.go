package plugin

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/vearne/grpcreplay/http2"
	"github.com/vearne/grpcreplay/protocol"
	"github.com/vearne/grpcreplay/util"
	slog "github.com/vearne/simplelog"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	snapshotLen int32         = 1024 * 1024
	promiscuous bool          = false
	timeout     time.Duration = 5 * time.Second
	RSTNum                    = 3
)

type DeviceListener struct {
	device string
	port   int
	//outputChan chan gopacket.Packet
	rawInput *RAWInput
	handle   *pcap.Handle
}

func NewDeviceListener(device string, port int, rawInput *RAWInput) *DeviceListener {
	var l DeviceListener
	l.device = device
	l.port = port
	l.rawInput = rawInput
	return &l
}

func (l *DeviceListener) String() string {
	return fmt.Sprintf("device:%v, port:%v", l.device, l.port)
}

func (l *DeviceListener) listen() error {
	var err error
	l.handle, err = pcap.OpenLive(l.device, snapshotLen, promiscuous, timeout)
	if err != nil {
		return err
	}

	var filter = fmt.Sprintf("tcp and port %v", l.port)
	slog.Info("listener:%v, filter:%v", l, filter)
	err = l.handle.SetBPFFilter(filter)
	if err != nil {
		return err
	}
	packetSource := gopacket.NewPacketSource(l.handle, l.handle.LinkType())
	for packet := range packetSource.Packets() {
		netPkg, err := http2.ProcessPacket(packet, l.rawInput.ipSet, l.port)
		if err != nil {
			slog.Error("netPkg error:%v", err)
			continue
		}
		conn := netPkg.DirectConn()
		slog.Debug("DeviceListener.listen-connection:%v, Direction:%v",
			&conn, http2.GetDirection(netPkg.Direction))
		if netPkg.Direction == http2.DirIncoming {
			if l.rawInput.connSet.Has(conn) { // history connection
				if netPkg.TCP.ACK && netPkg.IPv4 != nil {
					slog.Debug("receive Ack package, for connection:%v, expected seq:%v, window:%v",
						&conn, netPkg.TCP.Ack, netPkg.TCP.Window)
					actualSeq := netPkg.TCP.Ack
					for i := 0; i < RSTNum; i++ {
						actualSeq += uint32(netPkg.TCP.Window) * uint32(i)
						slog.Debug("send RST, for connection:%v, seq:%v", &conn, actualSeq)
						// forge a packet from local -> remote
						err = SendRST(netPkg.Ethernet.DstMAC, netPkg.Ethernet.SrcMAC,
							netPkg.IPv4.DstIP, netPkg.IPv4.SrcIP, netPkg.TCP.DstPort,
							netPkg.TCP.SrcPort, actualSeq, l.handle)
						if err != nil {
							slog.Error("SendRST, for connection:%v, error:%v", &conn, err)
						}
					}
				} else if netPkg.TCP.SYN { // // new connection
					slog.Debug("got SYN, remove %v from connSet", &conn)
					l.rawInput.connSet.Remove(conn)
				}
			} else { // new connection
				l.rawInput.outputChan <- netPkg
			}
		} else if netPkg.Direction == http2.DirOutcoming {
			l.rawInput.outputChan <- netPkg
		}
	}
	return nil
}

func (l *DeviceListener) Close() {
	l.handle.Close()
}

// RAWInput used for intercepting traffic for given address
type RAWInput struct {
	connSet        *http2.ConnSet
	ipSet          *util.StringSet
	port           int
	outputChan     chan *http2.NetPkg
	listenerList   []*DeviceListener
	Processor      *http2.Processor
	recordResponse bool
}

// NewRAWInput constructor for RAWInput. Accepts raw input config as arguments.
func NewRAWInput(address string, recordResponse bool, finder http2.PBFinder) (*RAWInput, error) {
	slog.Debug("address:%q", address)

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	var i RAWInput
	i.recordResponse = recordResponse
	i.connSet = http2.NewConnSet()
	i.port, err = strconv.Atoi(port)
	if err != nil {
		slog.Fatal("RAWInput, port error:%v", port)
	}
	i.ipSet = util.NewStringSet()
	i.outputChan = make(chan *http2.NetPkg, 100)
	i.Processor = http2.NewProcessor(i.outputChan, recordResponse, finder)

	var deviceList []string
	itfStatList, err := psnet.Interfaces()
	if err != nil {
		return nil, err
	}

	// save all local IP addresses to determine the source of the packet later
	for _, itf := range itfStatList {
		if itf.MTU > 0 {
			for _, addr := range itf.Addrs {
				idx := strings.LastIndex(addr.Addr, "/")
				//slog.Debug("addr: %v", addr.Addr[0:idx])
				i.ipSet.Add(addr.Addr[0:idx])
			}
		}
	}

	slog.Info("ipSet:%v", i.ipSet.ToArray())

	host = strings.TrimSpace(host)
	if len(host) <= 0 || host == "0.0.0.0" { // all devices
		for _, itf := range itfStatList {
			if itf.MTU > 0 {
				slog.Debug("interface:%v", itf.Name)
				deviceList = append(deviceList, itf.Name)
			}
		}
	} else {
		for _, itf := range itfStatList {
			if itf.MTU > 0 {
				for _, addr := range itf.Addrs {
					slog.Debug("interface:%v, addr:%v, host:%v", itf.Name, addr.Addr, host)
					if strings.HasPrefix(addr.Addr, host) {
						deviceList = append(deviceList, itf.Name)
					}
				}
			}
		}
	}

	slog.Info("deviceList:%v", deviceList)
	i.listenerList = make([]*DeviceListener, 0)
	for j := 0; j < len(deviceList); j++ {
		i.listenerList = append(i.listenerList, NewDeviceListener(deviceList[j], i.port, &i))
	}

	go i.Listen()
	go i.Processor.ProcessTCPPkg()
	return &i, nil
}

func (i *RAWInput) Listen() {
	slog.Debug("RAWInput.Listen()")
	cons, err := listAllConns(i.port)
	if err != nil {
		slog.Fatal("listAllConns:%v", err)
	}
	i.connSet.AddAll(cons)
	slog.Debug("history connections:%v", i.connSet)
	// start the listener on each network card
	slog.Debug("len(i.listenerList):%v", len(i.listenerList))

	for _, listener := range i.listenerList {
		go listenerRun(listener)
	}

	// Until all old connections are exited
	for i.connSet.Size() > 0 {
		time.Sleep(3 * time.Second)
		cons, err := listAllConns(i.port)
		if err != nil {
			slog.Fatal("listAllConns:%v", err)
		}

		// A∩B
		newSet := http2.NewConnSet()
		newSet.AddAll(cons)

		i.connSet = i.connSet.Intersection(newSet)
		for _, conn := range i.connSet.ToArray() {
			// trigger challenge ack
			// remote -> local
			// src -> dst
			// Fake a packet from local -> remote
			err = SendSYN(IPtoByte(conn.DstAddr.IP), IPtoByte(conn.SrcAddr.IP),
				layers.TCPPort(conn.DstAddr.Port),
				layers.TCPPort(conn.SrcAddr.Port),
				uint32(rand.Intn(100)))
			if err != nil {
				slog.Error("SendSYN, for connection:%v, error:%v", &conn, err)
			}
		}
		slog.Debug("history connections:%v", i.connSet)
	}
	slog.Info("All history connections has exited.")
}

// PluginRead reads meassage from this plugin
func (i *RAWInput) Read() (*protocol.Message, error) {
	msg := <-i.Processor.OutputChan
	return msg, nil
}

// Close closes the input raw listener
func (i *RAWInput) Close() error {
	for _, listener := range i.listenerList {
		listener.Close()
	}
	return nil
}

func listAllConns(port int) ([]http2.DirectConn, error) {
	itemList, err := psnet.Connections("tcp4")
	if err != nil {
		return nil, err
	}
	conns := make([]http2.DirectConn, 0)
	for _, item := range itemList {
		if item.Laddr.Port == uint32(port) && item.Status == "ESTABLISHED" {
			var c http2.DirectConn
			c.DstAddr = item.Laddr
			c.SrcAddr = item.Raddr
			conns = append(conns, c)
		}
	}
	return conns, nil
}

func IPtoByte(ipStr string) []byte {
	return net.ParseIP(ipStr).To4()
}

func listenerRun(listener *DeviceListener) {
	listenErr := listener.listen()
	if listenErr != nil {
		slog.Fatal("listener.listen:%v", listenErr)
	}
}
