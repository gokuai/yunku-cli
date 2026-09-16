package security

import (
	"fmt"
	"net"
	"slices"
	"strings"
)

// virtualMACPrefixes lists MAC address prefixes commonly used by
// hypervisors, Docker, and VPN adapters.
var virtualMACPrefixes = []string{
	"00:00:00",
	"02:42:",
	"02:00:",
	"fe:ff:",
	"00:50:56",
	"00:0c:29",
	"00:1c:42",
	"08:00:27",
	"52:54:00",
}

func isVirtualMAC(mac string) bool {
	for _, p := range virtualMACPrefixes {
		if strings.HasPrefix(mac, p) {
			return true
		}
	}
	return false
}

// selectMAC picks the best MAC from the given interface list.
// Physical NICs are preferred; virtual MACs are used as fallback.
func selectMAC(ifaces []net.Interface) (string, error) {
	var physical, virtual []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		hw := iface.HardwareAddr
		if len(hw) == 0 {
			continue
		}
		s := strings.ToLower(hw.String())
		if isVirtualMAC(s) {
			virtual = append(virtual, s)
		} else {
			physical = append(physical, s)
		}
	}
	if len(physical) > 0 {
		slices.Sort(physical)
		return physical[0], nil
	}
	if len(virtual) > 0 {
		slices.Sort(virtual)
		return virtual[0], nil
	}
	return "", fmt.Errorf("no NIC MAC address found (no network interfaces available)")
}

// GetMACAddress returns the first physical (non-loopback, non-virtual) MAC
// address, sorted lexicographically for determinism. If no physical NIC is
// found (e.g. inside a Docker container), it falls back to the first
// non-loopback virtual MAC address so that token encryption still works.
func GetMACAddress() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("enumerating network interfaces: %w", err)
	}
	return selectMAC(ifaces)
}
