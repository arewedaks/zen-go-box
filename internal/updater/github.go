package updater

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/arewedaks/zengobox/internal/platform"
)

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type GitHubClient struct {
	client *http.Client
}

func NewGitHubClient() *GitHubClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &GitHubClient{
		client: &http.Client{
			Timeout:   15 * time.Second,
			Transport: tr,
		},
	}
}

// FetchLatestRelease mengambil informasi release terbaru dari GitHub repo
func (g *GitHubClient) FetchLatestRelease(owner, repo string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "zengobox-updater")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status: %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}

	return &rel, nil
}

var archMap = map[string][]string{
	"arm64": {"arm64", "aarch64", "armv8"},
	"arm":   {"armv7", "armv7l", "arm32", "arm"},
	"386":   {"386", "i386", "i686", "x86"},
	"amd64": {"amd64", "x86_64", "x64"},
}

// FindMatchingAsset mencocokkan nama file aset GitHub dengan arsitektur perangkat
func FindMatchingAsset(rel *Release, kernelName string) (string, error) {
	deviceArch := platform.GetArch()
	keywords, ok := archMap[deviceArch]
	if !ok {
		return "", fmt.Errorf("unsupported architecture: %s", deviceArch)
	}

	searchName := kernelName
	if kernelName == "clash" {
		searchName = "mihomo"
	}

	for _, asset := range rel.Assets {
		nameLower := strings.ToLower(asset.Name)

		// Filter berdasarkan nama kernel (sing-box, mihomo, dll)
		if !strings.Contains(nameLower, strings.ToLower(searchName)) {
			continue
		}

		// Cari kecocokan arsitektur
		for _, kw := range keywords {
			if strings.Contains(nameLower, kw) {
				return asset.BrowserDownloadURL, nil
			}
		}
	}

	return "", fmt.Errorf("no matching asset found for kernel %s on arch %s", kernelName, deviceArch)
}
