package netfilter

import (
	"fmt"
	"log/slog"

	"github.com/arewedaks/zen-go-box/internal/config"
)

func SetupDNSHijack(cfg *config.Config, ipt *IPT, ipt6 *IPT) {
	if !cfg.Network.ClashDNSForward {
		return
	}
	if cfg.Core.BinName != "clash" && cfg.Core.BinName != "hysteria" {
		return
	}

	slog.Info("Setting up explicit DNS hijacking (NAT REDIRECT)...")

	dnsPort := fmt.Sprintf("%d", cfg.Network.ClashDNSPort)

	// IPv4 NAT
	_ = ipt.Exec("-t", "nat", "-N", "CLASH_DNS_EXTERNAL")
	_ = ipt.Exec("-t", "nat", "-F", "CLASH_DNS_EXTERNAL")
	_ = ipt.Exec("-t", "nat", "-A", "CLASH_DNS_EXTERNAL", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", dnsPort)

	_ = ipt.Exec("-t", "nat", "-N", "CLASH_DNS_LOCAL")
	_ = ipt.Exec("-t", "nat", "-F", "CLASH_DNS_LOCAL")
	_ = ipt.Exec("-t", "nat", "-A", "CLASH_DNS_LOCAL", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", dnsPort)

	_ = ipt.Exec("-t", "nat", "-I", "OUTPUT", "-j", "CLASH_DNS_LOCAL")
	_ = ipt.Exec("-t", "nat", "-I", "PREROUTING", "-j", "CLASH_DNS_EXTERNAL")

	// Cegah IPv6 DNS leak jika IPv6 mati
	if !cfg.Network.IPv6 {
		_ = ipt6.Exec("-I", "OUTPUT", "-p", "udp", "--dport", "53", "-j", "DROP")
	}
}

func CleanDNSHijack() {
	ipt := NewIPT("iptables")
	ipt.ExecIgnoreError("-t", "nat", "-D", "OUTPUT", "-j", "CLASH_DNS_LOCAL")
	ipt.ExecIgnoreError("-t", "nat", "-D", "PREROUTING", "-j", "CLASH_DNS_EXTERNAL")
	ipt.ExecIgnoreError("-t", "nat", "-F", "CLASH_DNS_LOCAL")
	ipt.ExecIgnoreError("-t", "nat", "-X", "CLASH_DNS_LOCAL")
	ipt.ExecIgnoreError("-t", "nat", "-F", "CLASH_DNS_EXTERNAL")
	ipt.ExecIgnoreError("-t", "nat", "-X", "CLASH_DNS_EXTERNAL")

	ipt6 := NewIPT("ip6tables")
	ipt6.ExecIgnoreError("-D", "OUTPUT", "-p", "udp", "--dport", "53", "-j", "DROP")
}

