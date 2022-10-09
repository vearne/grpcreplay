package http2

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/vearne/grpcreplay/consts"
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
	IP        *layers.IPv4
	TCP       *layers.TCP
	Direction Dir
}

func ProcessPacket(packet gopacket.Packet, ipSet *util.StringSet, port int) (*NetPkg, error) {
	var p NetPkg
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return nil, consts.ErrProcessPacket
	}
	p.IP = ipLayer.(*layers.IPv4)
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		return nil, consts.ErrProcessPacket
	}
	p.TCP = tcpLayer.(*layers.TCP)
	if ipSet.Has(p.IP.DstIP.String()) && int(p.TCP.SrcPort) == port {
		p.Direction = DirOutcoming
	} else {
		p.Direction = DirIncoming
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
	c.SrcAddr.IP = p.IP.SrcIP.String()
	c.DstAddr.IP = p.IP.DstIP.String()
	c.SrcAddr.Port = uint32(p.TCP.SrcPort)
	c.DstAddr.Port = uint32(p.TCP.DstPort)
	return c
}
