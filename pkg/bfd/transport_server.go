package bfd

import (
	"context"
	"net"

	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

type Server struct {
	LocalAddr string
	listener  *net.UDPConn // udp conn
	Family    int
	RxQueue   chan *RxData
}

func NewServer(localAddr string, addressFamily int, rx chan *RxData) *Server {
	return &Server{
		LocalAddr: localAddr,
		Family:    addressFamily,
		RxQueue:   rx,
	}
}

// Start starts UDP listening server
func (s *Server) Start(ctx context.Context) error {
	udpAddr, err := net.ResolveUDPAddr("udp4", s.LocalAddr)
	if err != nil {
		logger.Error("resolve udp address error", "error", err)

		return err
	}

	s.listener, err = net.ListenUDP("udp4", udpAddr)
	if err != nil {
		logger.Error("listen udp error", "error", err)

		return err
	}

	defer func() {
		_ = s.listener.Close()
	}()

	s.Loop(ctx)

	return nil
}

// Loop infinitely reads from UDP call packer decoding in loop
func (s *Server) Loop(ctx context.Context) {
	select {
	case <-ctx.Done():
		logger.Info("cancel context in bfd process detected, starting stopping listener and exit")

		return
	default:
		for {
			data := make([]byte, 1024)

			n, addr, err := s.listener.ReadFromUDP(data)
			if err != nil {
				logger.Error("read from udp socket error", "error", err)

				// continue
				break
			}

			bfdPk, err := DecodePacket(data[:n])
			if err != nil {
				// continue
				break
			}

			rxData := &RxData{Data: bfdPk, Addr: addr.String()}

			s.RxQueue <- rxData
		}
	}
}
