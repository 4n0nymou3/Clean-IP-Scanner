package scanner

import (
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"strings"
	"time"
)

const maxIPsPerCIDR = 5_000_000

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

func cidrCount(ipNet *net.IPNet) *big.Int {
	ones, bits := ipNet.Mask.Size()
	if bits == 0 {
		return big.NewInt(0)
	}
	hostBits := uint(bits - ones)
	count := new(big.Int).Lsh(big.NewInt(1), hostBits)
	return count
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

	count := cidrCount(ipNet)
	limit := big.NewInt(maxIPsPerCIDR)
	if count.Cmp(limit) > 0 {
		ones, bits := ipNet.Mask.Size()
		proto := "IPv4"
		if bits == 128 {
			proto = "IPv6"
		}
		neededBits := bits - int(log2Ceil(maxIPsPerCIDR))
		fmt.Printf("Skipping %s (%s /%d): contains %s addresses — exceeds limit of %s.\n",
			cidr, proto, ones,
			formatBigInt(count),
			formatBigInt(limit),
		)
		if bits == 128 {
			fmt.Printf("  For IPv6, use /%d or smaller (e.g., %s/%d)\n",
				neededBits, ipNet.IP.String(), neededBits)
		}
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

func log2Ceil(n int) int {
	result := 0
	val := 1
	for val < n {
		val <<= 1
		result++
	}
	return result
}

func formatBigInt(n *big.Int) string {
	s := n.String()
	if len(s) <= 3 {
		return s
	}
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
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