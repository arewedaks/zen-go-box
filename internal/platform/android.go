package platform

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// GetArch mengembalikan tipe arsitektur perangkat (arm64, arm, amd64, 386)
func GetArch() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		return "x86_64"
	}
	if arch == "386" {
		return "x86"
	}
	// Fallback ke command-line uname jika arm
	if arch == "arm" {
		cmd := exec.Command("uname", "-m")
		if out, err := cmd.Output(); err == nil {
			m := strings.TrimSpace(string(out))
			if strings.HasPrefix(m, "armv8") || strings.Contains(m, "64") {
				return "arm64"
			}
		}
		return "armv7"
	}
	return arch
}

// GetAndroidAPILevel mendeteksi level API Android menggunakan getprop ro.build.version.sdk
func GetAndroidAPILevel() int {
	cmd := exec.Command("getprop", "ro.build.version.sdk")
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	sdkStr := strings.TrimSpace(string(out))
	var sdk int
	_, _ = fmtSscan(sdkStr, &sdk)
	return sdk
}

func fmtSscan(str string, out *int) (int, error) {
	n := 0
	for _, c := range str {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	*out = n
	return 1, nil
}

// IsBusyboxInstalled mengecek apakah busybox terinstall di system path
func IsBusyboxInstalled() bool {
	_, err := exec.LookPath("busybox")
	return err == nil
}

// ParsePackagesList membaca UID dari package name dari data system packages.list Android
func ParsePackagesList(packagesPath string, targetPkg string) (int, error) {
	file, err := os.Open(packagesPath)
	if err != nil {
		return -1, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pkg := fields[0]
			uidStr := fields[1]
			if pkg == targetPkg {
				var uid int
				if _, err := fmtSscan(uidStr, &uid); err == nil {
					return uid, nil
				}
			}
		}
	}
	return -1, os.ErrNotExist
}
