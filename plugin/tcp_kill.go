package plugin

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	slog "github.com/vearne/simplelog"
	"net"
	"syscall"
)

func SendSYN(srcIp, dstIp net.IP, srcPort, dstPort layers.TCPPort, seq uint32) error {
	slog.Info("send %v:%v > %v:%v [SYN] seq %v", srcIp.String(), srcPort.String(),
		dstIp.String(), dstPort.String(), seq)

	iPv4 := layers.IPv4{
		SrcIP:    srcIp,
		DstIP:    dstIp,
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolTCP,
	}

	tcp := layers.TCP{
		SrcPort: srcPort,
		DstPort: dstPort,
		Seq:     seq,
		SYN:     true,
	}

	if err := tcp.SetNetworkLayerForChecksum(&iPv4); err != nil {
		return err
	}

	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	if err := gopacket.SerializeLayers(buffer, options, &tcp); err != nil {
		return err
	}

	// 创建一个原始套接字
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
	if err != nil {
		slog.Error("Socket creation error: %v", err)
		return err
	}
	defer syscall.Close(fd)
	// 设置目标地址
	addr := syscall.SockaddrInet4{
		Port: int(dstPort), // 目标端口
	}
	copy(addr.Addr[:], dstIp)
	// 发送 SYN 包
	err = syscall.Sendto(fd, buffer.Bytes(), 0, &addr)
	if err != nil {
		slog.Error("Sendto error: %v", err)
		return err
	}
	slog.Info("SYN packet sent")
	return nil
}

func SendRST(srcMac, dstMac net.HardwareAddr, srcIp, dstIp net.IP, srcPort, dstPort layers.TCPPort,
	seq uint32, handle *pcap.Handle) error {
	slog.Info("send %v:%v > %v:%v [RST] seq %v", srcIp.String(), srcPort.String(),
		dstIp.String(), dstPort.String(), seq)

	eth := layers.Ethernet{
		SrcMAC:       srcMac,
		DstMAC:       dstMac,
		EthernetType: layers.EthernetTypeIPv4,
	}

	iPv4 := layers.IPv4{
		SrcIP:    srcIp,
		DstIP:    dstIp,
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolTCP,
	}

	tcp := layers.TCP{
		SrcPort: srcPort,
		DstPort: dstPort,
		Seq:     seq,
		RST:     true,
	}

	if err := tcp.SetNetworkLayerForChecksum(&iPv4); err != nil {
		return err
	}

	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	if err := gopacket.SerializeLayers(buffer, options, &eth, &iPv4, &tcp); err != nil {
		return err
	}

	err := handle.WritePacketData(buffer.Bytes())
	if err != nil {
		return err
	}
	return nil
}
