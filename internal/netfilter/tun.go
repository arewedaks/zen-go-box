package netfilter

import (
	"log/slog"

	"github.com/arewedaks/zen-go-box/internal/config"
)

type TunMode struct {
	cfg *config.Config
}

func (t *TunMode) Name() string {
	return "tun"
}

func (t *TunMode) Setup(cfg *config.Config) error {
	t.cfg = cfg
	slog.Info("Setting up TUN netfilter forwarding rules...")

	CleanAllNetfilter()

	EnableIPForwarding(cfg.Network.IPv6)

	ipt4 := NewIPT("iptables")
	ipt6 := NewIPT("ip6tables")

	setupTunForward := func(ipt *IPT) {
		// Forward rules untuk interface tun0
		_ = ipt.Exec("-A", "FORWARD", "-o", "tun0", "-j", "ACCEPT")
		_ = ipt.Exec("-A", "FORWARD", "-i", "tun0", "-j", "ACCEPT")
	}

	setupTunForward(ipt4)
	if cfg.Network.IPv6 {
		setupTunForward(ipt6)
	}

	return nil
}

func (t *TunMode) Teardown() error {
	CleanAllNetfilter()
	return nil
}
