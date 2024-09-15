package plugin

import (
	"bytes"
	"encoding/binary"
	slog "github.com/vearne/simplelog"
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

func sendFakePkg(seq uint32, srcAddr string, srcPort uint16,
	dstAddr string, dstPort uint16, flag uint8) {
	var (
		msg       string
		ipheader  IPHeader
		tcpheader TCPHeader
		buffer    bytes.Buffer
	)

	/*Fill IP header*/
	ipheader.SrcAddr = inetAddr(srcAddr)
	ipheader.DstAddr = inetAddr(dstAddr)
	ipheader.Zero = 0
	ipheader.ProtoType = syscall.IPPROTO_TCP
	ipheader.TcpLength = uint16(unsafe.Sizeof(TCPHeader{})) + uint16(len(msg))

	/*Filling TCP header*/
	tcpheader.SrcPort = srcPort
	tcpheader.DstPort = dstPort
	tcpheader.SeqNum = seq
	tcpheader.AckNum = 0
	tcpheader.Offset = uint8(uint16(unsafe.Sizeof(TCPHeader{}))/4) << 4
	tcpheader.Flag |= flag
	tcpheader.Window = 60000
	tcpheader.Checksum = 0

	// calculate Checksum
	// nolint: errcheck
	binary.Write(&buffer, binary.BigEndian, ipheader)
	// nolint: errcheck
	binary.Write(&buffer, binary.BigEndian, tcpheader)
	tcpheader.Checksum = CheckSum(buffer.Bytes())

	// Next, clear the buffer and fill it with the part that is actually to be sent.
	buffer.Reset()
	//nolint:all
	binary.Write(&buffer, binary.BigEndian, tcpheader)
	//nolint:all
	binary.Write(&buffer, binary.BigEndian, msg)

	/*raw socket*/
	var (
		sockfd int
		addr   syscall.SockaddrInet4
		err    error
	)
	if sockfd, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP); err != nil {
		slog.Error("Socket() error: %v", err)
		return
	}
	// nolint: errcheck
	defer syscall.Shutdown(sockfd, syscall.SHUT_RDWR)
	addr.Addr = IPtoByte(dstAddr)
	addr.Port = int(dstPort)

	slog.Debug("send %v, %v:%v -> %v:%v", flagStr(flag),
		srcAddr, srcPort, dstAddr, dstPort)
	if err = syscall.Sendto(sockfd, buffer.Bytes(), 0, &addr); err != nil {
		slog.Error("Sendto() error: %v", err)
		return
	}
	slog.Debug("Send success!")
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

type IPHeader struct {
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

func inetAddr(ipaddr string) uint32 {
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
