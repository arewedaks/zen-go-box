package network

import (
	"bytes"
	"os/exec"
	"regexp"
	"strings"

	"github.com/arewedaks/zen-go-box/internal/config"
)

// GetWiFiSSID mengambil SSID WiFi aktif dengan 3 metode fallback
func GetWiFiSSID() (string, bool) {
	// Method 1: cmd wifi status
	if ssid, ok := getSSIDViaCmdWifi(); ok {
		return ssid, true
	}

	// Method 2: iwconfig wlan0
	if ssid, ok := getSSIDViaIwconfig(); ok {
		return ssid, true
	}

	// Method 3: dumpsys wifi
	if ssid, ok := getSSIDViaDumpsys(); ok {
		return ssid, true
	}

	return "", false
}

func getSSIDViaCmdWifi() (string, bool) {
	cmd := exec.Command("cmd", "wifi", "status")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", false
	}

	re := regexp.MustCompile(`SSID:\s*"?([^"\n\r]+)"?`)
	match := re.FindStringSubmatch(out.String())
	if len(match) >= 2 {
		return strings.TrimSpace(match[1]), true
	}
	return "", false
}

func getSSIDViaIwconfig() (string, bool) {
	// Cari interface wlan0
	cmd := exec.Command("iwconfig", "wlan0")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", false
	}

	re := regexp.MustCompile(`ESSID:\s*"?([^"\n\r]+)"?`)
	match := re.FindStringSubmatch(out.String())
	if len(match) >= 2 {
		ssid := strings.TrimSpace(match[1])
		if ssid != "off/any" && ssid != "" {
			return ssid, true
		}
	}
	return "", false
}

func getSSIDViaDumpsys() (string, bool) {
	cmd := exec.Command("dumpsys", "wifi")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", false
	}

	// Cari pola WifiInfo SSID
	re := regexp.MustCompile(`mWifiInfo\s+SSID:\s*"?([^"\n\r]+)"?`)
	match := re.FindStringSubmatch(out.String())
	if len(match) >= 2 {
		ssid := strings.TrimSpace(match[1])
		if ssid != "<unknown ssid>" && ssid != "" {
			return ssid, true
		}
	}
	return "", false
}

// EvaluateWifiState returns true if the proxy should run based on the current Wi-Fi conditions.
// It also logs the decision.
func EvaluateWifiState(cfg *config.Config) bool {
	if !cfg.Wifi.Enabled {
		return true // Always run if Smart Wi-Fi is disabled
	}

	ssid, isConnected := GetWiFiSSID()

	if !isConnected {
		// Device is NOT connected to Wi-Fi
		if cfg.Wifi.UseOnDisconnect {
			return true
		}
		return false
	}

	// Device IS connected to Wi-Fi
	if !cfg.Wifi.SSIDMatching {
		return cfg.Wifi.UseOnWifi
	}

	// Match SSID
	matched := false
	for _, allowed := range cfg.Wifi.SSIDList {
		if strings.TrimSpace(allowed) == ssid {
			matched = true
			break
		}
	}

	if cfg.Wifi.SSIDMode == "whitelist" {
		if matched {
			return true
		}
		return false
	} else {
		// blacklist
		if matched {
			return false
		}
		return true
	}
}
