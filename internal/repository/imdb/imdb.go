package imdb

type Storage struct {
	*BGPPeerStorage
	*BGPVRFStorage
	*VPPVRFStorage
	*VPPFIPRouteStorage
	*VPPUDPTunnelStorage
}
