package netfilter

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/arewedaks/zengobox/internal/config"
)

type Mode interface {
	Setup(cfg *config.Config) error
	Teardown() error
	Name() string
}

func GetMode(cfg *config.Config) (Mode, error) {
	switch cfg.Network.Mode {
	case "tproxy":
		return &TProxyMode{cfg: cfg}, nil
	case "redirect":
		return &RedirectMode{cfg: cfg}, nil
	case "tun":
		return &TunMode{cfg: cfg}, nil
	case "mixed":
		return &MixedMode{cfg: cfg}, nil
	case "enhance":
		return &EnhanceMode{cfg: cfg}, nil
	default:
		return nil, fmt.Errorf("unsupported network mode: %s", cfg.Network.Mode)
	}
}

// CleanAllNetfilter membersihkan semua chains custom zengobox di iptables & ip6tables
func CleanAllNetfilter() {
	CleanDNSHijack()
	ipt4 := NewIPT("iptables")
	ipt6 := NewIPT("ip6tables")

	// Helper untuk flush + delete chain di table nat
	cleanTable := func(ipt *IPT, table string, chains []string) {
		for _, chain := range chains {
			ipt.ExecIgnoreError("-t", table, "-F", chain)
			ipt.ExecIgnoreError("-t", table, "-D", "PREROUTING", "-j", chain)
			ipt.ExecIgnoreError("-t", table, "-D", "OUTPUT", "-j", chain)
			ipt.ExecIgnoreError("-t", table, "-X", chain)
		}
	}

	// Bersihkan chain buatan kita
	chains := []string{"ZENNODE_EXTERNAL", "ZENNODE_LOCAL", "CLASH_DNS_EXTERNAL", "CLASH_DNS_LOCAL"}
	cleanTable(ipt4, "nat", chains)
	cleanTable(ipt6, "nat", chains)

	cleanTable(ipt4, "mangle", chains)
	cleanTable(ipt6, "mangle", chains)

	// Bersihkan rules ip rule & ip route jika ada
	// Mencegah error jika rules sudah terhapus
	execIgnoreError("ip", "rule", "del", "fwmark", FWMark, "table", TableID)
	execIgnoreError("ip", "route", "del", "local", "default", "dev", "lo", "table", TableID)
	execIgnoreError("ip", "-6", "rule", "del", "fwmark", FWMark, "table", TableID)
	execIgnoreError("ip", "-6", "route", "del", "local", "default", "dev", "lo", "table", TableID)
}

func execIgnoreError(name string, args ...string) {
	cmd := exec.Command(name, args...)
	_ = cmd.Run()
}

func ParseUserGroup(ug string) (string, string) {
	parts := strings.Split(ug, ":")
	if len(parts) != 2 {
		return "0", "3005"
	}

	uStr := parts[0]
	gStr := parts[1]

	uid := uStr
	if uStr == "root" {
		uid = "0"
	}
	
	gid := gStr
	if gStr == "net_admin" {
		gid = "3005"
	}

	return uid, gid
}
