package scanner

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMaxPrefix        = 18
	defaultMaxPortScanHosts = 512
	largeSubnetThreshold    = 256
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

// MaxAllowedPrefix returns the smallest prefix length allowed (e.g. 18 = up to /18).
func MaxAllowedPrefix() int {
	if v := os.Getenv("NMAP_MAX_PREFIX"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p >= 8 && p <= 24 {
			return p
		}
	}
	return defaultMaxPrefix
}

// SubnetHostCount returns the number of addresses in a CIDR block.
func SubnetHostCount(ipNet *net.IPNet) int {
	ones, bits := ipNet.Mask.Size()
	if bits-ones >= 31 {
		return 1
	}
	return 1 << (bits - ones)
}

// ValidateSubnet ensures CIDR is valid and within configured size limits.
func ValidateSubnet(cidr string) (*net.IPNet, error) {
	ip, ipNet, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil {
		return nil, fmt.Errorf("invalid subnet CIDR")
	}
	if ip.To4() == nil {
		return nil, fmt.Errorf("only IPv4 subnets are supported")
	}
	ones, _ := ipNet.Mask.Size()
	maxPrefix := MaxAllowedPrefix()
	if ones < maxPrefix {
		return nil, fmt.Errorf("subnet too large: maximum /%d (%d hosts) allowed", maxPrefix, 1<<(32-maxPrefix))
	}
	if ip.IsLoopback() || ip.IsMulticast() {
		return nil, fmt.Errorf("cannot scan loopback or multicast ranges")
	}
	return ipNet, nil
}

// RunScan executes nmap and returns discovered hosts.
// For subnets larger than /24, port-scan profiles use two phases: discovery then port scan on live hosts.
func RunScan(ctx context.Context, subnet, scanType string) ([]HostResult, error) {
	if scanType == "" {
		scanType = "discovery"
	}

	ipNet, err := ValidateSubnet(subnet)
	if err != nil {
		return nil, err
	}

	large := SubnetHostCount(ipNet) > largeSubnetThreshold
	needsPorts := scanType == "standard" || scanType == "ap" || scanType == "full"

	if large && needsPorts {
		live, err := runNmap(ctx, subnet, "discovery", true)
		if err != nil {
			return nil, err
		}
		if len(live) == 0 {
			return live, nil
		}
		maxHosts := maxPortScanHosts()
		if len(live) > maxHosts {
			live = live[:maxHosts]
		}
		return runPortScanOnHosts(ctx, live, scanType)
	}

	return runNmap(ctx, subnet, scanType, large)
}

func maxPortScanHosts() int {
	if v := os.Getenv("NMAP_MAX_PORTSCAN_HOSTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultMaxPortScanHosts
}

func runNmap(ctx context.Context, subnet, scanType string, large bool) ([]HostResult, error) {
	args := buildNmapArgs(subnet, scanType, large)
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

func runPortScanOnHosts(ctx context.Context, hosts []HostResult, scanType string) ([]HostResult, error) {
	targets := make([]string, len(hosts))
	for i, h := range hosts {
		targets[i] = h.IPAddress
	}

	args := buildPortScanArgs(scanType, false)
	args = append(args, targets...)

	cmd := exec.CommandContext(ctx, "nmap", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("nmap port scan failed: %s", msg)
	}

	scanned, err := parseNmapXML(out)
	if err != nil {
		return nil, err
	}

	byIP := map[string]*HostResult{}
	for i := range hosts {
		byIP[hosts[i].IPAddress] = &hosts[i]
	}

	results := make([]HostResult, 0, len(scanned))
	for _, h := range scanned {
		if h.Status != "up" {
			continue
		}
		if base, ok := byIP[h.IPAddress]; ok {
			h.Hostname = coalesce(h.Hostname, base.Hostname)
			h.MACAddress = coalesce(h.MACAddress, base.MACAddress)
			h.Vendor = coalesce(h.Vendor, base.Vendor)
			if h.LatencyMs == 0 {
				h.LatencyMs = base.LatencyMs
			}
		}
		classifyHost(&h)
		results = append(results, h)
	}

	if len(results) < len(hosts) {
		found := map[string]bool{}
		for _, r := range results {
			found[r.IPAddress] = true
		}
		for _, h := range hosts {
			if !found[h.IPAddress] {
				classifyHost(&h)
				results = append(results, h)
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return ipLess(results[i].IPAddress, results[j].IPAddress)
	})

	return results, nil
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func buildNmapArgs(subnet, scanType string, large bool) []string {
	base := []string{"-oX", "-", "-T4", "--max-retries", "1"}

	if large {
		base = append(base,
			"--min-rate", "300",
			"--max-rtt-timeout", "1500ms",
			"--host-timeout", "15s",
		)
	} else {
		base = append(base, "--host-timeout", "30s")
	}

	switch scanType {
	case "ping":
		return append(base, "-sn", subnet)
	case "standard":
		return append(base, "-sT", "--open", "--top-ports", "100", subnet)
	case "ap", "full":
		args := buildPortScanArgs(scanType, large)
		args = append(args, subnet)
		return args
	default:
		if large {
			return append(base, "-sn", "-PE", "-PP", "-PS22,80,443,8080,8291", subnet)
		}
		return append(base, "-sn", "-PS22,80,443,8080,8291", "-PE", subnet)
	}
}

func buildPortScanArgs(scanType string, large bool) []string {
	base := []string{"-oX", "-", "-T4", "--max-retries", "1", "-sT", "--open"}
	if large {
		base = append(base, "--min-rate", "200", "--host-timeout", "60s")
	} else {
		base = append(base, "--host-timeout", "30s")
	}

	switch scanType {
	case "standard":
		return append(base, "--top-ports", "100")
	case "ap", "full":
		return append(base,
			"-p", "21,22,23,53,80,161,443,8080,8443,8291,8728,8729,1812,1813,9090,10443",
			"--top-ports", "50",
		)
	default:
		return append(base, "--top-ports", "50")
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

// ScanTimeout returns max duration for a scan based on subnet size and type.
func ScanTimeout(subnet, scanType string) time.Duration {
	_, ipNet, err := net.ParseCIDR(strings.TrimSpace(subnet))
	if err != nil {
		return 15 * time.Minute
	}
	hosts := SubnetHostCount(ipNet)
	large := hosts > largeSubnetThreshold
	needsPorts := scanType == "standard" || scanType == "ap" || scanType == "full"

	if large {
		if needsPorts {
			return 90 * time.Minute
		}
		if hosts > 4096 {
			return 60 * time.Minute
		}
		return 30 * time.Minute
	}

	switch scanType {
	case "ping", "discovery":
		return 5 * time.Minute
	case "standard":
		return 10 * time.Minute
	default:
		return 15 * time.Minute
	}
}

// NmapAvailable checks if nmap is installed.
func NmapAvailable() bool {
	_, err := exec.LookPath("nmap")
	return err == nil
}
