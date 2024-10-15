package bfd

import (
	"context"
	"fmt"
	"strings"

	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

type CallbackFunc func(ipAddr string, preState, currState int)

type Control struct {
	LocalIP  string
	Family   int
	RxQueue  chan *RxData
	sessions []*Session
}

func NewControl(ctx context.Context, localIP string, family int) *Control {
	c := &Control{
		LocalIP: localIP,
		Family:  family,
		RxQueue: make(chan *RxData),
	}
	c.Run(ctx)

	return c
}

// AddSession adds a peer that need to be detected
func (c *Control) AddSession(
	remoteIP string,
	passive bool,
	rxInterval,
	txInterval,
	detectMult int,
	fn CallbackFunc,
	chBFDDone chan struct{},
) {
	s := NewSession(
		c.LocalIP,
		remoteIP,
		c.Family,
		passive,
		rxInterval*1000,
		txInterval*1000,
		detectMult,
		fn,
		chBFDDone,
	)

	c.sessions = append(c.sessions, s)

	logger.Info("bfd session created successfully", "remote ip", remoteIP)
}

// DelSession deletes an instance that needs to be detected
func (c *Control) DelSession(remoteIP string) error {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("failed to delete bfd session", "remote ip", remoteIP, "error", err)

			return
		}
	}()

	for i, session := range c.sessions {
		if session.RemoteIP == remoteIP {
			session.clientQuit <- true

			c.sessions = append(c.sessions[:i], c.sessions[i+1:]...)
		}
	}

	return nil
}

func (c *Control) Run(ctx context.Context) {
	go c.backgroundRun(ctx)

	logger.Debug("bfd process running in the background")
}

func (c *Control) backgroundRun(ctx context.Context) {
	c.initServer(ctx)

	logger.Debug("bfd daemon configured successfully")

	for rxData := range c.RxQueue {
		c.processPackets(rxData)
	}
}

func (c *Control) initServer(ctx context.Context) {
	addr := fmt.Sprintf("%s:%d", c.LocalIP, ControlPort)

	s := NewServer(addr, c.Family, c.RxQueue)

	go func() {
		if err := s.Start(ctx); err != nil {
			logger.Error("failed to start udp listening server", "error", err)
		}
	}()

	logger.Debug("udp server started successfully", "local address", c.LocalIP, "port", ControlPort)
}

// processPackets processes received packets
func (c *Control) processPackets(rxdt *RxData) {
	logger.Debug("received a new bfd packet", "remote ip", rxdt.Addr)

	bfdPacket := rxdt.Data

	if bfdPacket.YourDiscriminator > 0 {
		for _, session := range c.sessions {
			if session.LocalDiscr == bfdPacket.YourDiscriminator {
				session.RxPacket(bfdPacket)

				return
			}
		}
	} else {
		for _, session := range c.sessions {
			addrIP := strings.Split(rxdt.Addr, ":")[0]
			if session.RemoteIP == addrIP {
				session.RxPacket(bfdPacket)

				return
			}
		}
	}
}
