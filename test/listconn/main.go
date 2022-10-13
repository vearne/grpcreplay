package main

import (
	"github.com/shirou/gopsutil/v3/net"
	"log"
)

func main() {
	itemList, err := net.Connections("tcp4")
	if err != nil {
		log.Println(err)
		return
	}
	for _, item := range itemList {
		if item.Laddr.Port == 8080 && item.Status == "ESTABLISHED" {
			log.Println(item.String())
		}
	}
	itfStatList, err := net.Interfaces()
	for _, itf := range itfStatList {
		log.Println(itf.Addrs)
		log.Println(itf.Name)
	}

}
