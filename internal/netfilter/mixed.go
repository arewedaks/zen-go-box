package netfilter

import (
	"log/slog"
	"strconv"

	"github.com/arewedaks/zen-go-box/internal/config"
)

type MixedMode struct {
	cfg *config.Config
}

func (m *MixedMode) Name() string {
	return "mixed"
}

func (m *MixedMode) Setup(cfg *config.Config) error {
	m.cfg = cfg
	slog.Info("Setting up MIXED netfilter rules (Redirect TCP + Tun UDP)...")

	CleanAllNetfilter()

	EnableIPForwarding(cfg.Network.IPv6)

	ipt4 := NewIPT("iptables")
	ipt6 := NewIPT("ip6tables")

	setupMixed := func(ipt *IPT, isV6 bool) {
		// 1. Setup Redirect TCP di NAT Table
		ipt.ExecIgnoreError("-t", "nat", "-N", "ZENNODE_EXTERNAL")
		ipt.ExecIgnoreError("-t", "nat", "-N", "ZENNODE_LOCAL")
		ipt.ExecIgnoreError("-t", "nat", "-A", "PREROUTING", "-j", "ZENNODE_EXTERNAL")
		ipt.ExecIgnoreError("-t", "nat", "-A", "OUTPUT", "-j", "ZENNODE_LOCAL")

		intranet := IntranetV4
		if isV6 {
			intranet = IntranetV6
		}
		for _, cidr := range intranet {
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_EXTERNAL", "-d", cidr, "-j", "RETURN")
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-d", cidr, "-j", "RETURN")
		}

		// Bypass self process
		uid, gid := ParseUserGroup(cfg.Process.UserGroup)
		ipt.ExecIgnoreError("-t", "nat", "-A", "ZENNODE_LOCAL", "-m", "owner", "--uid-owner", uid, "--gid-owner", gid, "-j", "RETURN")

		// Bypass based on GIDs
		for _, g := range cfg.Proxy.GIDs {
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-m", "owner", "--gid-owner", g, "-j", "RETURN")
		}

		// Handle Ignore Out List
		for _, ignore := range cfg.Proxy.APList.Ignore {
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-o", ignore, "-j", "RETURN")
		}

		// DNS Bypass (Biarkan NAT CLASH_DNS yang urus)
		if cfg.Network.ClashDNSForward && (cfg.Core.BinName == "clash" || cfg.Core.BinName == "hysteria") {
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_EXTERNAL", "-p", "udp", "--dport", "53", "-j", "RETURN")
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "udp", "--dport", "53", "-j", "RETURN")
		}

		portStr := strconv.Itoa(cfg.Network.RedirPort)
		_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_EXTERNAL", "-p", "tcp", "-i", "lo", "-j", "REDIRECT", "--to-ports", portStr)
		for _, ap := range cfg.Proxy.APList.Allow {
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_EXTERNAL", "-p", "tcp", "-i", ap, "-j", "REDIRECT", "--to-ports", portStr)
		}

		// Proxy Mode (Blacklist / Whitelist)
		uids := ResolvePackagesUIDs(cfg.Proxy.Packages)
		if cfg.Proxy.Mode == "whitelist" || cfg.Proxy.Mode == "white" {
			if len(uids) == 0 {
				_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-j", "REDIRECT", "--to-ports", portStr)
			} else {
				for _, u := range uids {
					uStr := strconv.Itoa(u)
					_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-m", "owner", "--uid-owner", uStr, "-j", "REDIRECT", "--to-ports", portStr)
				}
				_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-m", "owner", "--uid-owner", "0", "-j", "REDIRECT", "--to-ports", portStr)
				_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-m", "owner", "--uid-owner", "1052", "-j", "REDIRECT", "--to-ports", portStr)
			}
		} else {
			// Blacklist
			for _, u := range uids {
				uStr := strconv.Itoa(u)
				_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-m", "owner", "--uid-owner", uStr, "-j", "RETURN")
			}
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-j", "REDIRECT", "--to-ports", portStr)
		}

		// 2. Setup TUN Forwarding untuk UDP
		_ = ipt.Exec("-A", "FORWARD", "-o", "tun0", "-j", "ACCEPT")
		_ = ipt.Exec("-A", "FORWARD", "-i", "tun0", "-j", "ACCEPT")
	}

	setupMixed(ipt4, false)
	if cfg.Network.IPv6 {
		setupMixed(ipt6, true)
	}

	SetupDNSHijack(cfg, ipt4, ipt6)

	return nil
}

func (m *MixedMode) Teardown() error {
	CleanAllNetfilter()
	return nil
}
