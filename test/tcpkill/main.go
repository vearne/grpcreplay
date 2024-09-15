package main

import (
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"log"
	"net"
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
		SendSYN(IPtoByte(item.srcIP), IPtoByte(item.dstIP), item.srcPort, item.dstPort, 10, nil)
	}
	time.Sleep(5 * time.Minute)
}

func getAllConn(localPort int) []*ConnEntry {
	result := make([]*ConnEntry, 0)
	item1 := ConnEntry{srcIP: "192.168.8.233", srcPort: 34932, dstIP: "192.168.8.218", dstPort: 9000}
	result = append(result, &item1)
	//item2 := ConnEntry{srcIP: "127.0.0.1", srcPort: 63842, dstIP: "127.0.0.1", dstPort: 8080}
	//result = append(result, &item2)
	return result
}

func fmtAddr(ipAddr net.IP, port int) string {
	return fmt.Sprintf("%v:%v", ipAddr.String(), port)
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
		fmt.Println("got package")

		ethLayer := packet.Layer(layers.LayerTypeEthernet)
		if ethLayer == nil {
			continue
		}
		eth := ethLayer.(*layers.Ethernet)
		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		if ipLayer == nil {
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

func SendSYN(srcIp, dstIp net.IP, srcPort, dstPort layers.TCPPort, seq uint32, handle *pcap.Handle) error {
	log.Printf("send %v:%v > %v:%v [SYN] seq %v", srcIp.String(), srcPort.String(), dstIp.String(), dstPort.String(), seq)
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

	err := handle.WritePacketData(buffer.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func SendRST(srcMac, dstMac net.HardwareAddr, srcIp, dstIp net.IP, srcPort, dstPort layers.TCPPort, seq uint32, handle *pcap.Handle) error {
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
