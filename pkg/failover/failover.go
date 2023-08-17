package failover

import (
	"net/netip"

	"k8s.io/klog"
)

type Failover struct {
	PrimaryInterface   string
	SecondaryInterface string
	VLANID             int
	IP                 netip.Prefix
	ArpTarget          netip.Addr
	StandbyVLANMode    bool
}

func New(primaryInterface, secondaryInterface string, vlanID int, ip netip.Prefix, arpTarget netip.Addr, standbyVLANMode bool) *Failover {
	return &Failover{
		PrimaryInterface:   primaryInterface,
		SecondaryInterface: secondaryInterface,
		VLANID:             vlanID,
		IP:                 ip,
		ArpTarget:          arpTarget,
		StandbyVLANMode:    standbyVLANMode,
	}
}

func (f *Failover) Run() {
	klog.Info("Running failover daemon with ... PrimaryInterface: %s, SecondaryInterface: %s, VLAN ID: %d, IP: %s, arpTarget: %s, standbyVLANMode: %b",
		f.PrimaryInterface, f.SecondaryInterface, f.VLANID, f.IP, f.ArpTarget, f.StandbyVLANMode)
}
