package model

import (
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

type Dir int

type NetPkg struct {
	IP        *layers.IPv4
	TCP       *layers.TCP
	Direction Dir
}

func ProcessPacket(packet gopacket.Packet, ipSet *util.StringSet) (*NetPkg, error) {
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
	if ipSet.Has(p.IP.DstIP.String()) {
		p.Direction = consts.DirIncoming
	} else {
		p.Direction = consts.DirOutcoming
	}
	return &p, nil
}

func (p *NetPkg) DirectConn() DirectConn {
	var c DirectConn
	c.SrcAddr.IP = p.IP.SrcIP.String()
	c.DstAddr.IP = p.IP.DstIP.String()
	c.SrcAddr.Port = uint32(p.TCP.SrcPort)
	c.DstAddr.Port = uint32(p.TCP.DstPort)
	return c
}
