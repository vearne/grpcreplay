package plugin

import (
	"fmt"
	"github.com/google/gopacket"
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
	snapshotLen int32         = 1024
	promiscuous bool          = false
	timeout     time.Duration = 5 * time.Second
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

	var filter string = fmt.Sprintf("tcp and port %v", l.port)
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
			//l.rawInput.outputChan <- packet
			conn := netPkg.DirectConn()
			if l.rawInput.connSet.Has(conn) { // history connection
				if netPkg.TCP.ACK {
					slog.Debug("send RST, for connection:%v", &conn)
					// 伪造一个从local -> remote的packet
					sendFakePkg(netPkg.TCP.Ack, conn.DstAddr.IP, uint16(conn.DstAddr.Port),
						conn.SrcAddr.IP, uint16(conn.SrcAddr.Port), RST)
				} else if netPkg.TCP.SYN { // // new connection
					slog.Debug("got SYN, remove %v from connSet", &conn)
					l.rawInput.connSet.Remove(conn)
				}
			} else { // new connection
				l.rawInput.outputChan <- netPkg
			}
		}
	}
	return nil
}

func (l *DeviceListener) Close() {
	l.handle.Close()
}

// RAWInput used for intercepting traffic for given address
type RAWInput struct {
	connSet      *http2.ConnSet
	ipSet        *util.StringSet
	port         int
	outputChan   chan *http2.NetPkg
	listenerList []*DeviceListener
	Processor    *http2.Processor
}

func findOneServerAddr(host string, port int) string {
	if len(host) <= 0 {
		itfStatList, err := psnet.Interfaces()
		if err != nil {
			panic(err)
		}
		for _, itf := range itfStatList {
			for _, addr := range itf.Addrs {
				idx := strings.LastIndex(addr.Addr, "/")
				ip := addr.Addr[0:idx]
				if util.IsIPv4(ip) {
					return fmt.Sprintf("%v:%v", ip, port)
				}
			}
		}
	}
	return fmt.Sprintf("%v:%v", host, port)
}

// NewRAWInput constructor for RAWInput. Accepts raw input config as arguments.
func NewRAWInput(address string) (*RAWInput, error) {
	slog.Debug("address:%q", address)

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	var i RAWInput
	i.connSet = http2.NewConnSet()
	i.port, err = strconv.Atoi(port)
	if err != nil {
		slog.Fatal("RAWInput, port error:%v", port)
	}
	i.ipSet = util.NewStringSet()
	i.outputChan = make(chan *http2.NetPkg, 100)

	i.Processor = http2.NewProcessor(i.outputChan, findOneServerAddr(host, i.port))

	var deviceList []string
	itfStatList, err := psnet.Interfaces()
	if err != nil {
		return nil, err
	}

	// 保存本地所有IP地址，以便后期判断包的来源
	for _, itf := range itfStatList {
		for _, addr := range itf.Addrs {
			idx := strings.LastIndex(addr.Addr, "/")
			//slog.Debug("addr: %v", addr.Addr[0:idx])
			i.ipSet.Add(addr.Addr[0:idx])
		}
	}

	slog.Info("ipSet:%v", i.ipSet.ToArray())

	host = strings.TrimSpace(host)
	if len(host) <= 0 || host == "0.0.0.0" { // all devices
		for _, itf := range itfStatList {
			deviceList = append(deviceList, itf.Name)
		}
	} else {
		for _, itf := range itfStatList {
			for _, addr := range itf.Addrs {
				if strings.HasPrefix(addr.Addr, host) {
					deviceList = append(deviceList, itf.Name)
				}
			}
		}
	}

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
	// 在每个网卡启动listener
	slog.Debug("len(i.listenerList):%v", len(i.listenerList))

	for _, listener := range i.listenerList {
		go func(listener *DeviceListener) {
			listenErr := listener.listen()
			if listenErr != nil {
				slog.Fatal("listener.listen:%v", listenErr)
			}
		}(listener)
	}

	// 直到所有的旧有连接都退出
	for i.connSet.Size() > 0 {
		time.Sleep(3 * time.Second)
		cons, err := listAllConns(i.port)
		if err != nil {
			slog.Fatal("listAllConns:%v", err)
		}

		// A∩B
		B := http2.NewConnSet()
		B.AddAll(cons)

		A := i.connSet.Clone()
		A.RemoveAll(B)
		i.connSet.RemoveAll(A)
		for _, conn := range i.connSet.ToArray() {
			// trigger challenge ack
			// remote -> local
			// src -> dst
			// Fake a packet from local -> remote
			sendFakePkg(uint32(rand.Intn(100)), conn.DstAddr.IP, uint16(conn.DstAddr.Port),
				conn.SrcAddr.IP, uint16(conn.SrcAddr.Port), SYN)
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
