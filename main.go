package main

import (
	"net/netip"
	"os"
	"strings"

	"github.com/andreaskaris/userspace-failover/pkg/failover"
	"k8s.io/klog"
)

func main() {
	primaryInterface := os.Getenv("PRIMARY_INTERFACE")
	if primaryInterface == "" {
		klog.Fatalf("Please provide a primary interface name via PRIMARY_INTERFACE env var")
	}

	secondaryInterface := os.Getenv("SECONDARY_INTERFACE")
	if secondaryInterface == "" {
		klog.Fatalf("Please provide a secondary interface via SECONDARY_INTERFACE env var")
	}

	ip, err := netip.ParsePrefix(os.Getenv("IP_ADDRESS"))
	if err != nil {
		klog.Fatal("Please provide a valid prefix via IP_ADDRESS env var, err: %q", err)
	}

	arpTargetIP, err := netip.ParseAddr(os.Getenv("ARP_TARGET"))
	if err != nil {
		klog.Fatalf("Please provide a valid IP address via ARP_TARGET env var, err: %q", err)
	}

	standbyVLAN := false
	if strings.ToLower(os.Getenv("STANDBY_VLAN_MODE")) == "true" {
		standbyVLAN = true
	}

	failover.New(primaryInterface, secondaryInterface, ip, arpTargetIP, standbyVLAN).Run()
}
