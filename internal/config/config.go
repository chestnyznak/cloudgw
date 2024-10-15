package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct { // https://yaml2go.prasadg.dev/
	Logging      Logging      `yaml:"Logging" env-required:"true"`
	HTTP         HTTP         `yaml:"HTTP" env-required:"true"`
	Pyroscope    Pyroscope    `yaml:"Pyroscope"`
	TFController TFController `yaml:"TFController" env-required:"true"`
	GoBGP        GoBGP        `yaml:"GoBGP" env-required:"true"`
	VPP          VPP          `yaml:"VPP" env-required:"true"`
	VRF          []VRF        `yaml:"VRF" env-required:"true"`
}

type Logging struct {
	Level  string `yaml:"Level" env-required:"true"`
	Format string `yaml:"Format" env-required:"true"`
	Output string `yaml:"Output" env-required:"true"`
	Source bool   `yaml:"Source"`
}

type HTTP struct {
	Enable  bool   `yaml:"Enable" env-default:"false"`
	Address string `yaml:"Address"`
}

type Pyroscope struct {
	Enable bool   `yaml:"Enable" env-default:"false"`
	URL    string `yaml:"URL"`
}

type TFController struct {
	BGPPeerASN   uint32   `yaml:"BGPPeerASN" env-required:"true"`
	BGPTTL       uint32   `yaml:"BGPTTL" env-required:"true"`
	BGPKeepAlive uint64   `yaml:"BGPKeepAlive" env-required:"true"`
	BGPHoldTimer uint64   `yaml:"BGPHoldTimer" env-required:"true"`
	Address      []string `yaml:"Address" env-required:"true"`
}

type GoBGP struct {
	GRPCListenAddress     string `yaml:"GRPCListenAddress" env-required:"true"`
	BGPLocalASN           uint32 `yaml:"BGPLocalASN" env-required:"true"`
	BGPLocalPort          int32  `yaml:"BGPLocalPort" env-default:"179"`
	RID                   string `yaml:"RID" env-required:"true"`
	MetricPollingInterval int    `yaml:"MetricPollingInterval" env-default:"3"`
}

type VPP struct {
	BinAPISock             string `yaml:"BinAPISock" env-default:"/home/enikolaev/vpp_api.sock"`
	MainInterfaceID        uint32 `yaml:"MainInterfaceID" env-required:"true"`
	TunLocalIP             string `yaml:"TunLocalIP" env-required:"true"`
	TunDefaultGW           string `yaml:"TunDefaultGW" env-required:"true"`
	InterfaceMonitorEnable bool   `yaml:"InterfaceMonitorEnable"`
	MetricPollingInterval  int    `yaml:"MetricPollingInterval"`
}

type VRF struct {
	FIPPrefixes   []string `yaml:"FIPPrefixes" env-required:"true"`
	VRFName       string   `yaml:"VRFName" env-required:"true"`
	VRFID         uint32   `yaml:"VRFID" env-required:"true"`
	LocalIP       string   `yaml:"LocalIP" env-required:"true"`
	VLANID        uint32   `yaml:"VLANID" env-required:"true"`
	BGPPeerIP     string   `yaml:"BGPPeerIP" env-required:"true"`
	BGPPeerASN    uint32   `yaml:"BGPPeerASN" env-required:"true"`
	BGPTTL        uint32   `yaml:"BGPTTL" env-required:"true"`
	BGPKeepAlive  uint64   `yaml:"BGPKeepAlive" env-required:"true"`
	BGPHoldTimer  uint64   `yaml:"BGPHoldTimer" env-required:"true"`
	BGPPassword   string   `yaml:"BGPPassword"`
	BFDEnable     bool     `yaml:"BFDEnable"`
	BFDLocalIP    string   `yaml:"BFDLocalIP"`
	BFDTxRate     int      `yaml:"BFDTxRate"`
	BFDRxMin      int      `yaml:"BFDRxMin"`
	BFDMultiplier int      `yaml:"BFDMultiplier"`
}

func ParseConfig(configPath string) (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
