package netfilter

import (
	"bufio"
	"os"
	"strings"
)

// ResolvePackagesUIDs meresolusi list package names → list UID numerik
func ResolvePackagesUIDs(packages []string) []int {
	var uids []int
	packagesFile := "/data/system/packages.list"

	// Dev fallback
	if _, err := os.Stat(packagesFile); err != nil {
		packagesFile = "test_packages.list" // untuk dev/testing lokal
	}

	file, err := os.Open(packagesFile)
	if err != nil {
		return uids
	}
	defer file.Close()

	// Simpan dalam map agar pencarian O(1)
	pkgSet := make(map[string]bool)
	for _, p := range packages {
		// support format "user_id:package_name" atau "package_name"
		if strings.Contains(p, ":") {
			parts := strings.SplitN(p, ":", 2)
			pkgSet[parts[1]] = true
		} else {
			pkgSet[p] = true
		}
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pkgName := fields[0]
			uidStr := fields[1]

			if pkgSet[pkgName] {
				// Parse uid
				var uid int
				// Kita parse manual atau panggil helper
				n := 0
				for _, c := range uidStr {
					if c >= '0' && c <= '9' {
						n = n*10 + int(c-'0')
					} else {
						break
					}
				}
				uid = n
				uids = append(uids, uid)
			}
		}
	}

	return uids
}
