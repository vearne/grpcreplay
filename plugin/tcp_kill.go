package plugin

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	FIN = 1 << iota
	SYN
	RST
	PSH
	ACK
	URG
)

// srcAddr, dstAddr: TCP地址
func sendFakePkg(seq uint32, srcAddr string, srcPort uint16,
	dstAddr string, dstPort uint16, flag uint8) {

	//seq += uint32(rand.Intn(1000) + 200)

	var (
		msg       string
		psdheader PsdHeader
		tcpheader TCPHeader
		buffer    bytes.Buffer
	)

	/*填充PSD首部*/
	psdheader.SrcAddr = inet_addr(srcAddr)
	psdheader.DstAddr = inet_addr(dstAddr)
	psdheader.Zero = 0
	psdheader.ProtoType = syscall.IPPROTO_TCP
	psdheader.TcpLength = uint16(unsafe.Sizeof(TCPHeader{})) + uint16(len(msg))

	/*填充TCP首部*/
	tcpheader.SrcPort = srcPort
	tcpheader.DstPort = dstPort
	tcpheader.SeqNum = seq
	tcpheader.AckNum = 0
	tcpheader.Offset = uint8(uint16(unsafe.Sizeof(TCPHeader{}))/4) << 4
	tcpheader.Flag |= flag
	tcpheader.Window = 60000
	tcpheader.Checksum = 0

	// 只是为了计算Checksum
	binary.Write(&buffer, binary.BigEndian, psdheader)
	binary.Write(&buffer, binary.BigEndian, tcpheader)
	tcpheader.Checksum = CheckSum(buffer.Bytes())
	buffer.Reset()

	/*接下来清空buffer，填充实际要发送的部分*/
	binary.Write(&buffer, binary.BigEndian, tcpheader)
	binary.Write(&buffer, binary.BigEndian, msg)

	/*下面的操作都是raw socket操作，大家都看得懂*/
	var (
		sockfd int
		addr   syscall.SockaddrInet4
		err    error
	)
	if sockfd, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP); err != nil {
		fmt.Println("Socket() error: ", err.Error())
		return
	}
	defer syscall.Shutdown(sockfd, syscall.SHUT_RDWR)
	addr.Addr = IPtoByte(dstAddr)
	addr.Port = int(dstPort)

	fmt.Println("seq:", seq)
	fmt.Printf("send %v from %v:%v\n", flagStr(flag), srcAddr, srcPort)
	fmt.Printf("send %v to %v:%v\n", flagStr(flag), dstAddr, dstPort)
	if err = syscall.Sendto(sockfd, buffer.Bytes(), 0, &addr); err != nil {
		fmt.Println("Sendto() error: ", err.Error())
		return
	}
	fmt.Println("Send success!")
}

type TCPHeader struct {
	SrcPort   uint16
	DstPort   uint16
	SeqNum    uint32
	AckNum    uint32
	Offset    uint8
	Flag      uint8
	Window    uint16
	Checksum  uint16
	UrgentPtr uint16
}

type PsdHeader struct {
	SrcAddr   uint32
	DstAddr   uint32
	Zero      uint8
	ProtoType uint8
	TcpLength uint16
}

func flagStr(flag uint8) string {
	m := make(map[uint8]string)
	m[FIN] = "FIN"
	m[SYN] = "SYN"
	m[RST] = "RST"
	m[PSH] = "PSH"
	m[ACK] = "ACK"
	m[URG] = "URG"

	tmpList := make([]string, 0)
	for _, f := range []uint8{FIN, SYN, RST, PSH, ACK, URG} {
		if flag&f > 0 {
			tmpList = append(tmpList, m[f])
		}
	}
	return strings.Join(tmpList, "|")
}

func IPtoByte(ipStr string) [4]byte {
	var addr [4]byte
	for i, bt := range net.ParseIP(ipStr).To4() {
		addr[i] = bt
	}
	return addr
}

func inet_addr(ipaddr string) uint32 {
	var (
		segments []string = strings.Split(ipaddr, ".")
		ip       [4]uint64
		ret      uint64
	)
	for i := 0; i < 4; i++ {
		ip[i], _ = strconv.ParseUint(segments[i], 10, 64)
	}
	ret = ip[3]<<24 + ip[2]<<16 + ip[1]<<8 + ip[0]
	return uint32(ret)
}

func CheckSum(data []byte) uint16 {
	var (
		sum    uint32
		length int = len(data)
		index  int
	)
	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		index += 2
		length -= 2
	}
	if length > 0 {
		sum += uint32(data[index])
	}
	sum += (sum >> 16)

	return uint16(^sum)
}
