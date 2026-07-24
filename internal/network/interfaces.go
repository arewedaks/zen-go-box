package network

import (
	"log/slog"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/arewedaks/zen-go-box/internal/config"
	"github.com/arewedaks/zen-go-box/internal/core"
	"github.com/arewedaks/zen-go-box/internal/netfilter"
	"net/http"
	"os/exec"
)

type NetworkWatcher struct {
	watcher *fsnotify.Watcher
	cfg     *config.Config
	mgr     *core.Manager
	done    chan bool
}

func NewNetworkWatcher(cfg *config.Config, mgr *core.Manager) (*NetworkWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &NetworkWatcher{
		watcher: watcher,
		cfg:     cfg,
		mgr:     mgr,
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
						slog.Info("Network event detected. Triggering rules refresh...")
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
	if nw.cfg.Wifi.Enabled {
		shouldRun := EvaluateWifiState(nw.cfg)
		isRunning, _ := nw.mgr.Status()
		if !shouldRun && isRunning {
			slog.Info("Smart Wi-Fi: Conditions not met. Stopping Proxy...")
			nw.mgr.Stop()
			return
		} else if shouldRun && !isRunning {
			slog.Info("Smart Wi-Fi: Conditions met. Starting Proxy...")
			nw.mgr.Start()
		}
	}

	// Only refresh rules if the proxy is actually running
	isRunning, _ := nw.mgr.Status()
	if !isRunning {
		return
	}

	slog.Info("Network state changed. Refreshing all netfilter rules...")

	mode, err := netfilter.GetMode(nw.cfg)
	if err != nil {
		slog.Error("Failed to get netfilter mode for refresh", "error", err)
		return
	}

	if err := mode.Setup(nw.cfg); err != nil {
		slog.Error("Failed to re-apply netfilter rules", "error", err)
	} else {
		slog.Info("Netfilter rules successfully refreshed")
	}

	// 1. Connection Reset (Robust Connection Tracking)
	nw.flushConnections()

	// 2. Secondary Flush (Tertunda)
	// Untuk mengatasi lambatnya negosiasi Data Seluler (LTE/5G) yang kadang memakan waktu 5-8 detik
	// saat di-ON-kan, kita jadwalkan tembakan flush kedua agar Mihomo tidak bengong setelah sinyal benar-benar masuk.
	time.AfterFunc(6*time.Second, func() {
		slog.Info("Executing secondary delayed flush for slow cellular negotiations...")
		nw.flushConnections()
	})
}

func (nw *NetworkWatcher) flushConnections() {
	// Membunuh stale sockets agar aplikasi langsung reconnect tanpa menunggu timeout
	slog.Info("Flushing stale connections to force immediate reconnect...")
	
	// Flush conntrack table agar kernel melupakan state lama
	_ = exec.Command("conntrack", "-F").Run()
	
	// Jika tersedia di sistem, gunakan ss -K untuk mengirim SOCK_DESTROY (TCP RST)
	// ss -K -t state established
	_ = exec.Command("ss", "-K", "-t", "state", "established").Run()

	// Mihomo/Clash REST API Flush
	slog.Info("Sending REST API signal to flush proxy core connections...")
	req, err := http.NewRequest("DELETE", "http://127.0.0.1:9090/connections", nil)
	if err == nil {
		if nw.cfg.Core.APISecret != "" {
			req.Header.Set("Authorization", "Bearer "+nw.cfg.Core.APISecret)
		}
		client := &http.Client{Timeout: 2 * time.Second}
		_, _ = client.Do(req)
	}
}
