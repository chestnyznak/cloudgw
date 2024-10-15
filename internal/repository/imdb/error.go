package imdb

import (
	"errors"
)

var (
	ErrNoVPPFIPFoundInStorage       = errors.New("no vpp floating ip found in storage")
	ErrNoBGPPeerFoundInStorage      = errors.New("no bgp peer found in storage")
	ErrNoBGPPeersFoundInStorage     = errors.New("no bgp peers found in storage")
	ErrNoVPPVRFsFoundInStorage      = errors.New("no vpp vrfs found in storage")
	ErrNoVPPUDPTunnelFoundInStorage = errors.New("no vpp udp tunnel found in storage")
)
