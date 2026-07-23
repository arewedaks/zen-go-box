package platform

import (
	"os"
	"path/filepath"
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
