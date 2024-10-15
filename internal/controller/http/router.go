package rest

import (
	"net/http"

	"github.com/alecthomas/kingpin/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/node_exporter/collector"
	vppapi "go.fd.io/govpp/api"

	controller "git.crptech.ru/cloud/cloudgw/internal/controller/http/v1"
	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
	"git.crptech.ru/cloud/cloudgw/pkg/exporter/gobgpexporter"
	"git.crptech.ru/cloud/cloudgw/pkg/exporter/vppexporter"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

func NewRouter(appStorage *imdb.Storage, stream vppapi.Stream) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	engine.Use(gin.Recovery())

	flag.AddFlags(kingpin.CommandLine, &promlog.Config{})
	kingpin.Parse()

	nodeExporter, err := collector.NewNodeCollector(log.NewNopLogger())
	if err != nil {
		logger.Fatal(err.Error())
	}

	vppExporter := vppexporter.NewCloudgwExporter()

	goBGPExporter := gobgpexporter.NewCloudgwExporter()

	prometheus.MustRegister(vppExporter, goBGPExporter, nodeExporter)

	engine.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "OK"}) })
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	engine.GET("/summary", controller.Summary(*appStorage, stream))
	engine.GET("/bgp/vrfs", controller.BGPVRFs(appStorage.BGPVRFStorage))
	engine.GET("/bgp/peers", controller.BGPPeers(appStorage.BGPPeerStorage))
	engine.GET("/vpp/vrfs", controller.VPPVRFs(appStorage.VPPVRFStorage))
	engine.GET("/vpp/fips", controller.VPPFIPRoutes(appStorage.VPPFIPRouteStorage))
	engine.GET("/vpp/tunnels", controller.UDPTunnels(appStorage.VPPUDPTunnelStorage))

	return engine
}
