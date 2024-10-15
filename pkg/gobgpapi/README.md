# BGP Update Examples

## VPNv4 Update from Tungsten Fabric Controller for FIP=`10.11.64.36` (MP_REACH_NLRI)

`bgpPath.String()`

```json
nlri:{[type.googleapis.com/apipb.LabeledVPNIPAddressPrefix]:{
    labels:46
    rd:{[type.googleapis.com/apipb.RouteDistinguisherIPAddress]:{
        admin:"10.11.0.10"
        assigned:2}}
    prefix_len:32
    prefix:"10.11.64.36"}}
pattrs:{[type.googleapis.com/apipb.OriginAttribute]:{origin:2}}
pattrs:{[type.googleapis.com/apipb.MultiExitDiscAttribute]:{med:100}}
pattrs:{[type.googleapis.com/apipb.AsPathAttribute]:{segments:{
    type:AS_SEQUENCE
    numbers:64550}}}
pattrs:{[type.googleapis.com/apipb.ExtendedCommunitiesAttribute]:{
    communities:{[type.googleapis.com/apipb.TwoOctetAsSpecificExtended]:{
        is_transitive:true
        sub_type:2
        asn:64550
        local_admin:1}}
    communities:{[type.googleapis.com/apipb.TwoOctetAsSpecificExtended]:{
        is_transitive:true
        sub_type:2  asn:64550
        local_admin:8000005}}
    communities:{[type.googleapis.com/apipb.TwoOctetAsSpecificExtended]:{
        is_transitive:true
        sub_type:2
        asn:64552
        local_admin:100}}
    communities:{[type.googleapis.com/apipb.TwoOctetAsSpecificExtended]:{
        is_transitive:true
        sub_type:2  asn:64553
        local_admin:100}}
    communities:{[type.googleapis.com/apipb.EncapExtended]:{tunnel_type:2}}  // GRE
    communities:{[type.googleapis.com/apipb.EncapExtended]:{tunnel_type:13}} // MPLS in UDP
    communities:{[type.googleapis.com/apipb.RouterMacExtended]:{mac:"78:ac:44:65:30:58"}}
    communities:{[type.googleapis.com/apipb.UnknownExtended]:{
        type:128
        value:"q\xfc&\x00\x00\x00\x06"}}}}
pattrs:{[type.googleapis.com/apipb.MpReachNLRIAttribute]:{
    family:{
        afi:AFI_IP
        safi:SAFI_MPLS_VPN}
    next_hops:"10.11.0.10"
    nlris:{[type.googleapis.com/apipb.LabeledVPNIPAddressPrefix]:{
        labels:46
        rd:{[type.googleapis.com/apipb.RouteDistinguisherIPAddress]:{
            admin:"10.11.0.10"
            assigned:2}}
        prefix_len:32
        prefix:"10.11.64.36"}}}}
age:{seconds:1677173994}
validation:{}
family:{
    afi:AFI_IP
    safi:SAFI_MPLS_VPN}
source_asn:64550
source_id:"10.3.167.1"
neighbor_ip:"10.3.167.1"
local_identifier:3
```

## VPNv4 Update from Tungsten Fabric Controller for FIP=`10.11.64.1` (MP_UNREACH_NLRI)

```json
nlri:{[type.googleapis.com/apipb.LabeledVPNIPAddressPrefix]:{
    labels:0 
    rd:{[type.googleapis.com/apipb.RouteDistinguisherIPAddress]:{
        admin:"10.0.0.10" 
        assigned:1}} 
    prefix_len:32 
    prefix:"10.11.64.1"}} 
age:{seconds:1677556330} 
is_withdraw:true 
validation:{} 
family:{
    afi:AFI_IP 
    safi:SAFI_MPLS_VPN} 
source_asn:65001 
source_id:"10.255.255.107" 
neighbor_ip:"10.255.255.107" 
local_identifier:1
```

## VPNv4 Update from physical network for "0.0.0.0/0" in specific VRF

```json
nlri:{[type.googleapis.com/apipb.LabeledVPNIPAddressPrefix]:{
    labels:0 
    rd:{[type.googleapis.com/apipb.RouteDistinguisherIPAddress]:{
        admin:"10.1.1.2" 
        assigned:1}} 
    prefix:"0.0.0.0"}} 
pattrs:{[type.googleapis.com/apipb.OriginAttribute]:{origin:2}} 
pattrs:{[type.googleapis.com/apipb.AsPathAttribute]:{segments:{type:AS_SEQUENCE numbers:65535}}} 
pattrs:{[type.googleapis.com/apipb.ExtendedCommunitiesAttribute]:{
    communities:{[type.googleapis.com/apipb.TwoOctetAsSpecificExtended]:{
        is_transitive:true 
        asn:64999 
        local_admin:1}}}} 
pattrs:{[type.googleapis.com/apipb.MpReachNLRIAttribute]:{
    family:{
        afi:AFI_IP 
        safi:SAFI_MPLS_VPN} 
    next_hops:"172.20.20.200" 
    nlris:{[type.googleapis.com/apipb.LabeledVPNIPAddressPrefix]:{
        labels:0 
        rd:{[type.googleapis.com/apipb.RouteDistinguisherIPAddress]:{
            admin:"10.1.1.2" 
            assigned:1}} 
        prefix:"0.0.0.0"}}}} 
age:{seconds:1677180368} 
validation:{} 
family:{
    afi:AFI_IP 
    safi:SAFI_MPLS_VPN} 
source_asn:65535 
source_id:"172.20.20.200" 
neighbor_ip:"172.20.20.200" 
local_identifier:1
```

## VPNv4 Update from physical network for "0.0.0.0/0" in specific VRF (WITHDRAW)

```json
nlri:{[type.googleapis.com/apipb.LabeledVPNIPAddressPrefix]:{
    labels:0 
    rd:{[type.googleapis.com/apipb.RouteDistinguisherIPAddress]:{
        admin:"10.255.255.109" 
        assigned:1}} 
        prefix:"0.0.0.0"}} 
pattrs:{[type.googleapis.com/apipb.OriginAttribute]:{}} 
pattrs:{[type.googleapis.com/apipb.AsPathAttribute]:{segments:{type:AS_SEQUENCE numbers:65002}}} 
pattrs:{[type.googleapis.com/apipb.ExtendedCommunitiesAttribute]:{
    communities:{[type.googleapis.com/apipb.TwoOctetAsSpecificExtended]:{
        is_transitive:true 
        sub_type:2 asn:65000 
        local_admin:1}}}} 
pattrs:{[type.googleapis.com/apipb.MpReachNLRIAttribute]:{
    family:{afi:AFI_IP safi:SAFI_MPLS_VPN} 
    next_hops:"10.255.255.108" 
    nlris:{[type.googleapis.com/apipb.LabeledVPNIPAddressPrefix]:{
        labels:0 
        rd:{[type.googleapis.com/apipb.RouteDistinguisherIPAddress]:{
            admin:"10.255.255.109" 
            assigned:1}} 
        prefix:"0.0.0.0"}}}} 
age:{seconds:1677570254} 
is_withdraw:true validation:{} 
family:{afi:AFI_IP safi:SAFI_MPLS_VPN} 
source_asn:65002 
source_id:"10.255.255.108" 
neighbor_ip:"10.255.255.108" 
local_identifier:1
```
