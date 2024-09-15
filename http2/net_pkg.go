package http2

import (
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/vearne/grpcreplay/util"
)

type DirectConn struct {
	SrcAddr psnet.Addr
	DstAddr psnet.Addr
}

func (d *DirectConn) String() string {
	return fmt.Sprintf("%v:%v -> %v:%v", d.SrcAddr.IP,
		d.SrcAddr.Port, d.DstAddr.IP, d.DstAddr.Port)
}

type Dir uint8

type NetPkg struct {
	SrcIP string
	DstIP string

	Ethernet  *layers.Ethernet
	IPv4      *layers.IPv4
	IPv6      *layers.IPv6
	TCP       *layers.TCP
	Direction Dir
}

func ProcessPacket(packet gopacket.Packet, ipSet *util.StringSet, port int) (*NetPkg, error) {
	var p NetPkg

	ethernet := packet.Layer(layers.LayerTypeEthernet)
	ipLayerIPv4 := packet.Layer(layers.LayerTypeIPv4)
	ipLayerIPv6 := packet.Layer(layers.LayerTypeIPv6)
	if ethernet == nil || (ipLayerIPv4 == nil && ipLayerIPv6 == nil) {
		return nil, errors.New("invalid IP package")
	}

	p.Ethernet = ethernet.(*layers.Ethernet)
	if ipLayerIPv4 != nil {
		p.IPv4 = ipLayerIPv4.(*layers.IPv4)
		p.SrcIP = p.IPv4.SrcIP.String()
		p.DstIP = p.IPv4.DstIP.String()
	}
	if ipLayerIPv6 != nil {
		p.IPv6 = ipLayerIPv6.(*layers.IPv6)
		p.SrcIP = p.IPv6.SrcIP.String()
		p.DstIP = p.IPv6.DstIP.String()
	}

	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		return nil, errors.New("invalid TCP package")
	}
	p.TCP = tcpLayer.(*layers.TCP)
	if ipSet.Has(p.SrcIP) && int(p.TCP.SrcPort) == port {
		p.Direction = DirOutcoming
	} else if ipSet.Has(p.DstIP) && int(p.TCP.DstPort) == port {
		p.Direction = DirIncoming
	} else {
		p.Direction = DirUnknown
	}
	return &p, nil
}

func (p *NetPkg) TCPFlags() []string {
	flags := make([]string, 0)
	if p.TCP.FIN {
		flags = append(flags, "FIN")
	}
	if p.TCP.SYN {
		flags = append(flags, "SYN")
	}
	if p.TCP.RST {
		flags = append(flags, "RST")
	}
	if p.TCP.PSH {
		flags = append(flags, "PSH")
	}
	if p.TCP.ACK {
		flags = append(flags, "ACK")
	}
	if p.TCP.URG {
		flags = append(flags, "URG")
	}
	return flags
}

func (p *NetPkg) DirectConn() DirectConn {
	var c DirectConn
	c.SrcAddr.IP = p.SrcIP
	c.DstAddr.IP = p.DstIP
	c.SrcAddr.Port = uint32(p.TCP.SrcPort)
	c.DstAddr.Port = uint32(p.TCP.DstPort)
	return c
}
