#!/usr/bin/env python3

from scapy.all import *
from time import sleep

# Tungsten Fabric VM's VIP addresses
SRC_IP_IN1 = "100.64.0.1"
SRC_IP_IN2 = "100.64.0.2"
SRC_IP_IN3 = "100.64.0.3"

# External IP address (outside Cloud)
DST_IP_IN = "1.1.1.1"

# vRouter Address
SRC_IP_EXT = "192.168.57.30"

# VPP Main Interface Address
DST_IP_EXT = "192.168.57.10"

# MPLS over UDP ports
DST_PORT = 6635

SRC_PORT1 = 5001
SRC_PORT2 = 5002
SRC_PORT3 = 5003

# MPLS local-label of Cloud-GW announced to vRouters
LABEL = 1004051
TTL = 64

load_contrib("mpls")

def main():
    while True:

        ip_layer_in1 = IP(src=SRC_IP_IN1, dst=DST_IP_IN)
        ip_layer_in2 = IP(src=SRC_IP_IN2, dst=DST_IP_IN)
        ip_layer_in3 = IP(src=SRC_IP_IN3, dst=DST_IP_IN)

        ip_layer_ext = IP(src=SRC_IP_EXT, dst=DST_IP_EXT)

        udp_layer1 = UDP(sport=SRC_PORT1, dport=DST_PORT)
        udp_layer2 = UDP(sport=SRC_PORT2, dport=DST_PORT)
        udp_layer3 = UDP(sport=SRC_PORT3, dport=DST_PORT)

        mpls_layer = MPLS(label=LABEL, s=1, ttl=TTL)

        ping_icmp = ICMP(type="echo-request")
        ping_raw = Raw(load="123456789012345678901234567890")

        packet1 = (ip_layer_ext / udp_layer1 / mpls_layer / ip_layer_in1 / ping_icmp / ping_raw)
        packet2 = (ip_layer_ext / udp_layer2 / mpls_layer / ip_layer_in2 / ping_icmp / ping_raw)
        packet3 = (ip_layer_ext / udp_layer3 / mpls_layer / ip_layer_in3 / ping_icmp / ping_raw)

        print("Send: ", repr(packet1))
        send(packet1, verbose=False)
        print("Send: ", repr(packet2))
        send(packet2, verbose=False)
        print("Send: ", repr(packet3))
        send(packet3, verbose=False)

        sleep(2)

if __name__ == "__main__":
    main()
