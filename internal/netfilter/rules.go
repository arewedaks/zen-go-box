package netfilter

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// CIDR Intranet V4 & V6 (Bypass list)
var IntranetV4 = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.255.255.255/32",
}

var IntranetV6 = []string{
	"::/128",
	"::1/128",
	"::ffff:0:0/96",
	"100::/64",
	"64:ff9b::/96",
	"2001::/32",
	"2001:2::/48",
	"2001:db8::/32",
	"2002::/16",
	"fc00::/7",
	"fe80::/10",
	"ff00::/8",
}

// SetIPv6Enable mengaktifkan/menonaktifkan IPv6 secara system-wide di kernel Linux
func SetIPv6Enable(enable bool) error {
	val := "1"
	if enable {
		val = "0" // 0 = disable_ipv6 false (artinya enable)
	}

	paths := []string{
		"/proc/sys/net/ipv6/conf/all/disable_ipv6",
		"/proc/sys/net/ipv6/conf/default/disable_ipv6",
	}

	for _, path := range paths {
		if err := os.WriteFile(path, []byte(val), 0644); err != nil {
			// Beberapa Android custom ROM/kernel tidak mem-mount writeable /proc
			return fmt.Errorf("failed to write to %s: %w", path, err)
		}
	}
	return nil
}

// GetLocalIPs mengembalikan semua alamat IP IPv4 & IPv6 aktif dari semua network interfaces
func GetLocalIPs() ([]string, []string, error) {
	var v4, v6 []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}

	for _, iface := range ifaces {
		// Abaikan interfaces loopback dan non-up
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil {
				continue
			}

			if ip.To4() != nil {
				v4 = append(v4, ip.String())
			} else {
				// Pastikan IPv6 global, bukan link-local (fe80::)
				if !strings.HasPrefix(ip.String(), "fe80") {
					v6 = append(v6, ip.String())
				}
			}
		}
	}

	return v4, v6, nil
}
