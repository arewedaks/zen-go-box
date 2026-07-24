package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/arewedaks/zen-go-box/internal/logger"
	"github.com/arewedaks/zen-go-box/internal/netfilter"
	"github.com/arewedaks/zen-go-box/internal/network"
	"github.com/arewedaks/zen-go-box/internal/updater"
	"github.com/arewedaks/zen-go-box/internal/web"
)

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(toggleCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(iptablesCmd)
	iptablesCmd.AddCommand(iptablesEnableCmd)
	iptablesCmd.AddCommand(iptablesDisableCmd)
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(updateKernelCmd)
	updateCmd.AddCommand(updateGeoCmd)
	updateCmd.AddCommand(updateSubscriptionCmd)
	updateCmd.AddCommand(updateDashboardCmd)
	updateCmd.AddCommand(updateAllCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the proxy service core",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Starting service...")
		if err := mgr.Start(); err != nil {
			slog.Error("Failed to start service", "error", err)
			if cfg.Log.Toast {
				logger.Toast(fmt.Sprintf("Failed to start service: %v", err))
			}
			os.Exit(1)
		}
		// Aktifkan iptables secara otomatis setelah start
		mode, err := netfilter.GetMode(cfg)
		if err == nil {
			_ = mode.Setup(cfg)
		}
		if cfg.Log.Toast {
			logger.Toast(fmt.Sprintf("Service started: %s", cfg.Core.BinName))
		}
		fmt.Println("Service started successfully.")
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the proxy service core and clean up iptables",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Stopping service...")
		// Bersihkan netfilter
		netfilter.CleanAllNetfilter()
		if err := mgr.Stop(); err != nil {
			slog.Error("Failed to stop service", "error", err)
			os.Exit(1)
		}
		if cfg.Log.Toast {
			logger.Toast("Service stopped")
		}
		fmt.Println("Service stopped successfully.")
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the proxy service core",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Restarting service...")
		netfilter.CleanAllNetfilter()
		_ = mgr.Stop()
		if err := mgr.Start(); err != nil {
			slog.Error("Failed to restart service", "error", err)
			os.Exit(1)
		}
		mode, err := netfilter.GetMode(cfg)
		if err == nil {
			_ = mode.Setup(cfg)
		}
		fmt.Println("Service restarted successfully.")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of running proxy service core",
	Run: func(cmd *cobra.Command, args []string) {
		running, pid := mgr.Status()
		if running {
			fmt.Printf("zengobox status: active (pid: %d)\n", pid)
		} else {
			fmt.Println("zengobox status: inactive")
		}
	},
}

var toggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle proxy service core on/off",
	Run: func(cmd *cobra.Command, args []string) {
		running, _ := mgr.Status()
		if running {
			netfilter.CleanAllNetfilter()
			_ = mgr.Stop()
			fmt.Println("Service toggled OFF.")
		} else {
			if err := mgr.Start(); err != nil {
				slog.Error("Failed to start service during toggle", "error", err)
				os.Exit(1)
			}
			mode, err := netfilter.GetMode(cfg)
			if err == nil {
				_ = mode.Setup(cfg)
			}
			fmt.Println("Service toggled ON.")
		}
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run zengobox in daemon watch mode (handles boot and network changes)",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Running daemon watch mode...")

		// 1. Start proxy core service
		_ = mgr.Start()
		mode, err := netfilter.GetMode(cfg)
		if err == nil {
			_ = mode.Setup(cfg)
		}

		// 2. Start Network Watcher (dynamic anti-loopback local IPs refresh)
		netWatcher, err := network.NewNetworkWatcher(cfg)
		if err == nil {
			netWatcher.Start()
			defer netWatcher.Stop()
		}

		// 3. Start Module Status Watcher (Magisk disable file trigger)
		modWatcher, err := network.NewModuleWatcher(cfg, mgr)
		if err == nil {
			modWatcher.Start()
			defer modWatcher.Stop()
		}

		// 4. Start Zashboard Web Server
		web.StartServer(mgr)

		select {} // Keep running
	},
}

var iptablesCmd = &cobra.Command{
	Use:   "iptables",
	Short: "Manage transparent proxy iptables rules",
}

var iptablesEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable netfilter transparent proxy rules",
	Run: func(cmd *cobra.Command, args []string) {
		mode, err := netfilter.GetMode(cfg)
		if err != nil {
			slog.Error("Failed to get netfilter mode", "error", err)
			os.Exit(1)
		}
		if err := mode.Setup(cfg); err != nil {
			slog.Error("Failed to enable iptables rules", "error", err)
			os.Exit(1)
		}
		fmt.Println("Iptables rules enabled.")
	},
}

var iptablesDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable netfilter transparent proxy rules",
	Run: func(cmd *cobra.Command, args []string) {
		netfilter.CleanAllNetfilter()
		fmt.Println("Iptables rules disabled.")
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update core components (binary, geodata, subscriptions, dashboard)",
}

var updateKernelCmd = &cobra.Command{
	Use:   "kernel [name]",
	Short: "Update proxy core kernel binary",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := cfg.Core.BinName
		if len(args) > 0 {
			name = args[0]
		}
		slog.Info("Starting kernel update...", "kernel", name)
		if err := updater.UpdateKernel(name, cfg); err != nil {
			slog.Error("Kernel update failed", "error", err)
			os.Exit(1)
		}
	},
}

var updateGeoCmd = &cobra.Command{
	Use:   "geo",
	Short: "Update GeoIP & GeoSite databases",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Starting geodata update...")
		if err := updater.UpdateGeo(cfg.Paths.BoxDir, cfg.Core.BinName); err != nil {
			slog.Error("Geodata update failed", "error", err)
			os.Exit(1)
		}
	},
}

var updateSubscriptionCmd = &cobra.Command{
	Use:   "subscription",
	Short: "Update Clash / sing-box subscriptions",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Starting subscription update...")
		if err := updater.UpdateSubscription(cfg); err != nil {
			slog.Error("Subscription update failed", "error", err)
			os.Exit(1)
		}
	},
}

var updateDashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Update dashboard web UI panel",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Starting dashboard update...")
		if err := updater.UpdateDashboard(cfg); err != nil {
			slog.Error("Dashboard update failed", "error", err)
			os.Exit(1)
		}
	},
}

var updateAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Update kernel, geodata, subscriptions and dashboard",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Starting full update...")
		_ = updater.UpdateKernel(cfg.Core.BinName, cfg)
		_ = updater.UpdateGeo(cfg.Paths.BoxDir, cfg.Core.BinName)
		_ = updater.UpdateSubscription(cfg)
		_ = updater.UpdateDashboard(cfg)
		slog.Info("Full update completed.")
	},
}
