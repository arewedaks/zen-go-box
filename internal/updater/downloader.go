package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Downloader struct {
	client *http.Client
}

func NewDownloader() *Downloader {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &Downloader{
		client: &http.Client{
			Timeout:   5 * time.Minute,
			Transport: tr,
		},
	}
}

// DownloadFile mengunduh url ke filepath tujuan, opsional menggunakan mirror ghproxy
func (d *Downloader) DownloadFile(url string, dest string, useMirror bool) error {
	if useMirror && strings.Contains(url, "github.com") {
		// Gunakan ghproxy mirror
		url = "https://mirror.ghproxy.com/" + url
	}

	slog.Info("Downloading...", "url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "zengobox-downloader")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	// Buat directory tujuan jika belum ada
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	// Track progress download
	total := resp.ContentLength
	counter := &writeCounter{total: total}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	return err
}

type writeCounter struct {
	total      int64
	downloaded int64
	lastUpdate time.Time
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.downloaded += int64(n)

	// Batasi log progress tiap 2 detik agar tidak spam
	if time.Since(wc.lastUpdate) > 2*time.Second {
		wc.lastUpdate = time.Now()
		if wc.total > 0 {
			pct := float64(wc.downloaded) / float64(wc.total) * 100
			slog.Info(fmt.Sprintf("Download progress: %.2f%% (%d/%d bytes)", pct, wc.downloaded, wc.total))
		} else {
			slog.Info(fmt.Sprintf("Downloaded %d bytes (unknown size)", wc.downloaded))
		}
	}
	return n, nil
}

// ExtractArchive meng-extract file zip, tar.gz, atau gz ke outputDir
func ExtractArchive(src string, destDir string) error {
	ext := strings.ToLower(filepath.Ext(src))
	if ext == ".zip" {
		return extractZip(src, destDir)
	} else if strings.HasSuffix(strings.ToLower(src), ".tar.gz") {
		return extractTarGz(src, destDir)
	} else if ext == ".gz" {
		return extractGzipOnly(src, destDir)
	}
	return fmt.Errorf("unsupported archive format: %s", ext)
}

func extractGzipOnly(src string, destDir string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	// Nama target file adalah nama file asli tanpa .gz
	fileName := filepath.Base(src)
	fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// Jika nama file temporer "archive", ganti dengan nama default "clash"
	if fileName == "archive" {
		fileName = "clash" // fallback
	}

	destPath := filepath.Join(destDir, fileName)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, gzr)
	return err
}

func extractZip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Cegah Zip Slip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(src string, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // Akhir tar archive
		}
		if err != nil {
			return err
		}

		fpath := filepath.Join(dest, header.Name)

		// Cegah Directory Traversal
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in tar: %s", fpath)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(fpath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(fpath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}
