package netfilter

import (
	"log/slog"
	"os/exec"
	"strconv"

	"github.com/arewedaks/zen-go-box/internal/config"
)

type TProxyMode struct {
	cfg *config.Config
}

func (t *TProxyMode) Name() string {
	return "tproxy"
}

func (t *TProxyMode) Setup(cfg *config.Config) error {
	t.cfg = cfg
	slog.Info("Setting up TPROXY netfilter rules...")

	// 1. Bersihkan rules lama
	CleanAllNetfilter()

	EnableIPForwarding(cfg.Network.IPv6)

	ipt4 := NewIPT("iptables")
	ipt6 := NewIPT("ip6tables")

	// 2. Tambahkan custom routing table untuk TPROXY
	// ip rule add fwmark 16777216/16777216 table 2024
	// ip route add local default dev lo table 2024
	if err := exec.Command("ip", "rule", "add", "fwmark", FWMark, "table", TableID, "pref", RulePref).Run(); err != nil {
		slog.Warn("Failed to add ip rule v4", "error", err)
	}
	if err := exec.Command("ip", "route", "add", "local", "default", "dev", "lo", "table", TableID).Run(); err != nil {
		slog.Warn("Failed to add ip route v4", "error", err)
	}

	if cfg.Network.IPv6 {
		_ = exec.Command("ip", "-6", "rule", "add", "fwmark", FWMark, "table", TableID, "pref", RulePref).Run()
		_ = exec.Command("ip", "-6", "route", "add", "local", "default", "dev", "lo", "table", TableID).Run()
	}

	// 3. Buat chain ZENNODE_EXTERNAL & ZENNODE_LOCAL di mangle table
	setupMangle := func(ipt *IPT, isV6 bool) {
		ipt.ExecIgnoreError("-t", "mangle", "-N", "ZENNODE_EXTERNAL")
		ipt.ExecIgnoreError("-t", "mangle", "-N", "ZENNODE_LOCAL")

		// Sambungkan chain ke PREROUTING dan OUTPUT
		ipt.ExecIgnoreError("-t", "mangle", "-A", "PREROUTING", "-j", "ZENNODE_EXTERNAL")
		ipt.ExecIgnoreError("-t", "mangle", "-A", "OUTPUT", "-j", "ZENNODE_LOCAL")

		// Mangle PREROUTING (ZENNODE_EXTERNAL)
		// Bypass intranet CIDR
		intranet := IntranetV4
		if isV6 {
			intranet = IntranetV6
		}
		for _, cidr := range intranet {
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_EXTERNAL", "-d", cidr, "-j", "RETURN")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-d", cidr, "-j", "RETURN")
		}

		// Bypass self process (owner match)
		uid, gid := ParseUserGroup(cfg.Process.UserGroup)
		ipt.ExecIgnoreError("-t", "mangle", "-A", "ZENNODE_LOCAL", "-m", "owner", "--uid-owner", uid, "--gid-owner", gid, "-j", "RETURN")

		// Filter by GID if defined
		for _, g := range cfg.Proxy.GIDs {
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-m", "owner", "--gid-owner", g, "-j", "RETURN")
		}

		// Handle Ignore Out List
		for _, ignore := range cfg.Proxy.APList.Ignore {
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-o", ignore, "-j", "RETURN")
		}

		// DNS Bypass in Mangle (Biarkan NAT yang urus jika ClashDNSForward enable)
		if cfg.Network.ClashDNSForward && (cfg.Core.BinName == "clash" || cfg.Core.BinName == "hysteria") {
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_EXTERNAL", "-p", "tcp", "--dport", "53", "-j", "RETURN")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_EXTERNAL", "-p", "udp", "--dport", "53", "-j", "RETURN")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "tcp", "--dport", "53", "-j", "RETURN")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "--dport", "53", "-j", "RETURN")
		}

		// TPROXY target rules untuk TCP & UDP
		portStr := strconv.Itoa(cfg.Network.TProxyPort)
		_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_EXTERNAL", "-p", "tcp", "-i", "lo", "-j", "TPROXY", "--on-port", portStr, "--tproxy-mark", FWMark)
		_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_EXTERNAL", "-p", "udp", "-i", "lo", "-j", "TPROXY", "--on-port", portStr, "--tproxy-mark", FWMark)

		for _, ap := range cfg.Proxy.APList.Allow {
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_EXTERNAL", "-p", "tcp", "-i", ap, "-j", "TPROXY", "--on-port", portStr, "--tproxy-mark", FWMark)
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_EXTERNAL", "-p", "udp", "-i", ap, "-j", "TPROXY", "--on-port", portStr, "--tproxy-mark", FWMark)
		}

		// Proxy Mode (Blacklist / Whitelist)
		uids := ResolvePackagesUIDs(cfg.Proxy.Packages)
		if cfg.Proxy.Mode == "whitelist" || cfg.Proxy.Mode == "white" {
			if len(uids) == 0 {
				_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-j", "MARK", "--set-mark", FWMark)
				_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "-j", "MARK", "--set-mark", FWMark)
			} else {
				for _, u := range uids {
					uStr := strconv.Itoa(u)
					_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-m", "owner", "--uid-owner", uStr, "-j", "MARK", "--set-mark", FWMark)
					_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "-m", "owner", "--uid-owner", uStr, "-j", "MARK", "--set-mark", FWMark)
				}
				_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-m", "owner", "--uid-owner", "0", "-j", "MARK", "--set-mark", FWMark)
				_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "-m", "owner", "--uid-owner", "0", "-j", "MARK", "--set-mark", FWMark)
				_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-m", "owner", "--uid-owner", "1052", "-j", "MARK", "--set-mark", FWMark)
				_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "-m", "owner", "--uid-owner", "1052", "-j", "MARK", "--set-mark", FWMark)
			}
		} else {
			// Blacklist
			for _, u := range uids {
				uStr := strconv.Itoa(u)
				_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-m", "owner", "--uid-owner", uStr, "-j", "RETURN")
			}
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-j", "MARK", "--set-mark", FWMark)
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "-j", "MARK", "--set-mark", FWMark)
		}
		
		// QUIC Block
		if cfg.Network.QUICBlock {
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "--dport", "443", "-j", "REJECT")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "--dport", "80", "-j", "REJECT")
		}
	}

	setupMangle(ipt4, false)
	if cfg.Network.IPv6 {
		setupMangle(ipt6, true)
	}

	SetupDNSHijack(cfg, ipt4, ipt6)

	return nil
}

func (t *TProxyMode) Teardown() error {
	CleanAllNetfilter()
	return nil
}
