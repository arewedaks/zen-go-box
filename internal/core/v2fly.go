package core

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/arewedaks/zen-go-box/internal/config"
)

type V2flyInjector struct{}

func (v *V2flyInjector) Prepare(cfg *config.Config) error {
	confName := cfg.Core.ConfigNames["v2fly"]
	if confName == "" {
		confName = "config.json"
	}
	srcPath := filepath.Join(cfg.Paths.BoxDir, "v2fly", confName)
	destPath := filepath.Join(cfg.Paths.BoxDir, "v2fly", "run.json")

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		raw = make(map[string]interface{})
	}
	if raw == nil {
		raw = make(map[string]interface{})
	}

	if inboundsRaw, ok := raw["inbounds"]; ok {
		if inbounds, ok := inboundsRaw.([]interface{}); ok {
			for _, ibRaw := range inbounds {
				if ib, ok := ibRaw.(map[string]interface{}); ok {
					protocol, _ := ib["protocol"].(string)
					if protocol == "dokodemo-door" {
						ib["port"] = cfg.Network.TProxyPort
					}
				}
			}
		}
	}
	
	out, _ := json.MarshalIndent(raw, "", "  ")
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(destPath, out, 0644)
}
