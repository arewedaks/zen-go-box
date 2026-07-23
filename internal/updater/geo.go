package updater

import (
	"log/slog"
	"path/filepath"

	"github.com/arewedaks/zengobox/internal/config"
)

// UpdateGeo mendownload database geoip.dat & geosite.dat terbaru
func UpdateGeo(cfg *config.Config) error {
	downloader := NewDownloader()

	geoipURL := "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.dat"
	geositeURL := "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat"

	geoipDest := filepath.Join(cfg.Paths.BoxDir, "geoip.dat")
	geositeDest := filepath.Join(cfg.Paths.BoxDir, "geosite.dat")

	slog.Info("Updating geoip.dat database...")
	if err := downloader.DownloadFile(geoipURL, geoipDest, false); err != nil {
		slog.Error("Failed to update geoip.dat", "error", err)
		return err
	}

	slog.Info("Updating geosite.dat database...")
	if err := downloader.DownloadFile(geositeURL, geositeDest, false); err != nil {
		slog.Error("Failed to update geosite.dat", "error", err)
		return err
	}

	slog.Info("Geo databases updated successfully.")
	return nil
}
