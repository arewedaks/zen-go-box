package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load loading Config dari file YAML
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg.NeedsSetup = true
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Make it portable: Override loaded YAML paths with actual binary location
	if exePath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(exePath)
		baseDir := execDir
		if filepath.Base(execDir) == "bin" {
			baseDir = filepath.Dir(execDir)
		}
		if cfg.Paths.BoxDir == "" || cfg.Paths.BoxDir == "/data/adb/zengobox" {
			cfg.Paths.BoxDir = baseDir
		}
		if cfg.Paths.BinDir == "" || cfg.Paths.BinDir == "/data/adb/zengobox/bin" {
			cfg.Paths.BinDir = filepath.Join(baseDir, "bin")
		}
		if cfg.Paths.RunDir == "" || cfg.Paths.RunDir == "/data/adb/zengobox/run" {
			cfg.Paths.RunDir = filepath.Join(baseDir, "run")
		}
		if cfg.Paths.LogDir == "" || cfg.Paths.LogDir == "/data/adb/zengobox/run" {
			cfg.Paths.LogDir = filepath.Join(baseDir, "run")
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Validate if core binary exists, otherwise mark as NeedsSetup
	binPath := filepath.Join(cfg.Paths.BinDir, cfg.Core.BinName)
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		cfg.NeedsSetup = true
	}

	return cfg, nil
}

// Save menulis Config ke file YAML
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file for writing: %w", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}



// CopyEmbedDefaultConfig menulis template default jika tidak ditemukan config existing
func CopyEmbedDefaultConfig(embedData string, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.WriteString(out, embedData)
	return err
}
