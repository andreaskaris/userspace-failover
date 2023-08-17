package main

import (
	"fmt"
	"net/netip"
	"os"
	"strconv"
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

	vlanID, err := strconv.Atoi(os.Getenv("VLAN_ID"))
	if err != nil {
		klog.Fatal("Please provide a valid VLAN ID via VLAN_ID env var, err: %q", err)
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

	arpInterval := 100
	if ai, err := strconv.Atoi(os.Getenv("ARP_INTERVAL")); err != nil && ai > 0 {
		arpInterval = ai
	}

	arpTimeout := 50
	if ai, err := strconv.Atoi(os.Getenv("ARP_TIMEOUT")); err != nil && ai > 0 {
		arpInterval = ai
	}

	linkMonitorInterval := 100
	if lmi, err := strconv.Atoi(os.Getenv("LINK_MONITOR_INTERVAL")); err != nil && lmi > 0 {
		linkMonitorInterval = lmi
	}

	f := failover.New(primaryInterface, secondaryInterface, vlanID, ip, arpTargetIP, standbyVLAN,
		arpInterval, arpTimeout, linkMonitorInterval)
	f.Setup()

	errChan := make(chan error)
	go func() {
		errChan <- fmt.Errorf("LinkMonitor exited with error, err: %q", f.LinkMonitor())
	}()
	go func() {
		errChan <- fmt.Errorf("Prober exited with error, err: %q", f.Prober())
	}()
	select {
	case err := <-errChan:
		klog.Error(err)
	}
}
