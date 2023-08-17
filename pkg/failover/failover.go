package failover

import (
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"time"

	"github.com/j-keck/arping"
	"github.com/vishvananda/netlink"
	"k8s.io/klog"
)

type InterfaceType bool

const (
	InterfacePrimary   InterfaceType = true
	InterfaceSecondary InterfaceType = false
)

type Failover struct {
	PrimaryInterface    string
	SecondaryInterface  string
	VLANID              int
	IP                  netip.Prefix
	ArpTarget           netip.Addr
	StandbyVLANMode     bool
	ArpInterval         int // ArpInterval in ms.
	ArpTimeout          int // ArpTimeout in ms.
	LinkMonitorInterval int // LinkMonitorInterval in ms.
	activeInterface     InterfaceType
	arpState            bool
	lastArpTimestamp    time.Time
}

func New(primaryInterface, secondaryInterface string, vlanID int, ip netip.Prefix, arpTarget netip.Addr,
	standbyVLANMode bool, arpInterval, arpTimeout, linkMonitorInterval int) *Failover {
	return &Failover{
		PrimaryInterface:    primaryInterface,
		SecondaryInterface:  secondaryInterface,
		VLANID:              vlanID,
		IP:                  ip,
		ArpTarget:           arpTarget,
		StandbyVLANMode:     standbyVLANMode,
		ArpInterval:         arpInterval,
		ArpTimeout:          arpTimeout,
		LinkMonitorInterval: linkMonitorInterval,
		arpState:            true,
	}
}

// Setup sets up the failover daemon.
func (f *Failover) Setup() {
	klog.Infof("Running failover daemon with ... PrimaryInterface: %s, SecondaryInterface: %s, VLAN ID: %d, IP: %s, arpTarget: %s, standbyVLANMode: %t",
		f.PrimaryInterface, f.SecondaryInterface, f.VLANID, f.IP, f.ArpTarget, f.StandbyVLANMode)

	// Keep track of our active interface (the one holding the VLAN).
	f.activeInterface = InterfacePrimary

	if err := f.SetupVLANInterface(InterfacePrimary); err != nil {
		klog.Fatalf("Error setting up VLAN interface, err: %q", err)
	}
	if err := f.RemoveVLANInterface(InterfaceSecondary); err != nil {
		klog.Fatalf("Error deleting VLAN interface, err: %q", err)
	}
	if err := f.AddIPOnVLANInterface(InterfacePrimary); err != nil {
		klog.Fatalf("Error adding IP address to VLAN interface, err: %q", err)
	}

	klog.Infof("Done with interface setup")
	klog.Infof("Interface output:\n%s", runCmd("ip", "address"))
}

// LinkMonitor runs the failover mechanism.
func (f *Failover) LinkMonitor() error {
	for {
		time.Sleep(time.Duration(f.LinkMonitorInterval) * time.Millisecond)

		// We can't do anything if both interfaces are down.
		if !f.LinkUp(f.activeInterface) && !f.LinkUp(!f.activeInterface) {
			klog.Warningf("Both interfaces are down, checking again in 10 * LinkMonitorInterval ms")
			time.Sleep(time.Duration(f.LinkMonitorInterval*9) * time.Millisecond)
			continue
		}

		// Otherwise, if the active interface is down, failover to the inactive interface.
		if !f.LinkUp(f.activeInterface) {
			if err := f.RemoveVLANInterface(f.activeInterface); err != nil {
				return fmt.Errorf("error removing VLAN interface during failover, err: %q", err)
			}
			// Change active interface, failover.
			f.activeInterface = !f.activeInterface
			if err := f.SetupVLANInterface(f.activeInterface); err != nil {
				return fmt.Errorf("error setting up VLAN interface after failover, err: %q", err)
			}
			if err := f.AddIPOnVLANInterface(f.activeInterface); err != nil {
				return fmt.Errorf("error adding IP address to VLAN interface after failover, err: %q", err)
			}
		}
	}
}

// Prober runs the prober mechanism.
func (f *Failover) Prober() error {
	var lastUpdateTime time.Time

	for {
		time.Sleep(time.Duration(f.ArpInterval) * time.Millisecond)
		if f.SendArp() {
			if !f.arpState {
				arpDownDuration := time.Now().Sub(f.lastArpTimestamp)
				klog.Infof("arping reported success. ARP was down for %s", arpDownDuration)
			}
			f.lastArpTimestamp = time.Now()
			f.arpState = true
		} else {
			// Transition from good ARP state to bad ARP state.
			if f.arpState {
				klog.Infof("arping reported failure")
				lastUpdateTime = time.Now()
				f.arpState = false
				continue
			}

			// ARP state was bad before.
			if time.Now().Sub(lastUpdateTime) > 1*time.Second {
				arpDownDuration := time.Now().Sub(f.lastArpTimestamp)
				klog.Infof("arping still reporting down. ARP is down for %s", arpDownDuration)
				lastUpdateTime = time.Now()
			}
			f.arpState = false
		}
	}
}

// SetupVLANInterface creates the VLAN interface on top of the selected interface.
func (f *Failover) SetupVLANInterface(interfaceType InterfaceType) error {
	interfaceName := f.SecondaryInterface
	if interfaceType == InterfacePrimary {
		interfaceName = f.PrimaryInterface
	}
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("could not find link, err: %q", err)
	}

	vName := vlanName(interfaceName, f.VLANID)
	vlan := netlink.Vlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        vName,
			ParentIndex: link.Attrs().Index,
		},
		VlanId: f.VLANID,
	}

	if err := netlink.LinkAdd(&vlan); err != nil {
		return fmt.Errorf("could not setup VLAN interface %v, err: %q", vlan, err)
	}
	return netlink.LinkSetUp(&vlan)
}

// RemoveVLANInterface returns the VLAN interface on the selected interface if it exists.
func (f *Failover) RemoveVLANInterface(interfaceType InterfaceType) error {
	interfaceName := f.SecondaryInterface
	if interfaceType == InterfacePrimary {
		interfaceName = f.PrimaryInterface
	}

	vName := vlanName(interfaceName, f.VLANID)
	link, err := netlink.LinkByName(vName)
	if err != nil {
		klog.V(2).Infof("could not find VLAN interface for removal, err: %q", err)
		return nil
	}
	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("could not remove VLAN interface %q, err: %q", vName, err)
	}
	return nil
}

// AddIPOnVLANInterface adds the IP address to the specified interface.
func (f *Failover) AddIPOnVLANInterface(interfaceType InterfaceType) error {
	interfaceName := f.SecondaryInterface
	if interfaceType == InterfacePrimary {
		interfaceName = f.PrimaryInterface
	}
	vName := vlanName(interfaceName, f.VLANID)
	link, err := netlink.LinkByName(vName)
	if err != nil {
		return fmt.Errorf("could not find VLAN interface to add IP address to, err: %q", err)
	}
	addr, err := netlink.ParseAddr(f.IP.String())
	if err != nil {
		return fmt.Errorf("netlink could not parse IP address %q, this should never happen, err: %q", f.IP.String(), err)
	}
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("could not add IP address %q to VLAN interface %q, err: %q", f.IP.String(), interfaceName, err)
	}
	return nil
}

// RemoveIPFromVLANInterface removes the IP address from the specified interface.
func (f *Failover) RemoveIPFromVLANInterface(interfaceType InterfaceType) error {
	interfaceName := f.SecondaryInterface
	if interfaceType == InterfacePrimary {
		interfaceName = f.PrimaryInterface
	}
	vName := vlanName(interfaceName, f.VLANID)
	link, err := netlink.LinkByName(vName)
	if err != nil {
		klog.V(2).Infof("could not find VLAN interface for IP address removal, err: %q", err)
		return nil
	}
	addr, err := netlink.ParseAddr(f.IP.String())
	if err != nil {
		return fmt.Errorf("netlink could not parse IP address %q, this should never happen, err: %q", f.IP.String(), err)
	}
	err = netlink.AddrDel(link, addr)
	if err != nil {
		klog.V(2).Infof("could not delete IP address %q from interface %q (ignoring), err: %q", f.IP.String(), interfaceName, err)
	}
	return nil
}

// LinkUp reports if the current interface is up.
func (f *Failover) LinkUp(interfaceType InterfaceType) bool {
	interfaceName := f.SecondaryInterface
	if interfaceType == InterfacePrimary {
		interfaceName = f.PrimaryInterface
	}
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		klog.Warningf("Could not find link during LinkUp(), err: %q", err)
		return false
	}
	return link.Attrs().OperState == netlink.OperUp
}

// SendArp sends an ARP packet and reports if a reply was received - it blocks until it receives the answer.
func (f *Failover) SendArp() bool {
	arping.SetTimeout(time.Duration(f.ArpTimeout) * time.Millisecond)
	hwAddr, d, err := arping.Ping(net.ParseIP(f.ArpTarget.String()))
	klog.V(2).Infof("Sent arping to %s, got hwAddr: %s, duration: %s, err: %q", f.ArpTarget, hwAddr, d, err)
	return err == nil
}

// runCmd is a small helper to exec commands and get their output.
func runCmd(args ...string) string {
	cmd := exec.Command(args[0], args[1:]...)
	output, _ := cmd.CombinedOutput()
	return string(output)
}

// vlanName is a helper to return a formatted VLAN for an interface / ID combination.
func vlanName(ifName string, vlanID int) string {
	return fmt.Sprintf("%s.%d", ifName, vlanID)
}
