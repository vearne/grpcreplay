package main

import (
	"fmt"
	"strings"
)

func main() {

	fmt.Println(strings.HasPrefix("192.168.8.128aa", "192.168.8.128"))
	fmt.Println(strings.HasPrefix("192.168.8.218 aaaa", "192.168.8.128"))
}
