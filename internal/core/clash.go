package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arewedaks/zen-go-box/internal/config"
	"gopkg.in/yaml.v3"
)

type ClashInjector struct{}

func (c *ClashInjector) Prepare(cfg *config.Config) error {
	confName := cfg.Core.ConfigNames["clash"]
	if confName == "" {
		confName = "config.yaml"
	}
	srcPath := filepath.Join(cfg.Paths.BoxDir, "clash", confName)
	destPath := filepath.Join(cfg.Paths.BoxDir, "clash", "run.yaml")

	// 1. Baca config asli
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read clash config: %w", err)
	}

	var rawMap map[string]interface{}
	if err := yaml.Unmarshal(data, &rawMap); err != nil {
		return fmt.Errorf("failed to unmarshal clash config: %w", err)
	}
	if rawMap == nil {
		rawMap = make(map[string]interface{})
	}

	// 2. Modifikasi port dan controller
	rawMap["directory"] = filepath.Join(cfg.Paths.BoxDir, "clash") // Paksa set working directory ke /data/adb/zengobox/clash
	rawMap["tproxy-port"] = cfg.Network.TProxyPort
	rawMap["redir-port"] = cfg.Network.RedirPort
	rawMap["mixed-port"] = 0 // Disable mixed port jika ada demi stabilitas tproxy/redir terpisah
	rawMap["mode"] = "rule"

	// Tentukan external-controller agar bisa diakses dari jaringan lokal (bind all interfaces)
	rawMap["external-controller"] = ":9090"
	rawMap["external-ui"] = filepath.Join(cfg.Paths.BoxDir, "clash", "dashboard")

	// Paksa injeksi header CORS untuk akses dashboard dari browser eksternal (PC Desktop)
	headersMap := map[string]interface{}{
		"Access-Control-Allow-Origin": "*",
	}
	rawMap["headers"] = headersMap

	// Modifikasi "tun" config jika mode tun diaktifkan
	if cfg.Network.Mode == "tun" || cfg.Network.Mode == "mixed" {
		tunMap := map[string]interface{}{
			"enable":              true,
			"stack":               "system",
			"device":              "tun0",
			"auto-route":          true,
			"auto-detect-interface": true,
		}

		if len(cfg.Proxy.Packages) > 0 {
			if cfg.Proxy.Mode == "whitelist" || cfg.Proxy.Mode == "white" {
				tunMap["include-package"] = cfg.Proxy.Packages
				tunMap["exclude-package"] = []string{}
			} else {
				tunMap["include-package"] = []string{}
				tunMap["exclude-package"] = cfg.Proxy.Packages
			}
		} else {
			tunMap["include-package"] = []string{}
			tunMap["exclude-package"] = []string{}
		}

		rawMap["tun"] = tunMap
	} else {
		// Disable tun jika mode non-tun
		if tunRaw, ok := rawMap["tun"]; ok {
			if tm, ok := tunRaw.(map[string]interface{}); ok {
				tm["enable"] = false
			}
		}
	}

	// 3. Pastikan DNS listen port diatur jika DNS Hijack aktif
	if cfg.Network.ClashDNSForward {
		dnsPortStr := fmt.Sprintf("0.0.0.0:%d", cfg.Network.ClashDNSPort)
		if dnsRaw, ok := rawMap["dns"]; ok {
			if dm, ok := dnsRaw.(map[string]interface{}); ok {
				dm["listen"] = dnsPortStr
			}
		} else {
			rawMap["dns"] = map[string]interface{}{
				"enable": true,
				"listen": dnsPortStr,
				"enhanced-mode": "fake-ip",
			}
		}
	}

	// 4. Tulis modified yaml
	outData, err := yaml.Marshal(rawMap)
	if err != nil {
		return fmt.Errorf("failed to marshal modified clash config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(destPath, outData, 0644)
}
