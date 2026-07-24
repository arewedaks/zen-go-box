package updater

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/arewedaks/zen-go-box/internal/config"
)

// UpdateSubscription mengunduh konfigurasi server / proxies dari subscription URLs
func UpdateSubscription(cfg *config.Config) error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
	}

	// 1. Update Sing-box subscription jika dikonfigurasi
	if cfg.Subscription.SingboxURL != "" {
		slog.Info("Updating sing-box subscription...")
		data, err := fetchSubscriptionData(client, cfg.Subscription.SingboxURL)
		if err != nil {
			slog.Error("Failed to fetch sing-box subscription", "error", err)
		} else {
			destPath := filepath.Join(cfg.Paths.BoxDir, "sing-box", "config.json")
			_ = os.MkdirAll(filepath.Dir(destPath), 0755)
			if err := os.WriteFile(destPath, data, 0644); err != nil {
				slog.Error("Failed to write sing-box config", "error", err)
			} else {
				slog.Info("Sing-box subscription updated.")
			}
		}
	}

	// 2. Update Clash subscription jika dikonfigurasi
	if len(cfg.Subscription.ClashURLs) > 0 {
		slog.Info("Updating clash subscriptions...")
		for i, url := range cfg.Subscription.ClashURLs {
			data, err := fetchSubscriptionData(client, url)
			if err != nil {
				slog.Error("Failed to fetch clash subscription", "index", i, "error", err)
				continue
			}

			// Generate nama file subscription: subscription.yaml, subscription2.yaml, dst.
			name := "subscription.yaml"
			if i > 0 {
				name = fmt.Sprintf("subscription%d.yaml", i+1)
			}

			destPath := filepath.Join(cfg.Paths.BoxDir, "clash", name)
			_ = os.MkdirAll(filepath.Dir(destPath), 0755)

			// Custom Rules Injector
			if cfg.Subscription.InjectRules {
				rulesPath := filepath.Join(cfg.Paths.BoxDir, "clash", "rules.yaml")
				if rulesData, err := os.ReadFile(rulesPath); err == nil {
					data = append(data, []byte("\n# --- Injected Custom Rules ---\n")...)
					data = append(data, rulesData...)
				}
			}

			if err := os.WriteFile(destPath, data, 0644); err != nil {
				slog.Error("Failed to write clash config", "file", name, "error", err)
			} else {
				slog.Info("Clash subscription updated", "file", name)
			}
		}
	}

	return nil
}

func fetchSubscriptionData(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ClashMeta; BoxForRoot")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Jika data dibungkus base64 (format subsscribtion legasi), coba decode
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err == nil {
		return decoded, nil
	}

	return data, nil
}
