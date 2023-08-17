package failover

import (
	"net/netip"

	"k8s.io/klog"
)

type Failover struct {
	PrimaryInterface   string
	SecondaryInterface string
	ArpTarget          netip.Addr
	IP                 netip.Prefix
	StandbyVLANMode    bool
}

func New(primaryInterface, secondaryInterface string, arpTarget netip.Addr, ip netip.Prefix, standbyVLANMode bool) *Failover {
	return &Failover{
		PrimaryInterface:   primaryInterface,
		SecondaryInterface: secondaryInterface,
		ArpTarget:          arpTarget,
		IP:                 ip,
		StandbyVLANMode:    standbyVLANMode,
	}
}

func (f *Failover) Run() {
	klog.Info("Running failover daemon with ... PrimaryInterface: %s, SecondaryInterface: %s, arpTarget: %s, IP: %s, standbyVLANMode: %b",
		f.PrimaryInterface, f.SecondaryInterface, f.ArpTarget, f.IP, f.StandbyVLANMode)
}
