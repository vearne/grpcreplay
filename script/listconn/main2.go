package main

import (
	"log"
	"net"
)

func main() {
	itfStatList, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	for _, itf := range itfStatList {
		log.Println(itf.Network())
		log.Println(itf.String())
	}
}
