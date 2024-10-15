package v1

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	bgpapi "github.com/osrg/gobgp/v3/api"
	vppapi "go.fd.io/govpp/api"
	"go.fd.io/govpp/binapi/ip"

	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
	"git.crptech.ru/cloud/cloudgw/internal/repository/vpp"
)

type SummaryStatus struct {
	BGPPeerTotal         int      `json:"BGPPeerTotal"`
	BGPPeerActive        int      `json:"BGPPeerActive"`
	MemVPPFIPRouteTotal  int      `json:"MemVPPFIPRouteTotal"`
	MemVPPUDPTunnelTotal int      `json:"MemVPPUDPTunnelTotal"`
	VPPFIPRouteTotal     int      `json:"VPPFIPRouteTotal"`
	VPPUDPTunnelTotal    int      `json:"VPPUDPTunnelTotal"`
	Errors               []string `json:"Errors,omitempty"`
}

func Summary(storage imdb.Storage, stream vppapi.Stream) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var summaryStatus SummaryStatus

		// bgp peers (total and active)

		bgpPeers := storage.BGPPeerStorage.GetBGPPeers()

		if len(bgpPeers) == 0 {
			summaryStatus.Errors = append(summaryStatus.Errors, fmt.Errorf("no bgp peers found").Error())
		}

		summaryStatus.BGPPeerTotal = len(bgpPeers)
		summaryStatus.BGPPeerActive = 0

		for _, peer := range bgpPeers {
			if peer.BGPPeerState == bgpapi.PeerState_ESTABLISHED {
				summaryStatus.BGPPeerActive++
			}
		}

		// in-memory vpp floating ip routes

		fips := storage.VPPFIPRouteStorage.GetFIPRoutes()

		summaryStatus.MemVPPFIPRouteTotal = len(fips)

		// in-memory vpp udp tunnels

		tunnels := storage.VPPUDPTunnelStorage.GetUDPTunnels()

		summaryStatus.MemVPPUDPTunnelTotal = len(tunnels)

		// vpp floating ip routes

		summaryStatus.VPPFIPRouteTotal = 0

		vppVRFs := storage.VPPVRFStorage.GetVRFs()

		for i := 1; i < len(vppVRFs); i++ { // id=0 as vrf where are no floating ips
			_, fips, err := vpp.CountRoutesPerTable(stream, ip.IPTable{TableID: uint32(i)})
			if err != nil {
				summaryStatus.Errors = append(summaryStatus.Errors, err.Error())
			}

			summaryStatus.VPPFIPRouteTotal += int(fips)
		}

		// vpp udp tunnels

		udpCount, err := vpp.CountUDPTunnels(stream)
		if err != nil {
			summaryStatus.Errors = append(summaryStatus.Errors, err.Error())
		}

		summaryStatus.VPPUDPTunnelTotal = int(udpCount)

		c.JSON(http.StatusOK, gin.H{"summary": summaryStatus})
	}

	return fn
}

func BGPPeers(bgpPeerStorage *imdb.BGPPeerStorage) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		bgpPeers := bgpPeerStorage.GetBGPPeers()
		if len(bgpPeers) == 0 {
			c.JSON(http.StatusOK, gin.H{"error": fmt.Errorf("no bgp peers found").Error()})

			return
		}

		c.JSON(http.StatusOK, gin.H{"bgp peers": bgpPeers})
	}

	return fn
}

func BGPVRFs(bgpVRFStorage *imdb.BGPVRFStorage) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		bgpVRFs := bgpVRFStorage.GetVRFs()
		c.JSON(http.StatusOK, gin.H{"bgp vrfs": bgpVRFs})
	}

	return fn
}

func VPPVRFs(vppVRFStorage *imdb.VPPVRFStorage) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		vppVRFs := vppVRFStorage.GetVRFs()
		c.JSON(http.StatusOK, gin.H{"vpp vrfs": vppVRFs})
	}

	return fn
}

func VPPFIPRoutes(vppFIPRouteStorage *imdb.VPPFIPRouteStorage) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		fips := vppFIPRouteStorage.GetFIPRoutes()
		c.JSON(http.StatusOK, gin.H{"vpp fips": fips})
	}

	return fn
}

func UDPTunnels(vppUDPTunnelStorage *imdb.VPPUDPTunnelStorage) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		tunnels := vppUDPTunnelStorage.GetUDPTunnels()
		c.JSON(http.StatusOK, gin.H{"vpp udp tunnels": tunnels})
	}

	return fn
}
