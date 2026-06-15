package scanner

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// HostResult is one discovered network device from nmap.
type HostResult struct {
	IPAddress  string
	Hostname   string
	MACAddress string
	Vendor     string
	OSGuess    string
	OpenPorts  []int
	DeviceType string
	IsAP       bool
	IsRadius   bool
	LatencyMs  float64
	Status     string
}

// ValidateSubnet ensures CIDR is valid and not too large (max /24).
func ValidateSubnet(cidr string) (*net.IPNet, error) {
	ip, ipNet, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil {
		return nil, fmt.Errorf("invalid subnet CIDR")
	}
	if ip.To4() == nil {
		return nil, fmt.Errorf("only IPv4 subnets are supported")
	}
	ones, bits := ipNet.Mask.Size()
	if bits-ones > 8 {
		return nil, fmt.Errorf("subnet too large: maximum /24 (256 hosts) allowed")
	}
	if ip.IsLoopback() || ip.IsMulticast() {
		return nil, fmt.Errorf("cannot scan loopback or multicast ranges")
	}
	return ipNet, nil
}

// RunScan executes nmap and returns discovered hosts.
func RunScan(ctx context.Context, subnet, scanType string) ([]HostResult, error) {
	if scanType == "" {
		scanType = "discovery"
	}

	args := buildNmapArgs(subnet, scanType)
	cmd := exec.CommandContext(ctx, "nmap", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("nmap failed: %s", msg)
	}

	hosts, err := parseNmapXML(out)
	if err != nil {
		return nil, err
	}

	results := make([]HostResult, 0, len(hosts))
	for _, h := range hosts {
		if h.Status != "up" {
			continue
		}
		classifyHost(&h)
		results = append(results, h)
	}

	sort.Slice(results, func(i, j int) bool {
		return ipLess(results[i].IPAddress, results[j].IPAddress)
	})

	return results, nil
}

func buildNmapArgs(subnet, scanType string) []string {
	base := []string{
		"-oX", "-",
		"-T4",
		"--max-retries", "1",
		"--host-timeout", "30s",
	}

	switch scanType {
	case "ping":
		return append(base, "-sn", subnet)
	case "standard":
		return append(base, "-sT", "--open", "--top-ports", "100", subnet)
	case "ap", "full":
		return append(base,
			"-sT", "--open",
			"-p", "21,22,23,53,80,161,443,8080,8443,8291,8728,8729,1812,1813,9090,10443",
			"--top-ports", "50",
			subnet,
		)
	default:
		return append(base, "-sn", "-PS22,80,443,8080,8291", "-PE", subnet)
	}
}

func ipLess(a, b string) bool {
	ipA := net.ParseIP(a).To4()
	ipB := net.ParseIP(b).To4()
	if ipA == nil || ipB == nil {
		return a < b
	}
	for i := 0; i < 4; i++ {
		if ipA[i] != ipB[i] {
			return ipA[i] < ipB[i]
		}
	}
	return false
}

type nmapRun struct {
	Hosts []nmapHost `xml:"host"`
}

type nmapHost struct {
	Status    nmapStatus     `xml:"status"`
	Addresses []nmapAddress  `xml:"address"`
	Hostnames []nmapHostname `xml:"hostnames>hostname"`
	Ports     nmapPorts      `xml:"ports"`
	OS        nmapOS         `xml:"os"`
	Times     nmapTimes      `xml:"times"`
}

type nmapStatus struct {
	State string `xml:"state,attr"`
}

type nmapAddress struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"`
	Vendor   string `xml:"vendor,attr"`
}

type nmapHostname struct {
	Name string `xml:"name,attr"`
}

type nmapPorts struct {
	Port []nmapPort `xml:"port"`
}

type nmapPort struct {
	Protocol string      `xml:"protocol,attr"`
	PortID   int         `xml:"portid,attr"`
	State    nmapState   `xml:"state"`
	Service  nmapService `xml:"service"`
}

type nmapState struct {
	State string `xml:"state,attr"`
}

type nmapService struct {
	Name string `xml:"name,attr"`
}

type nmapOS struct {
	Matches []nmapOSMatch `xml:"osmatch"`
}

type nmapOSMatch struct {
	Name string `xml:"name,attr"`
}

type nmapTimes struct {
	RTT string `xml:"rtt,attr"`
}

func parseNmapXML(data []byte) ([]HostResult, error) {
	var run nmapRun
	if err := xml.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("parse nmap XML: %w", err)
	}

	results := make([]HostResult, 0, len(run.Hosts))
	for _, h := range run.Hosts {
		r := HostResult{Status: h.Status.State}
		for _, addr := range h.Addresses {
			switch addr.AddrType {
			case "ipv4", "ipv6":
				if r.IPAddress == "" {
					r.IPAddress = addr.Addr
				}
			case "mac":
				r.MACAddress = addr.Addr
				r.Vendor = addr.Vendor
			}
		}
		for _, hn := range h.Hostnames {
			if hn.Name != "" {
				r.Hostname = hn.Name
				break
			}
		}
		for _, p := range h.Ports.Port {
			if p.State.State == "open" || p.State.State == "open|filtered" {
				r.OpenPorts = append(r.OpenPorts, p.PortID)
			}
		}
		sort.Ints(r.OpenPorts)
		if len(h.OS.Matches) > 0 {
			r.OSGuess = h.OS.Matches[0].Name
		}
		if h.Times.RTT != "" {
			if ms, err := strconv.ParseFloat(strings.TrimSuffix(h.Times.RTT, "ms"), 64); err == nil {
				r.LatencyMs = ms
			}
		}
		if r.IPAddress != "" {
			results = append(results, r)
		}
	}
	return results, nil
}

func classifyHost(h *HostResult) {
	portSet := map[int]bool{}
	for _, p := range h.OpenPorts {
		portSet[p] = true
	}

	vendor := strings.ToLower(h.Vendor)
	host := strings.ToLower(h.Hostname)
	osName := strings.ToLower(h.OSGuess)

	h.IsRadius = portSet[1812] || portSet[1813]

	if portSet[8291] || portSet[8728] || portSet[8729] || strings.Contains(vendor, "mikrotik") {
		h.DeviceType = "router"
		if portSet[80] || portSet[443] || portSet[8080] {
			h.IsAP = true
			h.DeviceType = "access_point"
		}
		return
	}

	if strings.Contains(vendor, "ubiquiti") || strings.Contains(host, "unifi") || strings.Contains(host, "uap-") {
		h.IsAP = true
		h.DeviceType = "access_point"
		return
	}

	apVendors := []string{"aruba", "ruckus", "tp-link", "meraki", "cisco", "engenius", "ruijie", "h3c", "netgear"}
	for _, v := range apVendors {
		if strings.Contains(vendor, v) {
			if portSet[80] || portSet[443] || portSet[8080] || portSet[8443] {
				h.IsAP = true
				h.DeviceType = "access_point"
				return
			}
			h.DeviceType = "network_device"
			return
		}
	}

	if strings.Contains(host, "ap-") || strings.Contains(host, "wifi") || strings.Contains(host, "wireless") {
		h.IsAP = true
		h.DeviceType = "access_point"
		return
	}

	if portSet[161] && (portSet[22] || portSet[23]) {
		h.DeviceType = "switch"
		return
	}

	if portSet[22] && (portSet[80] || portSet[443]) {
		h.DeviceType = "network_device"
		return
	}

	if portSet[80] || portSet[443] || portSet[8080] {
		h.DeviceType = "web_device"
		return
	}

	if strings.Contains(osName, "linux") || strings.Contains(osName, "windows") {
		h.DeviceType = "server"
		return
	}

	if h.MACAddress != "" || len(h.OpenPorts) > 0 {
		h.DeviceType = "host"
		return
	}

	h.DeviceType = "unknown"
}

// ScanTimeout returns max duration for a scan type.
func ScanTimeout(scanType string) time.Duration {
	switch scanType {
	case "ping", "discovery":
		return 3 * time.Minute
	case "standard":
		return 8 * time.Minute
	default:
		return 12 * time.Minute
	}
}

// NmapAvailable checks if nmap is installed.
func NmapAvailable() bool {
	_, err := exec.LookPath("nmap")
	return err == nil
}
