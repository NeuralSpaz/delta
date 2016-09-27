package main

import (
	"fmt"
	"log"
	"net"

	"github.com/NeuralSpaz/delta"
)

const xportAddress string = "192.168.1.14:10001"
const inverterAddress = 0x01

func main() {
	fmt.Println("Starting Delta Solar Monitor")
	conn, _ := net.Dial("tcp", xportAddress)
	solar := delta.New(conn, inverterAddress)

	for {
		_, v, err := solar.ReadFloat(uint16(str1volts), 100, "string1volt")
		if err != nil {
			log.Println(err)
		}
		log.Println(v)
	}
}

type register uint16

const (
	str1volts    register = 0x1001
	str1amps     register = 0x1002
	str1power    register = 0x1003
	str2volts    register = 0x1004
	str2amps     register = 0x1005
	str2power    register = 0x1006
	totalwatthrs register = 0x1303
	totalpower   register = 0x1309
)
