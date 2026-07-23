package core

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/arewedaks/zengobox/internal/config"
)

type HysteriaInjector struct{}

func (h *HysteriaInjector) Prepare(cfg *config.Config) error {
	srcPath := filepath.Join(cfg.Paths.BoxDir, "hysteria", "config.yaml")
	destPath := filepath.Join(cfg.Paths.BoxDir, "hysteria", "run.yaml")

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		raw = make(map[string]interface{})
	}
	if raw == nil {
		raw = make(map[string]interface{})
	}

	tproxyPort := fmt.Sprintf(":%d", cfg.Network.TProxyPort)
	redirPort := fmt.Sprintf(":%d", cfg.Network.RedirPort)

	if tp, ok := raw["tcpTProxy"].(map[string]interface{}); ok {
		tp["listen"] = tproxyPort
	}
	if up, ok := raw["udpTProxy"].(map[string]interface{}); ok {
		up["listen"] = tproxyPort
	}
	if tr, ok := raw["tcpRedirect"].(map[string]interface{}); ok {
		tr["listen"] = redirPort
	}

	out, _ := yaml.Marshal(raw)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(destPath, out, 0644)
}
