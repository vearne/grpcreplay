package plugin

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/vearne/grpcreplay/consts"
	"github.com/vearne/grpcreplay/http2"
	"github.com/vearne/grpcreplay/model"
	"github.com/vearne/grpcreplay/protocol"
	"github.com/vearne/grpcreplay/util"
	slog "github.com/vearne/simplelog"
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
	port   string
	//outputChan chan gopacket.Packet
	rawInput *RAWInput
	handle   *pcap.Handle
}

func NewDeviceListener(device, port string, rawInput *RAWInput) *DeviceListener {
	var l DeviceListener
	l.device = device
	l.port = port
	l.rawInput = rawInput
	return &l
}

func (l *DeviceListener) listen() error {
	var err error
	l.handle, err = pcap.OpenLive(l.device, snapshotLen, promiscuous, timeout)
	if err != nil {
		return err
	}

	var filter string = fmt.Sprintf("tcp and port %v", l.port)
	err = l.handle.SetBPFFilter(filter)
	if err != nil {
		return err
	}
	packetSource := gopacket.NewPacketSource(l.handle, l.handle.LinkType())
	for packet := range packetSource.Packets() {
		netPkg, err := model.ProcessPacket(packet, l.rawInput.ipSet)
		if err != nil {
			slog.Error("netPkg error:%v", err)
			continue
		}
		if netPkg.Direction == consts.DirIncoming {
			//l.rawInput.outputChan <- packet
			conn := netPkg.DirectConn()
			if l.rawInput.connSet.Has(conn) { // history connection
				if netPkg.TCP.ACK {
					// trigger challenge ack
					sendFakePkg(netPkg.TCP.Ack, conn.SrcAddr.IP, uint16(conn.SrcAddr.Port),
						conn.DstAddr.IP, uint16(conn.DstAddr.Port), RST)
				} else if netPkg.TCP.SYN { // // new connection
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
	connSet      *util.ConnSet
	ipSet        *util.StringSet
	port         int
	outputChan   chan *model.NetPkg
	listenerList []*DeviceListener
}

// NewRAWInput constructor for RAWInput. Accepts raw input config as arguments.
func NewRAWInput(address string) (*RAWInput, error) {
	slog.Debug("address:%q", address)

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	var i RAWInput
	i.connSet = util.NewConnSet()
	i.port, err = strconv.Atoi(port)
	i.ipSet = util.NewStringSet()

	var deviceList []string
	itfStatList, err := psnet.Interfaces()
	if err != nil {
		return nil, err
	}

	// 保存本地所有IP地址，以便后期判断包的来源
	for _, itf := range itfStatList {
		for _, addr := range itf.Addrs {
			addrStr := addr.String()
			idx := strings.LastIndex(addrStr, "/")
			i.ipSet.Add(addrStr[0:idx])
		}
	}

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

	if err != nil {
		return nil, err
	}
	i.outputChan = make(chan *model.NetPkg, 100)
	i.listenerList = make([]*DeviceListener, 0)
	for _, device := range deviceList {
		i.listenerList = append(i.listenerList, NewDeviceListener(device, port, &i))
	}

	go i.Listen()
	return &i, nil
}

func (i *RAWInput) Listen() {
	cons, err := listAllConns(i.port)
	if err != nil {
		slog.Fatal("listAllConns:%v", err)
	}
	i.connSet.AddAll(cons)
	// 在每个网卡启动listener
	for _, listener := range i.listenerList {
		go func() {
			listenErr := listener.listen()
			slog.Fatal("listener.listen:%v", listenErr)
		}()
	}

	// 直到所有的旧有连接都退出
	for i.connSet.Size() > 0 {
		time.Sleep(1 * time.Second)
		cons, err := listAllConns(i.port)
		if err != nil {
			slog.Fatal("listAllConns:%v", err)
		}

		// A∩B
		B := util.NewConnSet()
		B.AddAll(cons)

		A := i.connSet.Clone()
		A.RemoveAll(B)
		i.connSet.RemoveAll(A)
		for _, conn := range i.connSet.ToArray() {
			// trigger challenge ack
			sendFakePkg(10, conn.SrcAddr.IP, uint16(conn.SrcAddr.Port),
				conn.DstAddr.IP, uint16(conn.DstAddr.Port), SYN)
		}
	}
	slog.Info("All history connections has exited.")
}

// PluginRead reads meassage from this plugin
func (i *RAWInput) Read() (*protocol.Message, error) {
	var finish bool = false
	for !finish {
		pkg := <-i.outputChan
		payload := pkg.TCP.Payload
		if len(payload) <= http2.HeaderSize {
			continue
		}
		
	}

	return nil, nil
}

// Close closes the input raw listener
func (i *RAWInput) Close() error {
	for _, listener := range i.listenerList {
		listener.Close()
	}
	return nil
}

func listAllConns(port int) ([]model.DirectConn, error) {
	itemList, err := psnet.Connections("tcp4")
	if err != nil {
		return nil, err
	}
	conns := make([]model.DirectConn, 0)
	for _, item := range itemList {
		if item.Laddr.Port == uint32(port) && item.Status == "ESTABLISHED" {
			var c model.DirectConn
			c.DstAddr = item.Laddr
			c.SrcAddr = item.Raddr
			conns = append(conns, c)
		}
	}
	return conns, nil
}
