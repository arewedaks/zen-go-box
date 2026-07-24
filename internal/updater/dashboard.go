package updater

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/arewedaks/zen-go-box/internal/config"
)

// UpdateDashboard mendownload dashboard terbaru untuk UI control panel
func UpdateDashboard(cfg *config.Config, dashboardURL string) error {
	if dashboardURL == "" || dashboardURL == "none" {
		if dashboardURL == "none" {
			slog.Info("Skipping dashboard installation.")
			return nil
		}
		// Default
		dashboardURL = "https://github.com/Zephyruso/zashboard/archive/gh-pages.zip"
	}

	slog.Info("Downloading dashboard UI...", "url", dashboardURL)

	tempDir := filepath.Join(cfg.Paths.RunDir, "dashboard_temp")
	_ = os.RemoveAll(tempDir)
	_ = os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, "dashboard.zip")

	downloader := NewDownloader()
	if err := downloader.DownloadFile(dashboardURL, archivePath, true); err != nil {
		return fmt.Errorf("failed to download dashboard: %w", err)
	}

	slog.Info("Extracting dashboard UI...")
	destUIPath := filepath.Join(cfg.Paths.BoxDir, cfg.Core.BinName, "dashboard")
	_ = os.RemoveAll(destUIPath)
	_ = os.MkdirAll(destUIPath, 0755)

	if err := ExtractArchive(archivePath, destUIPath); err != nil {
		return fmt.Errorf("failed to extract dashboard: %w", err)
	}

	// Remove top-level directory if the zip wraps everything inside one folder
	entries, _ := os.ReadDir(destUIPath)
	if len(entries) == 1 && entries[0].IsDir() {
		subDir := filepath.Join(destUIPath, entries[0].Name())
		subEntries, _ := os.ReadDir(subDir)
		for _, e := range subEntries {
			_ = os.Rename(filepath.Join(subDir, e.Name()), filepath.Join(destUIPath, e.Name()))
		}
		_ = os.Remove(subDir)
	}

	slog.Info("Dashboard UI updated successfully!", "path", destUIPath)
	return nil
}
