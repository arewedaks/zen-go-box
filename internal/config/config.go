package config

import (
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Core         CoreConfig         `yaml:"core"`
	Network      NetworkConfig      `yaml:"network"`
	Proxy        ProxyConfig        `yaml:"proxy"`
	Process      ProcessConfig      `yaml:"process"`
	Cgroup       CgroupConfig       `yaml:"cgroup"`
	Paths        PathsConfig        `yaml:"paths"`
	Subscription SubscriptionConfig `yaml:"subscription"`
	Geo          GeoConfig          `yaml:"geo"`
	Schedule     ScheduleConfig     `yaml:"schedule"`
	Wifi         WifiConfig         `yaml:"wifi"`
	Log          LogConfig          `yaml:"log"`
	NeedsSetup   bool               `yaml:"-"`
}

type CoreConfig struct {
	BinName     string            `yaml:"bin_name"`
	BinList     []string          `yaml:"bin_list"`
	ClashOption string            `yaml:"clash_option"`
	ConfigNames map[string]string `yaml:"config_names"`
}

type NetworkConfig struct {
	Mode            string `yaml:"mode"`
	TProxyPort      int    `yaml:"tproxy_port"`
	RedirPort       int    `yaml:"redir_port"`
	IPv6            bool   `yaml:"ipv6"`
	QUICBlock       bool   `yaml:"quic_block"`
	ClashDNSForward bool   `yaml:"clash_dns_forward"`
	ClashDNSPort    int    `yaml:"clash_dns_port"`
}

type ProxyConfig struct {
	Mode     string   `yaml:"mode"`
	Packages []string `yaml:"packages"`
	GIDs     []string `yaml:"gids"`
	APList   APConfig `yaml:"ap_list"`
}

type APConfig struct {
	Allow  []string `yaml:"allow"`
	Ignore []string `yaml:"ignore"`
}

type ProcessConfig struct {
	UserGroup      string        `yaml:"user_group"`
	MaxRestarts    int           `yaml:"max_restarts"`
	RestartWindow  time.Duration `yaml:"-"`
	RestartWindowStr string      `yaml:"restart_window"`
	RestartBackoff time.Duration `yaml:"-"`
	RestartBackoffStr string     `yaml:"restart_backoff"`
}

type CgroupConfig struct {
	MemCG  MemCGConfig  `yaml:"memcg"`
	CPUSet CPUSetConfig `yaml:"cpuset"`
	BlkIO  BlkIOConfig  `yaml:"blkio"`
}

type MemCGConfig struct {
	Enabled bool   `yaml:"enabled"`
	Limit   string `yaml:"limit"`
}

type CPUSetConfig struct {
	Enabled bool `yaml:"enabled"`
}

type BlkIOConfig struct {
	Enabled bool `yaml:"enabled"`
}

type PathsConfig struct {
	BoxDir string `yaml:"box_dir"`
	BinDir string `yaml:"bin_dir"`
	RunDir string `yaml:"run_dir"`
	LogDir string `yaml:"log_dir"`
}

type SubscriptionConfig struct {
	ClashURLs   []string `yaml:"clash_urls"`
	SingboxURL  string   `yaml:"singbox_url"`
	Renew       bool     `yaml:"renew"`
	InjectRules bool     `yaml:"inject_rules"`
}

type GeoConfig struct {
	AutoUpdate bool `yaml:"auto_update"`
}

type ScheduleConfig struct {
	Enabled            bool   `yaml:"enabled"`
	Cron               string `yaml:"cron"`
	UpdateGeo          bool   `yaml:"update_geo"`
	UpdateSubscription bool   `yaml:"update_subscription"`
}

type WifiConfig struct {
	Enabled         bool     `yaml:"enabled"`
	UseOnWifi       bool     `yaml:"use_on_wifi"`
	UseOnDisconnect bool     `yaml:"use_on_disconnect"`
	SSIDMatching    bool     `yaml:"ssid_matching"`
	SSIDMode        string   `yaml:"ssid_mode"`
	SSIDList        []string `yaml:"ssid_list"`
}

type LogConfig struct {
	Level   string `yaml:"level"`
	MaxSize string `yaml:"max_size"`
	Toast   bool   `yaml:"toast"`
}

func DefaultConfig() *Config {
	var baseDir string
	if exePath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(exePath)
		if filepath.Base(execDir) == "bin" {
			baseDir = filepath.Dir(execDir)
		} else {
			baseDir = execDir
		}
	} else {
		baseDir = "/data/adb/zengobox"
	}

	return &Config{
		Core: CoreConfig{
			BinName:     "clash",
			BinList:     []string{"sing-box", "clash", "xray", "v2fly", "hysteria"},
			ClashOption: "mihomo",
			ConfigNames: map[string]string{
				"clash":    "config.yaml",
				"sing-box": "config.json",
				"xray":     "config.json",
				"v2fly":    "config.json",
				"hysteria": "config.yaml",
			},
		},
		Network: NetworkConfig{
			Mode:            "tproxy",
			TProxyPort:      9898,
			RedirPort:       9797,
			IPv6:            false,
			QUICBlock:       false,
			ClashDNSForward: true,
			ClashDNSPort:    1053,
		},
		Proxy: ProxyConfig{
			Mode: "blacklist",
			APList: APConfig{
				Allow:  []string{"ap+", "wlan+", "rndis+", "swlan+", "ncm+", "eth+"},
				Ignore: []string{},
			},
		},
		Process: ProcessConfig{
			UserGroup:         "root:net_admin",
			MaxRestarts:       5,
			RestartWindowStr:  "5m",
			RestartBackoffStr: "3s",
		},
		Cgroup: CgroupConfig{
			MemCG: MemCGConfig{
				Enabled: false,
				Limit:   "100M",
			},
		},
		Paths: PathsConfig{
			BoxDir: baseDir,
			BinDir: filepath.Join(baseDir, "bin"),
			RunDir: filepath.Join(baseDir, "run"),
			LogDir: filepath.Join(baseDir, "run"),
		},
		Log: LogConfig{
			Level:   "info",
			MaxSize: "1M",
			Toast:   true,
		},
	}
}
