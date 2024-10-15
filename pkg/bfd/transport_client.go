package bfd

import (
	"fmt"
	"math/rand/v2"
	"net"

	"github.com/google/gopacket/layers"
	"golang.org/x/net/ipv4"

	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

const (
	ControlPort = 3784
)

type RxData struct {
	Data *layers.BFD
	Addr string
}

// RandInt returns random int between min and max
func RandInt(min, max int) int {
	if min >= max || min == 0 || max == 0 {
		return max
	}

	return rand.IntN(max-min) + min
}

// NewClient creates UDP client connection
func NewClient(localAddr, remoteAddr string) (*net.UDPConn, error) {
	var (
		conn          *net.UDPConn
		err           error
		localUDPAddr  *net.UDPAddr
		remoteUDPAddr *net.UDPAddr
	)

	srcPort := RandInt(SourcePortMin, SourcePortMax)

	addr := fmt.Sprintf("%s:%d", localAddr, srcPort)

	serAddr := fmt.Sprintf("%s:%d", remoteAddr, ControlPort)

	localUDPAddr, _ = net.ResolveUDPAddr("udp4", addr)

	remoteUDPAddr, _ = net.ResolveUDPAddr("udp4", serAddr)

	conn, err = net.DialUDP("udp", localUDPAddr, remoteUDPAddr)
	if err != nil {
		return conn, err
	}

	// change ip attribute for outgoing packets (https://pkg.go.dev/golang.org/x/net/ipv4)
	if err = ipv4.NewConn(conn).SetTOS(0xC0); err != nil { // CS6 for control plane packets
		logger.Error("failed setting tos=cs6 for bfd packet", "error", err)
	}

	if err = ipv4.NewConn(conn).SetTTL(255); err != nil {
		logger.Error("failed setting ttl=255 for bfd packet", "error", err)
	}

	return conn, nil
}
