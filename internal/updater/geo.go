package updater

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// UpdateGeo mendownload database geoip.dat & geosite.dat terbaru ke folder yang tepat
func UpdateGeo(baseDir string, targetCore string) error {
	downloader := NewDownloader()

	geoipURL := "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.dat"
	geositeURL := "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat"

	cores := []string{"clash", "xray", "v2fly", "sing-box", "hysteria"}
	if targetCore != "all" && targetCore != "" {
		cores = []string{targetCore}
	}

	for _, core := range cores {
		// Xray dan v2fly membaca dari baseDir via env var, sedangkan clash/sing-box dari foldernya sendiri
		destDir := baseDir
		if core == "clash" || core == "sing-box" || core == "hysteria" {
			destDir = filepath.Join(baseDir, core)
		}

		_ = os.MkdirAll(destDir, 0755)
		geoipDest := filepath.Join(destDir, "geoip.dat")
		geositeDest := filepath.Join(destDir, "geosite.dat")

		slog.Info("Updating geo databases...", "target", core)
		if err := downloader.DownloadFile(geoipURL, geoipDest, true); err != nil {
			return fmt.Errorf("failed to download geoip.dat: %w", err)
		}
		if err := downloader.DownloadFile(geositeURL, geositeDest, true); err != nil {
			return fmt.Errorf("failed to download geosite.dat: %w", err)
		}
	}

	slog.Info("Geo databases downloaded successfully!")
	return nil
}
