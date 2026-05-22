package scanner

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

type CompactIP [16]byte

func compactFromNetIP(ip net.IP) CompactIP {
	var c CompactIP
	ip16 := ip.To16()
	if ip16 != nil {
		copy(c[:], ip16)
	}
	return c
}

func (c CompactIP) ToNetIPAddr() *net.IPAddr {
	ip := make(net.IP, 16)
	copy(ip, c[:])
	if ip4 := ip.To4(); ip4 != nil {
		return &net.IPAddr{IP: ip4}
	}
	return &net.IPAddr{IP: ip}
}

func (c CompactIP) String() string {
	ip := make(net.IP, 16)
	copy(ip, c[:])
	if ip4 := ip.To4(); ip4 != nil {
		return ip4.String()
	}
	return ip.String()
}

type IPRanges struct {
	ips  []CompactIP
	seen map[string]bool
}

func newIPRanges() *IPRanges {
	return &IPRanges{
		ips:  make([]CompactIP, 0),
		seen: make(map[string]bool),
	}
}

func (r *IPRanges) appendIP(ip net.IP) {
	r.ips = append(r.ips, compactFromNetIP(ip))
}

func (r *IPRanges) expandCIDR(cidr string) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		fmt.Printf("ParseCIDR error for %s: %v\n", cidr, err)
		return
	}
	networkKey := ipNet.String()
	if r.seen[networkKey] {
		return
	}
	r.seen[networkKey] = true
	ip := cloneIP(ipNet.IP)
	for ipNet.Contains(ip) {
		clone := make(net.IP, len(ip))
		copy(clone, ip)
		r.appendIP(clone)
		incrementIP(ip)
	}
}

func cloneIP(ip net.IP) net.IP {
	clone := make(net.IP, len(ip))
	copy(clone, ip)
	return clone
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

func isIPv4(ip string) bool {
	return strings.Contains(ip, ".")
}

func buildIPRanges(ranges []string) *IPRanges {
	ipRanges := newIPRanges()
	for _, r := range ranges {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if !strings.Contains(r, "/") {
			if isIPv4(r) {
				r += "/32"
			} else {
				r += "/128"
			}
		}
		ipRanges.expandCIDR(r)
	}
	return ipRanges
}

func GenerateIPs(ranges []string) ([]CompactIP, int64) {
	seed := time.Now().UnixNano()
	ipRanges := buildIPRanges(ranges)
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(ipRanges.ips), func(i, j int) {
		ipRanges.ips[i], ipRanges.ips[j] = ipRanges.ips[j], ipRanges.ips[i]
	})
	return ipRanges.ips, seed
}

func GenerateIPsWithSeed(ranges []string, seed int64) []CompactIP {
	ipRanges := buildIPRanges(ranges)
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(ipRanges.ips), func(i, j int) {
		ipRanges.ips[i], ipRanges.ips[j] = ipRanges.ips[j], ipRanges.ips[i]
	})
	return ipRanges.ips
}