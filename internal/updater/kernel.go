package updater

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/arewedaks/zengobox/internal/config"
)

type KernelRepo struct {
	Owner string
	Repo  string
}

// Map kernel name → repositori aslinya di GitHub
var repos = map[string]KernelRepo{
	"sing-box": {Owner: "SagerNet", Repo: "sing-box"},
	"clash":    {Owner: "MetaCubeX", Repo: "mihomo"},
	"xray":     {Owner: "XTLS", Repo: "Xray-core"},
	"v2fly":    {Owner: "v2fly", Repo: "v2ray-core"},
	"hysteria": {Owner: "apernet", Repo: "hysteria"},
}

// UpdateKernel mengecek release, mendownload, dan memasang biner proxy core baru
func UpdateKernel(name string, cfg *config.Config) error {
	repoInfo, ok := repos[name]
	if !ok {
		return fmt.Errorf("unknown kernel name: %s", name)
	}

	slog.Info("Checking for latest release on GitHub...", "repo", repoInfo.Owner+"/"+repoInfo.Repo)
	gh := NewGitHubClient()
	rel, err := gh.FetchLatestRelease(repoInfo.Owner, repoInfo.Repo)
	if err != nil {
		return fmt.Errorf("failed to fetch release info: %w", err)
	}

	slog.Info("Found latest version", "tag", rel.TagName)

	downloadURL, err := FindMatchingAsset(rel, name)
	if err != nil {
		return fmt.Errorf("failed to find matching asset: %w", err)
	}

	tempDir := filepath.Join(cfg.Paths.RunDir, "update_temp")
	_ = os.RemoveAll(tempDir)
	_ = os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	tempFile := filepath.Join(tempDir, "archive"+filepath.Ext(downloadURL))

	downloader := NewDownloader()
	// Gunakan mirror jika diaktifkan (kita anggap true untuk mempercepat di INA)
	if err := downloader.DownloadFile(downloadURL, tempFile, false); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	slog.Info("Extracting archive...")
	if err := ExtractArchive(tempFile, tempDir); err != nil {
		// Jika bukan format archive, mungkin biner mentah (.gz gzip atau raw executable)
		// Tapi release standar sing-box/clash selalu dikemas .tar.gz atau .zip
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Cari file executable hasil extract di dalam folder tempDir
	var targetBin string
	var largestSize int64
	var fallbackBin string

	searchName := name
	if name == "clash" {
		searchName = "mihomo"
	}

	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Jika nama file cocok persis
		if info.Name() == name {
			targetBin = path
			return filepath.SkipAll // stop search
		}

		// Simpan fallback file terbesar yang bukan archive .zip/.gz/.tar.gz
		ext := filepath.Ext(path)
		if ext != ".gz" && ext != ".zip" && ext != ".tar" {
			if info.Size() > largestSize {
				largestSize = info.Size()
				fallbackBin = path
			}
		}

		// Jika nama file mengandung kata kunci kernel asli (misal mihomo)
		if strings.Contains(strings.ToLower(info.Name()), strings.ToLower(searchName)) {
			targetBin = path
		}

		return nil
	})

	if targetBin == "" && fallbackBin != "" {
		targetBin = fallbackBin
	}

	if targetBin == "" {
		return fmt.Errorf("could not find binary '%s' in extracted files", name)
	}

	// Salin biner baru ke folder bin
	destBinPath := filepath.Join(cfg.Paths.BinDir, name)

	if name == "clash" {
		xclashDir := filepath.Join(cfg.Paths.BinDir, "xclash")
		_ = os.MkdirAll(xclashDir, 0755)
		destBinPath = filepath.Join(xclashDir, "mihomo")
	}

	_ = os.MkdirAll(filepath.Dir(destBinPath), 0755)

	// Backup biner lama (jika ada) sebelum overwrite
	backupPath := destBinPath + ".bak"
	_ = os.Rename(destBinPath, backupPath)

	// Pindahkan biner baru
	if err := os.Rename(targetBin, destBinPath); err != nil {
		// Restore backup jika gagal
		_ = os.Rename(backupPath, destBinPath)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Set executable permission
	_ = os.Chmod(destBinPath, 0755)
	_ = os.Remove(backupPath) // Hapus backup setelah sukses

	// Buat symlink langsung ke bin/clash
	if name == "clash" {
		clashSymlink := filepath.Join(cfg.Paths.BinDir, "clash")
		_ = os.Remove(clashSymlink)
		if err := os.Symlink(destBinPath, clashSymlink); err != nil {
			slog.Warn("Failed to create symlink for clash", "error", err)
		} else {
			slog.Info("Symlink updated successfully", "symlink", clashSymlink)
		}
	}

	slog.Info("Kernel updated successfully!", "kernel", name, "path", destBinPath)
	return nil
}
