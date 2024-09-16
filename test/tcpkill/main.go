package main

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"log"
	"math/rand"
	"net"
	"syscall"
	"time"
)

const (
	FIN = 1 << iota
	SYN
	RST
	PSH
	ACK
	URG
)

type ConnEntry struct {
	srcIP   string
	dstIP   string
	srcPort layers.TCPPort
	dstPort layers.TCPPort
}

var (
	device string = "en0"
	//device      string = "lo0"
	snapshotLen int32 = 1024
	promiscuous bool  = false
	err         error
	timeout     time.Duration = 5 * time.Second
	handle      *pcap.Handle
)

func main() {
	port := 9000

	go func() {
		killAllConn(port)
	}()
	time.Sleep(5 * time.Second)
	itemList := getAllConn(port)
	for _, item := range itemList {
		err := SendSYN(IPtoByte(item.srcIP), IPtoByte(item.dstIP), item.srcPort,
			item.dstPort, uint32(rand.Int31n(100)))
		if err != nil {

		}
	}
	time.Sleep(5 * time.Minute)
}

func getAllConn(localPort int) []*ConnEntry {
	result := make([]*ConnEntry, 0)
	item1 := ConnEntry{srcIP: "192.168.8.223", srcPort: 52456, dstIP: "192.168.8.218", dstPort: 9000}
	result = append(result, &item1)
	item2 := ConnEntry{srcIP: "192.168.8.218", srcPort: 9000, dstIP: "192.168.8.223", dstPort: 52456}
	result = append(result, &item2)
	return result
}

func killAllConn(port int) {
	// Open device
	handle, err = pcap.OpenLive(device, snapshotLen, promiscuous, timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()
	// Set filter
	var filter string = fmt.Sprintf("tcp and port %v", port)
	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Only capturing TCP port 8080 packets.")

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	var (
		srcIP   net.IP
		dstIP   net.IP
		srcPort layers.TCPPort
		dstPort layers.TCPPort
		ACKFlag bool
		seq     uint32
	)
	for packet := range packetSource.Packets() {
		ACKFlag = false
		//packet.Data()
		for i, layer := range packet.Layers() {
			fmt.Println(i, ":", layer.LayerType().String())
		}
		fmt.Println("got package")
		ethLayer := packet.Layer(layers.LayerTypeEthernet)
		if ethLayer == nil {
			log.Println("ethLayer == nil")
			continue
		}
		eth := ethLayer.(*layers.Ethernet)
		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		if ipLayer == nil {
			log.Println("ipLayer == nil")
			continue
		}
		ip, _ := ipLayer.(*layers.IPv4)
		srcIP = ip.SrcIP
		dstIP = ip.DstIP

		// Let's see if the packet is TCP
		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if tcpLayer != nil {
			tcp, _ := tcpLayer.(*layers.TCP)
			srcPort = tcp.SrcPort
			dstPort = tcp.DstPort
			if int(tcp.DstPort) == port && tcp.ACK {
				ACKFlag = true
				seq = tcp.Ack
				if ACKFlag {
					//var i uint32
					for i := 0; i < 3; i++ {
						seq += uint32(i) * uint32(tcp.Window)
						err = SendRST(eth.DstMAC, eth.SrcMAC, dstIP, srcIP, dstPort, srcPort, seq, handle)
						if err != nil {
							log.Println("SendNetPkg", err)
						}
					}
				}
			}
		}

	}
}

func SendSYN(srcIp, dstIp net.IP, srcPort, dstPort layers.TCPPort, seq uint32) error {
	log.Printf("send %v:%v > %v:%v [SYN] seq %v", srcIp.String(), srcPort.String(),
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
		fmt.Printf("Socket creation error: %v\n", err)
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
		fmt.Printf("Sendto error: %v\n", err)
		return err
	}
	fmt.Println("SYN packet sent")
	return nil
}

func SendRST(srcMac, dstMac net.HardwareAddr, srcIp, dstIp net.IP, srcPort, dstPort layers.TCPPort,
	seq uint32, handle *pcap.Handle) error {
	log.Printf("send %v:%v > %v:%v [RST] seq %v", srcIp.String(), srcPort.String(), dstIp.String(), dstPort.String(), seq)

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

func IPtoByte(ipStr string) []byte {
	return net.ParseIP(ipStr).To4()
}
