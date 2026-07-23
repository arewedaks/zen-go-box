package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type RootEnv string

const (
	EnvMagisk  RootEnv = "magisk"
	EnvKernelSU RootEnv = "kernelsu"
	EnvAPatch   RootEnv = "apatch"
	EnvUnknown  RootEnv = "unknown"
)

// DetectRootEnv mendeteksi root manager aktif
func DetectRootEnv() RootEnv {
	// Cek KernelSU
	if _, err := os.Stat("/data/adb/ksu"); err == nil {
		return EnvKernelSU
	}
	// Cek APatch
	if _, err := os.Stat("/data/adb/ap"); err == nil {
		return EnvAPatch
	}
	// Cek Magisk (biasanya ada folder /data/adb/magisk)
	if _, err := os.Stat("/data/adb/magisk"); err == nil {
		return EnvMagisk
	}
	return EnvUnknown
}

// GetModuleDir mengembalikan module directory based on root env
func GetModuleDir(modID string) string {
	// Di Magisk, KernelSU, dan APatch modul diletakkan di /data/adb/modules/
	return filepath.Join("/data/adb/modules", modID)
}

// IsModuleEnabled memeriksa apakah modul tidak sedang di-disable (.disable file)
func IsModuleEnabled(modID string) bool {
	modDir := GetModuleDir(modID)
	disableFile := filepath.Join(modDir, "disable")
	if _, err := os.Stat(disableFile); err == nil {
		return false
	}
	return true
}

// UpdateModulePropDescription modifies the module.prop description field dynamically.
func UpdateModulePropDescription(modID string, prefixMsg string) error {
	propFile := filepath.Join(GetModuleDir(modID), "module.prop")
	content, err := os.ReadFile(propFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	modified := false
	
	// Regex to match an existing bracket prefix (e.g., "[ 10:45 | ⭕ clash running ] ")
	re := regexp.MustCompile(`^description=(\[.*?\]\s*)?(.*)$`)

	for i, line := range lines {
		if strings.HasPrefix(line, "description=") {
			matches := re.FindStringSubmatch(line)
			if len(matches) == 3 {
				originalDesc := matches[2]
				lines[i] = fmt.Sprintf("description=[ %s | %s ] %s", time.Now().Format("15:04"), prefixMsg, originalDesc)
				modified = true
				break
			}
		}
	}

	if modified {
		return os.WriteFile(propFile, []byte(strings.Join(lines, "\n")), 0644)
	}
	return nil
}
