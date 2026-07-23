package netfilter

import (
	"log/slog"
	"os/exec"
	"strconv"

	"github.com/arewedaks/zen-go-box/internal/config"
)

type EnhanceMode struct {
	cfg *config.Config
}

func (e *EnhanceMode) Name() string {
	return "enhance"
}

func (e *EnhanceMode) Setup(cfg *config.Config) error {
	e.cfg = cfg
	slog.Info("Setting up ENHANCE netfilter rules (Redirect TCP + TProxy UDP)...")

	CleanAllNetfilter()

	EnableIPForwarding(cfg.Network.IPv6)

	ipt4 := NewIPT("iptables")
	ipt6 := NewIPT("ip6tables")

	// Setup IP rule & route untuk TProxy UDP
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

	setupEnhance := func(ipt *IPT, isV6 bool) {
		// 1. NAT Table (untuk TCP redirect)
		ipt.ExecIgnoreError("-t", "nat", "-N", "ZENNODE_EXTERNAL")
		ipt.ExecIgnoreError("-t", "nat", "-N", "ZENNODE_LOCAL")
		ipt.ExecIgnoreError("-t", "nat", "-A", "PREROUTING", "-j", "ZENNODE_EXTERNAL")
		ipt.ExecIgnoreError("-t", "nat", "-A", "OUTPUT", "-j", "ZENNODE_LOCAL")

		// 2. Mangle Table (untuk UDP TProxy)
		ipt.ExecIgnoreError("-t", "mangle", "-N", "ZENNODE_EXTERNAL")
		ipt.ExecIgnoreError("-t", "mangle", "-N", "ZENNODE_LOCAL")
		ipt.ExecIgnoreError("-t", "mangle", "-A", "PREROUTING", "-j", "ZENNODE_EXTERNAL")
		ipt.ExecIgnoreError("-t", "mangle", "-A", "OUTPUT", "-j", "ZENNODE_LOCAL")

		intranet := IntranetV4
		if isV6 {
			intranet = IntranetV6
		}
		for _, cidr := range intranet {
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_EXTERNAL", "-d", cidr, "-j", "RETURN")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_EXTERNAL", "-d", cidr, "-j", "RETURN") // samakan untuk mangle bypass
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-d", cidr, "-j", "RETURN")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-d", cidr, "-j", "RETURN")
		}

		// Bypass self process
		uid, gid := ParseUserGroup(cfg.Process.UserGroup)
		ipt.ExecIgnoreError("-t", "nat", "-A", "ZENNODE_LOCAL", "-m", "owner", "--uid-owner", uid, "--gid-owner", gid, "-j", "RETURN")
		ipt.ExecIgnoreError("-t", "mangle", "-A", "ZENNODE_LOCAL", "-m", "owner", "--uid-owner", uid, "--gid-owner", gid, "-j", "RETURN")

		for _, g := range cfg.Proxy.GIDs {
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-m", "owner", "--gid-owner", g, "-j", "RETURN")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-m", "owner", "--gid-owner", g, "-j", "RETURN")
		}

		for _, ignore := range cfg.Proxy.APList.Ignore {
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-o", ignore, "-j", "RETURN")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-o", ignore, "-j", "RETURN")
		}

		// DNS Bypass (Biarkan NAT CLASH_DNS yang urus jika ClashDNSForward enable)
		if cfg.Network.ClashDNSForward && (cfg.Core.BinName == "clash" || cfg.Core.BinName == "hysteria") {
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_EXTERNAL", "-p", "tcp", "--dport", "53", "-j", "RETURN")
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_EXTERNAL", "-p", "udp", "--dport", "53", "-j", "RETURN")
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "tcp", "--dport", "53", "-j", "RETURN")
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "udp", "--dport", "53", "-j", "RETURN")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_EXTERNAL", "-p", "tcp", "--dport", "53", "-j", "RETURN")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_EXTERNAL", "-p", "udp", "--dport", "53", "-j", "RETURN")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "tcp", "--dport", "53", "-j", "RETURN")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "--dport", "53", "-j", "RETURN")
		}

		redirPortStr := strconv.Itoa(cfg.Network.RedirPort)
		tproxyPortStr := strconv.Itoa(cfg.Network.TProxyPort)

		_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_EXTERNAL", "-p", "tcp", "-i", "lo", "-j", "REDIRECT", "--to-ports", redirPortStr)
		_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_EXTERNAL", "-p", "udp", "-i", "lo", "-j", "TPROXY", "--on-port", tproxyPortStr, "--tproxy-mark", FWMark)

		for _, ap := range cfg.Proxy.APList.Allow {
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_EXTERNAL", "-p", "tcp", "-i", ap, "-j", "REDIRECT", "--to-ports", redirPortStr)
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_EXTERNAL", "-p", "udp", "-i", ap, "-j", "TPROXY", "--on-port", tproxyPortStr, "--tproxy-mark", FWMark)
		}

		// Proxy Mode (Blacklist / Whitelist)
		uids := ResolvePackagesUIDs(cfg.Proxy.Packages)
		if cfg.Proxy.Mode == "whitelist" || cfg.Proxy.Mode == "white" {
			if len(uids) == 0 {
				_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-j", "REDIRECT", "--to-ports", redirPortStr)
				_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "-j", "MARK", "--set-mark", FWMark)
			} else {
				for _, u := range uids {
					uStr := strconv.Itoa(u)
					_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-m", "owner", "--uid-owner", uStr, "-j", "REDIRECT", "--to-ports", redirPortStr)
					_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "-m", "owner", "--uid-owner", uStr, "-j", "MARK", "--set-mark", FWMark)
				}
				_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-m", "owner", "--uid-owner", "0", "-j", "REDIRECT", "--to-ports", redirPortStr)
				_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "-m", "owner", "--uid-owner", "0", "-j", "MARK", "--set-mark", FWMark)
				_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-m", "owner", "--uid-owner", "1052", "-j", "REDIRECT", "--to-ports", redirPortStr)
				_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "-m", "owner", "--uid-owner", "1052", "-j", "MARK", "--set-mark", FWMark)
			}
		} else {
			// Blacklist
			for _, u := range uids {
				uStr := strconv.Itoa(u)
				_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-m", "owner", "--uid-owner", uStr, "-j", "RETURN")
				_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-m", "owner", "--uid-owner", uStr, "-j", "RETURN")
			}
			_ = ipt.Exec("-t", "nat", "-A", "ZENNODE_LOCAL", "-p", "tcp", "-j", "REDIRECT", "--to-ports", redirPortStr)
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "-j", "MARK", "--set-mark", FWMark)
		}
		
		// QUIC Block
		if cfg.Network.QUICBlock {
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "--dport", "443", "-j", "REJECT")
			_ = ipt.Exec("-t", "mangle", "-A", "ZENNODE_LOCAL", "-p", "udp", "--dport", "80", "-j", "REJECT")
		}
	}

	setupEnhance(ipt4, false)
	if cfg.Network.IPv6 {
		setupEnhance(ipt6, true)
	}

	SetupDNSHijack(cfg, ipt4, ipt6)

	return nil
}

func (e *EnhanceMode) Teardown() error {
	CleanAllNetfilter()
	return nil
}
