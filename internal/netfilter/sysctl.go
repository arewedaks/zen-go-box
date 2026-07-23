package netfilter

func EnableIPForwarding(ipv6 bool) {
	execIgnoreError("sysctl", "-w", "net.ipv4.ip_forward=1")
	execIgnoreError("sysctl", "-w", "net.ipv4.conf.default.rp_filter=2")
	execIgnoreError("sysctl", "-w", "net.ipv4.conf.all.rp_filter=2")

	if ipv6 {
		execIgnoreError("sysctl", "-w", "net.ipv6.conf.all.forwarding=1")
		execIgnoreError("sysctl", "-w", "net.ipv6.conf.all.accept_ra=2")
		execIgnoreError("sysctl", "-w", "net.ipv6.conf.wlan0.accept_ra=2")
		execIgnoreError("sysctl", "-w", "net.ipv6.conf.all.disable_ipv6=0")
		execIgnoreError("sysctl", "-w", "net.ipv6.conf.default.disable_ipv6=0")
		execIgnoreError("sysctl", "-w", "net.ipv6.conf.wlan0.disable_ipv6=0")
	}
}
