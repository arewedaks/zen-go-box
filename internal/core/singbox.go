package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arewedaks/zengobox/internal/config"
)

type SingboxInjector struct{}

func (s *SingboxInjector) Prepare(cfg *config.Config) error {
	srcPath := filepath.Join(cfg.Paths.BoxDir, "sing-box", "config.json")
	destPath := filepath.Join(cfg.Paths.BoxDir, "sing-box", "run.json")

	// 1. Baca config asli
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read sing-box config: %w", err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		// Abaikan error jika kosong, buat map baru
		rawMap = make(map[string]interface{})
	}
	if rawMap == nil {
		rawMap = make(map[string]interface{})
	}

	// 2. Modifikasi atau tambahkan "inbounds"
	inboundsRaw, exists := rawMap["inbounds"]
	var inbounds []interface{}
	if exists {
		if arr, ok := inboundsRaw.([]interface{}); ok {
			inbounds = arr
		}
	}

	// Buat inbound baru berdasarkan mode
	switch cfg.Network.Mode {
	case "tproxy":
		inbounds = append(inbounds, map[string]interface{}{
			"type":      "tproxy",
			"tag":       "tproxy-in",
			"listen":    "::",
			"listen_port": cfg.Network.TProxyPort,
			"sniff":     true,
		})
	case "redirect", "mixed", "enhance":
		inbounds = append(inbounds, map[string]interface{}{
			"type":      "redirect",
			"tag":       "redirect-in",
			"listen":    "::",
			"listen_port": cfg.Network.RedirPort,
			"sniff":     true,
		})
	case "tun":
		inbounds = append(inbounds, map[string]interface{}{
			"type":        "tun",
			"tag":         "tun-in",
			"interface_name": "tun0",
			"inet4_address": "172.19.0.1/30",
			"auto_route":   true,
			"strict_route": true,
			"sniff":        true,
		})
	}

	rawMap["inbounds"] = inbounds

	// 3. Tulis config modifikasi ke folder run
	outData, err := json.MarshalIndent(rawMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal modified sing-box config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(destPath, outData, 0644)
}
