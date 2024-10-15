package bfd

import (
	"math/rand/v2"
	"net"
	"time"

	"github.com/google/gopacket/layers"

	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

const (
	defaultDetectMultiplier = 3
	SourcePortMin           = 49152
	SourcePortMax           = 65535

	VERSION = 1

	DesiredMinTXInterval      = 1000000
	ControlPlaneIndependent   = false
	DemandMode                = false
	MULTIPOINT                = false
	RequiredMinEchoRxInterval = 0
)

type Session struct {
	conn *net.UDPConn

	clientDown chan bool // true: down
	clientQuit chan bool // true: quit

	// callback state
	callFunc CallbackFunc

	// bfd session
	LocalIP    string
	RemoteIP   string
	Family     int
	Passive    bool
	RxInterval int
	TxInterval int

	// as per 6.8.1 State Variables
	State       layers.BFDState
	RemoteState layers.BFDState
	LocalDiscr  layers.BFDDiscriminator
	RemoteDiscr layers.BFDDiscriminator
	LocalDiag   layers.BFDDiagnostic

	desiredMinTxInterval  uint32
	requiredMinRxInterval uint32
	remoteMinRxInterval   uint32

	DemandMode       bool // asynchronous or demand mode
	RemoteDemandMode bool
	DetectMult       uint8 // the maximum number of packet failures, layers.BFDDetectMultiplier,
	AuthType         bool
	RcvAuthSeq       int
	XmitAuthSeq      int64
	AuthSeqKnown     bool

	// state variables beyond those defined in RFC 5880
	asyncTxInterval      uint32
	finalAsyncTxInterval uint32 // layers.BFDTimeInterval
	LastRxPacketTime     int64  // in order to store the time (in msec) of the last packet fetched
	asyncDetectTime      uint32 // msec
	finalAsyncDetectTime uint32 //
	PollSequence         bool
	remoteDetectMult     uint32 // layers.BFDDetectMultiplier
	remoteMinTxInterval  uint32 // layers.BFDTimeInterval
}

// NewSession creates a new BFD session
func NewSession(
	localIP string,
	remoteIP string,
	family int,
	passive bool,
	rxInterval,
	txInterval,
	detectMult int,
	f CallbackFunc,
	chBFDDone chan struct{},
) *Session {
	if detectMult <= 0 {
		detectMult = defaultDetectMultiplier
	}

	s := &Session{
		clientDown: make(chan bool),
		clientQuit: make(chan bool),
		callFunc:   f,
		LocalIP:    localIP,
		RemoteIP:   remoteIP,
		Family:     family,
		Passive:    passive,
		RxInterval: rxInterval,
		TxInterval: txInterval,
		//
		State:                 layers.BFDStateDown,
		RemoteState:           layers.BFDStateDown,
		LocalDiscr:            layers.BFDDiscriminator(rand.Int32()), // 32-bit
		RemoteDiscr:           0,
		LocalDiag:             layers.BFDDiagnosticNone,
		desiredMinTxInterval:  DesiredMinTXInterval,
		requiredMinRxInterval: uint32(rxInterval),
		remoteMinRxInterval:   1,
		DemandMode:            DemandMode,
		RemoteDemandMode:      false,
		DetectMult:            uint8(detectMult),
		AuthType:              false,
		RcvAuthSeq:            0,
		XmitAuthSeq:           int64(rand.Int32()), // 32-bit
		AuthSeqKnown:          false,
		//
		asyncTxInterval: DesiredMinTXInterval,
		PollSequence:    false,
	}

	s.setDesiredMinTxInterval(DesiredMinTXInterval)
	s.setRequiredMinRxInterval(uint32(rxInterval))

	go s.sessionLoop(chBFDDone)

	return s
}

var isBdfSessionStarted bool

// sessionLoop runs session loop
func (s *Session) sessionLoop(chBFDDone chan struct{}) {
	logger.Debug("setting up udp client", "remote ip", s.RemoteIP, "port", ControlPort)

	conn, err := NewClient(s.LocalIP, s.RemoteIP)
	if err != nil {
		logger.Error("loop new client close client chan", "error", err)

		s.clientDown <- true
	} else {
		s.conn = conn
	}

	var interval float64

	for {
		if s.DetectMult == 1 {
			// the interval must not exceed 90% and must be no less than 75%
			interval = float64(s.asyncTxInterval) * (rand.Float64()*0.75 + 0.15)
		} else {
			// the periodic transmission of BFD Control packets MUST be jittered on
			// a per-packet basis by up to 25%, that is, the interval MUST be
			// reduced by a random value of 0 to 25%
			interval = float64(s.asyncTxInterval) * (1 - (rand.Float64() * 0.25))
		}

		select {
		case <-s.clientDown:
			conn, err = NewClient(s.LocalIP, s.RemoteIP)
			if err != nil {
				s.closeConn()

				time.Sleep(time.Duration(int(interval)) * time.Microsecond)

				continue
			}

			s.conn = conn

			s.clientDown = make(chan bool)

			logger.Debug("new bfd client added for detection", "remote address", s.RemoteIP)

			// start timeout detection

			if !isBdfSessionStarted {
				go s.DetectFailure(chBFDDone)

				isBdfSessionStarted = true
			}

		case <-s.clientQuit:
			s.closeConn()

			return

		default:
			if !((s.RemoteDiscr == 0 && s.Passive) ||
				(s.remoteMinRxInterval == 0) ||
				(!s.PollSequence && (s.RemoteDemandMode && s.State == layers.BFDStateUp && s.RemoteState == layers.BFDStateUp))) {
				// decide whether the package should be sent actively
				s.TxPacket(false)
			}

			// speed (frequency) of packets sending
			time.Sleep(time.Duration(int(interval)) * time.Microsecond)
			// Start timeout detection
			if !isBdfSessionStarted {
				go s.DetectFailure(chBFDDone)

				isBdfSessionStarted = true
			}
		}
	}
}

// RxPacket handles received packets
func (s *Session) RxPacket(p *layers.BFD) {
	if p.AuthPresent && !s.AuthType {
		logger.Error("received bfd packet with authentication while no authentication is configured locally")

		return
	}

	if !p.AuthPresent && s.AuthType {
		logger.Error("received bfd packet without authentication while authentication is configured locally")

		return
	}

	if p.AuthPresent != s.AuthType {
		logger.Error("authenticated bfd packet received, not supported!")

		return
	}

	s.RemoteDiscr = p.MyDiscriminator
	s.RemoteState = p.State
	s.RemoteDemandMode = p.Demand

	s.setRemoteMinRxInterval(uint32(p.RequiredMinRxInterval))
	s.setRemoteDetectMult(uint32(p.DetectMultiplier))
	s.setRemoteMinTxInterval(uint32(p.DesiredMinTxInterval))

	if s.State == layers.BFDStateAdminDown {
		logger.Warn("received bfd packet while in admin_down state", "remote address", s.RemoteIP)

		return
	}

	if p.State == layers.BFDStateAdminDown {
		if s.State != layers.BFDStateDown {
			s.LocalDiag = layers.BFDDiagnosticNeighborSignalDown
			currState := int(s.State)

			go s.callFunc(s.RemoteIP, currState, int(layers.BFDStateDown))

			s.State = layers.BFDStateDown
			s.desiredMinTxInterval = DesiredMinTXInterval

			logger.Error("bfd remote signaled going admin_down", "remote address", s.RemoteIP)
		}
	} else {
		if s.State == layers.BFDStateDown {
			if p.State == layers.BFDStateDown {
				currState := int(s.State)

				go s.callFunc(s.RemoteIP, currState, int(layers.BFDStateInit))

				s.State = layers.BFDStateInit

				logger.Debug("bfd session going to init state", "remote address", s.RemoteIP)
			} else if p.State == layers.BFDStateInit {
				currState := int(s.State)

				go s.callFunc(s.RemoteIP, currState, int(layers.BFDStateUp))

				s.State = layers.BFDStateUp

				s.setDesiredMinTxInterval(uint32(s.TxInterval))

				logger.Debug("bfd session going to up state", "remote address", s.RemoteIP)
			}
		} else if s.State == layers.BFDStateInit {
			if p.State == layers.BFDStateInit || p.State == layers.BFDStateUp {
				currState := int(s.State)

				go s.callFunc(s.RemoteIP, currState, int(layers.BFDStateUp))

				s.State = layers.BFDStateUp

				s.setDesiredMinTxInterval(uint32(s.TxInterval))

				logger.Debug("bgd session going to up state", "remote address", s.RemoteIP)
			}
		} else {
			if p.State == layers.BFDStateDown {
				s.LocalDiag = layers.BFDDiagnosticNeighborSignalDown
				currState := int(s.State)

				go s.callFunc(s.RemoteIP, currState, int(layers.BFDStateDown))

				s.State = layers.BFDStateDown

				logger.Error("bfd remote signaled going down", "remote address", s.RemoteIP)
			}
		}
	}

	// If a BFD Control packet is received with the Poll (P) bit set to 1,
	// the receiving system MUST transmit a BFD Control packet with the Poll
	// (P) bit clear and the Final (F) bit set as soon as practicable (refer to RFC 5880)
	if p.Poll {
		logger.Debug("received bfd packet with poll (p) bit set, sending packet with final (f) bit set", "remote address", s.RemoteIP)

		s.TxPacket(true)
	}

	// when the system sending the Poll sequence receives a packet with Final, the Poll Sequence is terminated
	if p.Final {
		logger.Debug("received bfd packet with final (g) bit set from %s, ending poll sequence", "remote address", s.RemoteIP)
		s.PollSequence = false

		if s.finalAsyncTxInterval > 0 {
			logger.Debug(
				"increasing tx interval now that poll sequence has ended",
				"async tx interval", s.asyncTxInterval,
				"final async tx interval", s.finalAsyncTxInterval,
			)

			s.asyncTxInterval = s.finalAsyncTxInterval

			s.finalAsyncTxInterval = 0
		}

		if s.finalAsyncDetectTime > 0 {
			logger.Debug(""+
				"increasing detect time now that poll sequence has ended",
				"async detect time", s.asyncDetectTime,
				"final async detect time", s.finalAsyncDetectTime,
			)

			s.asyncDetectTime = s.finalAsyncDetectTime

			s.finalAsyncDetectTime = 0
		}
	}

	s.LastRxPacketTime = time.Now().UnixNano() / 1e6 // ms
}

// TxPacket creates a packet to be sent to target
func (s *Session) TxPacket(final bool) {
	logger.Debug("sending bfd packet", "local address", s.conn.LocalAddr().String())

	var demand bool

	if s.DemandMode && s.State == layers.BFDStateUp && s.RemoteState == layers.BFDStateUp {
		demand = true
	} else {
		demand = false
	}

	var poll bool

	if !final {
		poll = s.PollSequence
	} else {
		poll = false
	}

	var tmpAuth *layers.BFDAuthHeader

	if s.AuthType {
		tmpAuth = auth
	} else {
		tmpAuth = nil
	}

	txByte := EncodePacket(
		VERSION,
		s.LocalDiag,
		s.State,
		poll,
		final,
		ControlPlaneIndependent,
		s.AuthType,
		demand,
		MULTIPOINT,
		layers.BFDDetectMultiplier(s.DetectMult),
		s.LocalDiscr,
		s.RemoteDiscr,
		layers.BFDTimeInterval(s.desiredMinTxInterval),
		layers.BFDTimeInterval(s.requiredMinRxInterval),
		RequiredMinEchoRxInterval,
		tmpAuth)

	if _, err := s.conn.Write(txByte); err != nil {
		logger.Debug("failed to send to udp server, connection closed", "error", err.Error())

		s.closeConn()
	}
}

func (s *Session) restartTxPackets() {
	logger.Info("restart closed bfd client chan")

	s.closeConn()
}

func (s *Session) closeConn() {
	defer func() {
		if err := recover(); err != nil {
			return
		}
	}()

	_ = s.conn.Close()

	close(s.clientDown)
}

// calcDetectTime calculates the BFD Detection Time
func (s *Session) calcDetectTime(detectMult, rxInterval, txInterval uint32) (ret uint32) {
	if detectMult == 0 && rxInterval == 0 && txInterval == 0 {
		logger.Debug(
			"detection time calculation not possible values",
			"detect multiplier", detectMult,
			"rx interval", rxInterval,
			"tx interval", txInterval,
		)

		return 0
	}

	if rxInterval > txInterval {
		ret = detectMult * rxInterval
	} else {
		ret = detectMult * txInterval
	}

	logger.Debug("detection time calculated",
		"detect multiplier", detectMult,
		"rx interval", rxInterval,
		"tx interval", txInterval,
	)

	return
}

func (s *Session) setRemoteDetectMult(value uint32) {
	if value == s.remoteDetectMult {
		return
	}

	s.asyncDetectTime = s.calcDetectTime(value, s.requiredMinRxInterval, s.remoteDetectMult)

	s.remoteDetectMult = value
}

func (s *Session) setRemoteMinTxInterval(value uint32) {
	if value == s.remoteMinTxInterval {
		return
	}

	s.asyncDetectTime = s.calcDetectTime(s.remoteDetectMult, s.requiredMinRxInterval, value)

	s.remoteMinRxInterval = value
}

func (s *Session) setRemoteMinRxInterval(value uint32) {
	if value == s.remoteMinRxInterval {
		return
	}

	oldTxInterval := s.asyncTxInterval

	if value > s.desiredMinTxInterval {
		s.asyncTxInterval = value
	} else {
		s.asyncTxInterval = s.desiredMinTxInterval
	}

	if s.asyncTxInterval < oldTxInterval {
		s.restartTxPackets()
	}

	s.remoteMinRxInterval = value
}

func (s *Session) setRequiredMinRxInterval(value uint32) {
	if value == s.requiredMinRxInterval {
		return
	}

	detectTime := s.calcDetectTime(s.remoteDetectMult, value, s.remoteMinRxInterval)

	if value < s.requiredMinRxInterval && s.State == layers.BFDStateUp {
		s.finalAsyncDetectTime = detectTime
	} else {
		s.asyncDetectTime = detectTime
	}

	s.requiredMinRxInterval = value

	s.PollSequence = true
}

func (s *Session) setDesiredMinTxInterval(value uint32) {
	if value == s.desiredMinTxInterval {
		return
	}

	var txInterval uint32

	if value > s.remoteMinRxInterval {
		txInterval = value
	} else {
		txInterval = s.remoteMinRxInterval
	}

	if value > s.desiredMinTxInterval && s.State == layers.BFDStateUp {
		s.finalAsyncDetectTime = txInterval
	} else {
		s.asyncTxInterval = value
	}

	s.desiredMinTxInterval = value

	s.PollSequence = true
}

// DetectFailure detects BFP failures
func (s *Session) DetectFailure(chBFDDone chan struct{}) {
	const BFDDetectInterval = 150 // ms

	for {
		select {
		case <-s.clientDown:
			logger.Debug("bfd client down", "remote address", s.RemoteIP)

			return
		default:
			if !(s.DemandMode || s.asyncDetectTime == 0) {
				if (s.State == layers.BFDStateUp) &&
					((time.Now().UnixNano()/1e6 - s.LastRxPacketTime) > (int64(s.asyncDetectTime) / 1000)) {
					currState := int(s.State)

					go s.callFunc(s.RemoteIP, currState, int(layers.BFDStateDown))

					s.State = layers.BFDStateDown
					s.LocalDiag = layers.BFDDiagnosticTimeExpired

					s.setDesiredMinTxInterval(DesiredMinTXInterval)

					logger.Error("detected bfd peer going down", "remote address", s.RemoteIP)
					logger.Error(
						"time since last bfd packet exceed detect time;",
						"time since last packet received", time.Now().UnixNano()/1e6-s.LastRxPacketTime,
						"detect time", int64(s.asyncDetectTime)/1000,
					)

					logger.Info("closing the bfd channel to main function")

					close(chBFDDone)
				}
			}

			// waiting time
			time.Sleep(time.Millisecond * BFDDetectInterval)
		}
	}
}
