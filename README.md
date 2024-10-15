# Cloudgw. Gateway for the Tungsten Fabric

Simple gateway for Tungsten Fabric/Contrail/OpenSDN to terminate MPLS over UDP tunnels.
Can be used in as a gateway for Tungsten Fabric in Proof of Concept deployment or basis for production deployment. 

## Typical use case

Cloudgw acts as BGP-enable gateway and provides access to physical network for virtual machines in Tungsten Fabric.
Cloudgw will terminate MPLS over UDP tunnels inside Tungsten Fabric and pass traffic out of Tungsten Fabric using simple IP/BGP.
Each Network (aka VRF) with their Floating IP pools inside Tungsten Fabric will be mapped to physical network VRF using the Cloudgw.
Refer to [documentation](https://tungstenfabric.github.io/website/Tungsten-Fabric-Architecture.html#tf-bgp-gateway) for more details (non-Juniper device part).


```
[TF controllers] ---MP-BGP--- [Cloudgw] ---IP/BGP--- [Physical network]
                                  |
[vRouters] ------MPLSoverUDP------┘
```

## Features

- MPLS over UDP tunnels between the Cloudgw and vRouters
- VRF sandwich to physical network (using BGP)
- One dedicated interface for control plane
- One dedicated interface for VPP (10G or above)
- VPP as data plane engine (DPDK)
- Supports IPv4 only
- Support BFD to physical network BGP sessions
- YAML based configuration
- Graceful shutdown
- Prometheus metrics
- Basic statistics (VRF, BGP peers, routes) over HTTP (JSON)
- Run as systemd service

## Typical connection scheme

Refer to [docs/eng/cloudgw_scheme.adoc](docs/eng/cloudgw_scheme.adoc/)

## Requirements

- VPP version 23.10 or higher (GoVPP SDK uses stream mode API)
- Debian like OS Linux (e.g. Ubuntu 20.04 or higher), other OS is not tested but should work
- One dedicated interface for VPP (10G or above)
- One dedicated interface control plane (1G or above)

## Restrictions and limitations

- Does not support IPv6
- Does not support bonded interface
- Does not support NETCONF to interact with Tungsten Fabric
- Does not support on-fly change configuration (need to restart the cloudgw)
- Support only overlay scheme mentioned in [docs/eng/overlay.adoc](docs/eng/overlay.adoc)

## Startup

### As deb package

1. Install and configure VPP (refer to [docs/eng/vpp.adoc](docs/eng/vpp.adoc))
2. Generate deb package (refer to [Taskfile.yml](Taskfile.yml))
3. Install Cloudgw as deb package
4. Configure Cloudgw configuration file `/etc/cloudgw/config.conf` (refer to [docs/eng/cloudgw_config.adoc](docs/eng/cloudgw_config.adoc))
5. Run the cloudgw as systemd service (`sudo sytemctl restart cloudgw`)

### As binary file

1. Install and configure VPP (refer to [docs/eng/vpp.adoc](docs/eng/vpp.adoc))
2. Build Cloudgw binary (refer to `Taskfile.yml`)
3. Configure Cloudgw configuration file `config.conf` (refer to [docs/eng/cloudgw_config.adoc](docs/eng/cloudgw_config.adoc))
4. Run the Cloudgw (`sudo CLOUDGW_CONFIG_PATH=config.yml ./cloudgw`)

NOTE: You need to configure Cloudgw as BGP Router in Tungsten Fabric as well with correct Address Family (inet-vpn) and Route Target (refer to [docs/eng/overlay.adoc](docs/eng/overlay.adoc)) 

### Operational

Refer to [docs/eng/cloudgw_usage.adoc](docs/eng/cloudgw_usage.adoc)

## Documentation 

Refer to [docs/](docs/) for more documentation.
