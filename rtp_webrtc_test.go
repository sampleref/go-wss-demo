package main

import (
	"fmt"
	"net"
	"testing"
)

func TestRTPListener(t *testing.T) {

	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9898})
	if err != nil {
		panic(err)
	}

	inboundRTPPacket := make([]byte, 1600) // UDP MTU
	for {
		n, _, err := listener.ReadFrom(inboundRTPPacket)
		if err != nil {
			panic(fmt.Sprintf("error during read: %s", err))
		}
		fmt.Printf("Reading packet %d\n", n)
	}
}
