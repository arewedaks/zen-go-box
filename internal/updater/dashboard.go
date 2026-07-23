package updater

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/arewedaks/zengobox/internal/config"
)

// UpdateDashboard mendownload zashboard dashboard terbaru untuk UI control panel
func UpdateDashboard(cfg *config.Config) error {
	gh := NewGitHubClient()
	slog.Info("Checking for latest zashboard UI release...")
	rel, err := gh.FetchLatestRelease("Zephyruso", "zashboard")
	if err != nil {
		return fmt.Errorf("failed to fetch dashboard release: %w", err)
	}

	// Cari file asset zashboard.zip atau dist.zip
	var downloadURL string
	for _, asset := range rel.Assets {
		if filepath.Ext(asset.Name) == ".zip" {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("could not find dashboard zip asset in release")
	}

	tempDir := filepath.Join(cfg.Paths.RunDir, "dashboard_temp")
	_ = os.RemoveAll(tempDir)
	_ = os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, "dashboard.zip")

	downloader := NewDownloader()
	if err := downloader.DownloadFile(downloadURL, archivePath, false); err != nil {
		return fmt.Errorf("failed to download dashboard: %w", err)
	}

	slog.Info("Extracting dashboard UI...")
	destUIPath := filepath.Join(cfg.Paths.BoxDir, cfg.Core.BinName, "dashboard")
	_ = os.RemoveAll(destUIPath)
	_ = os.MkdirAll(destUIPath, 0755)

	if err := ExtractArchive(archivePath, destUIPath); err != nil {
		return fmt.Errorf("failed to extract dashboard: %w", err)
	}

	slog.Info("Dashboard UI updated successfully!", "path", destUIPath)
	return nil
}
