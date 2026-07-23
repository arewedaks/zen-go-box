package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/arewedaks/zen-go-box/internal/config"
	"github.com/arewedaks/zen-go-box/internal/core"
	"github.com/arewedaks/zen-go-box/internal/logger"
	"github.com/arewedaks/zen-go-box/internal/platform"
	"github.com/arewedaks/zen-go-box/internal/updater"
)

//go:embed configs/*
var defaultConfigs embed.FS

var (
	version = "dev"
	cfgFile string
	baseDir string
	cfg     *config.Config
	mgr     *core.Manager
)

var rootCmd = &cobra.Command{
	Use:   "zengobox",
	Short: "ZenGoBox daemon core transparent proxy manager",
	Long:  `ZenGoBox is a transparent proxy service daemon for rooted Android systems.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "setup" || cmd.Name() == "version" || cmd.Name() == "help" {
			return
		}
		initApp()
	},
}

func init() {
	if exePath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(exePath)
		if filepath.Base(execDir) == "bin" {
			baseDir = filepath.Dir(execDir)
		} else {
			baseDir = execDir
		}
	} else {
		baseDir = "/data/adb/zengobox"
	}
	defaultConfigPath := filepath.Join(baseDir, "zengobox.yaml")

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", defaultConfigPath, "path to configuration file")

	// Tambahkan command version langsung
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configCheckCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of zengobox",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("zengobox version: %s\n", version)
		fmt.Printf("OS/Arch:      android/%s\n", platform.GetArch())
		fmt.Printf("Root Env:     %s\n", platform.DetectRootEnv())
	},
}

var setupCmd = &cobra.Command{
	Use:   "setup [core]",
	Short: "Extract default configurations (clash, xray, sing-box, v2fly, hysteria, or all)",
	Run: func(cmd *cobra.Command, args []string) {
		target := "all"
		if len(args) > 0 {
			target = args[0]
		}
		fmt.Printf("Extracting %s configurations to %s ...\n", target, baseDir)
		extractEmbeddedConfigs(baseDir, target)
		
		geoTarget := target
		var loadedCfg *config.Config
		if geoTarget == "all" || geoTarget == "" {
			if cfg, err := config.Load(cfgFile); err == nil {
				loadedCfg = cfg
				geoTarget = cfg.Core.BinName
			} else {
				geoTarget = "clash" // fallback default
			}
		} else {
			loadedCfg, _ = config.Load(cfgFile)
		}
		
		fmt.Printf("Downloading geo databases for %s (this might take a while)...\n", geoTarget)
		_ = updater.UpdateGeo(baseDir, geoTarget)

		if loadedCfg != nil {
			fmt.Printf("Downloading core binary for %s...\n", loadedCfg.Core.BinName)
			_ = updater.UpdateKernel(loadedCfg.Core.BinName, loadedCfg)

			fmt.Println("Downloading dashboard UI...")
			_ = updater.UpdateDashboard(loadedCfg)
		}

		_ = platform.UpdateModulePropDescription("zengobox", "😴 System is Idle (Ready to start)")
		fmt.Println("Setup complete! You can now edit", cfgFile)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage zengobox configuration",
}

var configCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate zengobox configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cfg.Validate(); err != nil {
			slog.Error("Configuration validation failed", "error", err)
			os.Exit(1)
		}
		slog.Info("Configuration is valid.")
	},
}



func extractEmbeddedConfigs(dest string, targetCore string) {
	err := fs.WalkDir(defaultConfigs, "configs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel("configs", path)
		if relPath == "." || relPath == ".." || relPath == "" {
			return nil
		}

		// Jika pengguna hanya meminta core tertentu (misal: "clash"), kita lewati folder lain
		// Catatan: zengobox.yaml (file induk) akan selalu ikut diekstrak
		if relPath != "zengobox.yaml" && targetCore != "" && targetCore != "all" {
			// Mengecek apakah file/folder ini merupakan bagian dari targetCore (contoh: "clash/")
			if !strings.HasPrefix(relPath, targetCore) {
				return nil
			}
		}

		targetPath := filepath.Join(dest, relPath)

		if d.IsDir() {
			os.MkdirAll(targetPath, 0755)
			return nil
		}

		// Jika file belum ada di HP pengguna, kita salin template bawaan
		if _, statErr := os.Stat(targetPath); os.IsNotExist(statErr) {
			data, readErr := defaultConfigs.ReadFile(path)
			if readErr == nil {
				os.WriteFile(targetPath, data, 0644)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error extracting embedded configs: %v\n", err)
	}
}

func initApp() {
	// Load configuration
	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Init Logger
	if err := logger.InitLogger(cfg.Paths.LogDir, cfg.Log.Level, cfg.Log.MaxSize); err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}

	// Init Manager
	mgr = core.NewManager(cfg)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
