package network

import (
	"bytes"
	"os/exec"
	"regexp"
	"strings"
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
