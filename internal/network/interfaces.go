package network

import (
	"log/slog"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/arewedaks/zengobox/internal/config"
	"github.com/arewedaks/zengobox/internal/netfilter"
)

type NetworkWatcher struct {
	watcher *fsnotify.Watcher
	cfg     *config.Config
	done    chan bool
}

func NewNetworkWatcher(cfg *config.Config) (*NetworkWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &NetworkWatcher{
		watcher: watcher,
		cfg:     cfg,
		done:    make(chan bool),
	}, nil
}

// Start mengawasi perubahan interfaces di /data/misc/net/
func (nw *NetworkWatcher) Start() {
	// Android menyimpan status interfaces di /data/misc/net/
	watchPath := "/data/misc/net"
	if _, err := os.Stat(watchPath); err != nil {
		// Fallback dev
		watchPath = "/tmp"
	}

	slog.Info("Starting network interface watcher", "path", watchPath)
	_ = nw.watcher.Add(watchPath)

	go func() {
		// Debounce timer untuk menghindari double-trigger saat interface naik
		var timer *time.Timer
		const debounceDuration = 3 * time.Second

		for {
			select {
			case event, ok := <-nw.watcher.Events:
				if !ok {
					return
				}
				// Cek modifikasi file
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					if timer != nil {
						timer.Stop()
					}
					timer = time.AfterFunc(debounceDuration, func() {
						nw.refreshIPRules()
					})
				}
			case err, ok := <-nw.watcher.Errors:
				if !ok {
					return
				}
				slog.Error("Network watcher error", "error", err)
			case <-nw.done:
				return
			}
		}
	}()
}

func (nw *NetworkWatcher) Stop() {
	nw.done <- true
	nw.watcher.Close()
	slog.Info("Network interface watcher stopped.")
}

func (nw *NetworkWatcher) refreshIPRules() {
	slog.Info("Network state changed. Refreshing local IP netfilter rules...")

	v4, v6, err := netfilter.GetLocalIPs()
	if err != nil {
		slog.Error("Failed to get local IPs for refresh", "error", err)
		return
	}

	ipt4 := netfilter.NewIPT("iptables")
	ipt6 := netfilter.NewIPT("ip6tables")

	// 1. Bersihkan rules chain LOCAL_IP di mangle table
	// Mangle table
	ipt4.ExecIgnoreError("-t", "mangle", "-F", "ZENNODE_EXTERNAL")
	ipt6.ExecIgnoreError("-t", "mangle", "-F", "ZENNODE_EXTERNAL")

	// Mencegah loopback dengan mengizinkan (RETURN) IP lokal di PREROUTING mangle
	for _, ip := range v4 {
		_ = ipt4.Exec("-t", "mangle", "-I", "ZENNODE_EXTERNAL", "-d", ip, "-j", "RETURN")
	}

	if nw.cfg.Network.IPv6 {
		for _, ip := range v6 {
			_ = ipt6.Exec("-t", "mangle", "-I", "ZENNODE_EXTERNAL", "-d", ip, "-j", "RETURN")
		}
	}

	// 2. Lakukan hal yang sama untuk nat table (jika menggunakan redirect/mixed)
	if nw.cfg.Network.Mode == "redirect" || nw.cfg.Network.Mode == "mixed" || nw.cfg.Network.Mode == "enhance" {
		ipt4.ExecIgnoreError("-t", "nat", "-F", "ZENNODE_EXTERNAL")
		ipt6.ExecIgnoreError("-t", "nat", "-F", "ZENNODE_EXTERNAL")

		for _, ip := range v4 {
			_ = ipt4.Exec("-t", "nat", "-I", "ZENNODE_EXTERNAL", "-d", ip, "-j", "RETURN")
		}

		if nw.cfg.Network.IPv6 {
			for _, ip := range v6 {
				_ = ipt6.Exec("-t", "nat", "-I", "ZENNODE_EXTERNAL", "-d", ip, "-j", "RETURN")
			}
		}
	}

	slog.Info("Local IP rules refreshed", "v4_count", len(v4), "v6_count", len(v6))
}
