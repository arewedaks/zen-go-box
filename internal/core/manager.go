package core

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/arewedaks/zen-go-box/internal/cgroup"
	"github.com/arewedaks/zen-go-box/internal/config"
	"github.com/arewedaks/zen-go-box/internal/platform"
	"github.com/arewedaks/zen-go-box/internal/updater"
)

type Manager struct {
	cfg        *config.Config
	cmd        *exec.Cmd
	cancelFunc context.CancelFunc
	running    bool
	scheduler  *cron.Cron
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg: cfg,
	}
}

// Start menjalankan proxy core yang dikonfigurasi
func (m *Manager) Start() error {
	if m.running {
		return fmt.Errorf("service is already running")
	}

	// 0. Pastikan tidak ada proses zombie dari proxy sebelumnya yang masih menyangkut
	_ = m.Stop() // Coba hentikan via box.pid
	_ = exec.Command("killall", "-9", m.cfg.Core.BinName).Run() // Sapu bersih zombie process

	// Tampilkan informasi konfigurasi yang digunakan secara ringkas (1 baris ke samping)
	slog.Info("Configuration Loaded",
		"core", fmt.Sprintf("%s (%s)", m.cfg.Core.BinName, m.cfg.Core.ClashOption),
		"net", fmt.Sprintf("%s (v6:%v)", m.cfg.Network.Mode, m.cfg.Network.IPv6),
		"proxy", m.cfg.Proxy.Mode,
		"port", m.cfg.Network.TProxyPort,
	)

	// 1. Dapatkan injector berdasarkan bin_name
	var injector Injector
	switch m.cfg.Core.BinName {
	case "sing-box":
		injector = &SingboxInjector{}
	case "clash":
		injector = &ClashInjector{}
		m.prepareXClash()
	case "xray":
		injector = &XrayInjector{}
	case "v2fly":
		injector = &V2flyInjector{}
	case "hysteria":
		injector = &HysteriaInjector{}
	default:
		return fmt.Errorf("unsupported bin_name: %s", m.cfg.Core.BinName)
	}

	// 2. Prepare configuration
	if err := injector.Prepare(m.cfg); err != nil {
		return fmt.Errorf("failed to prepare configuration: %w", err)
	}

	// 3. Bangun exec command
	binPath := filepath.Join(m.cfg.Paths.BinDir, m.cfg.Core.BinName)
	var args []string

	switch m.cfg.Core.BinName {
	case "sing-box":
		args = []string{"run", "-D", filepath.Join(m.cfg.Paths.BoxDir, "sing-box"), "-c", filepath.Join(m.cfg.Paths.BoxDir, "sing-box", "run.json")}
	case "clash":
		args = []string{"-d", filepath.Join(m.cfg.Paths.BoxDir, "clash"), "-f", filepath.Join(m.cfg.Paths.BoxDir, "clash", "run.yaml")}
	case "xray":
		args = []string{"run", "-confdir", filepath.Join(m.cfg.Paths.BoxDir, "xray"), "-config", filepath.Join(m.cfg.Paths.BoxDir, "xray", "run.json")}
	case "v2fly":
		args = []string{"run", "-d", filepath.Join(m.cfg.Paths.BoxDir, "v2fly"), "-config", filepath.Join(m.cfg.Paths.BoxDir, "v2fly", "run.json")}
	case "hysteria":
		args = []string{"server", "-c", filepath.Join(m.cfg.Paths.BoxDir, "hysteria", "run.yaml")}
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	cmd := exec.CommandContext(ctx, binPath, args...)

	// Atur working directory agar relative path dari rule-provider terbaca di /data/adb/zengobox/clash atau /data/adb/zengobox/sing-box
	cmd.Dir = filepath.Join(m.cfg.Paths.BoxDir, m.cfg.Core.BinName)

	// Redirect output ke log file kernel (misal: /data/adb/zengobox/run/sing-box.log)
	logFileName := fmt.Sprintf("%s.log", m.cfg.Core.BinName)
	logFilePath := filepath.Join(m.cfg.Paths.LogDir, logFileName)
	_ = os.MkdirAll(filepath.Dir(logFilePath), 0755)
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	} else {
		slog.Warn("Failed to redirect kernel log, output will be discarded", "error", err)
	}

	// Parse user:group (e.g. root:net_admin atau 0:3005)
	uid, gid, err := parseUserGroup(m.cfg.Process.UserGroup)
	if err != nil {
		slog.Warn("Failed to parse user_group, falling back to default root:net_admin (0:3005)", "error", err)
		uid = 0
		gid = 3005 // net_admin
	}

	// Atur credential proses agar run as user target (setuidgid equivalent)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}

	// Set environment variables tambahan untuk asset location (xray/v2fly) dan Memory Optimization
	cmd.Env = os.Environ()
	// Optimasi RAM & Baterai untuk proxy core berbasis Go (Mihomo/Sing-box)
	// GOMEMLIMIT menjaga RAM agar tidak bocor (Soft Cap), sedangkan GOGC default (100) mencegah CPU bekerja terlalu keras (Hemat Baterai).
	cmd.Env = append(cmd.Env, "GOMEMLIMIT=100MiB")

	if m.cfg.Core.BinName == "xray" {
		cmd.Env = append(cmd.Env, "XRAY_LOCATION_ASSET="+m.cfg.Paths.BoxDir)
	} else if m.cfg.Core.BinName == "v2fly" {
		cmd.Env = append(cmd.Env, "V2RAY_LOCATION_ASSET="+m.cfg.Paths.BoxDir)
	}

	// 4. Jalankan process
	if err := cmd.Start(); err != nil {
		if logFile != nil {
			logFile.Close()
		}
		_ = platform.UpdateModulePropDescription("zengobox", fmt.Sprintf("💔 %s failed to start! (Check Logs)", m.cfg.Core.BinName))
		return fmt.Errorf("failed to start proxy core process: %w", err)
	}

	m.cmd = cmd
	m.running = true

	// 5. Tulis file PID
	pidPath := filepath.Join(m.cfg.Paths.RunDir, "box.pid")
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		slog.Warn("Failed to write PID file", "error", err)
	}

	slog.Info("Proxy core started successfully", "bin", m.cfg.Core.BinName, "pid", cmd.Process.Pid)
	_ = platform.UpdateModulePropDescription("zengobox", fmt.Sprintf("⚡ %s is currently ACTIVE & routing", m.cfg.Core.BinName))

	// 6. Terapkan Cgroup Limits jika diaktifkan
	if err := cgroup.Apply(cmd.Process.Pid, m.cfg); err != nil {
		slog.Warn("Failed to apply cgroup resource limits", "error", err)
	}

	// 7. Setup & Start Cron Scheduler untuk auto-update
	if m.cfg.Schedule.Enabled {
		m.scheduler = cron.New()
		// Tambahkan job update geo
		if m.cfg.Schedule.UpdateGeo {
			_, _ = m.scheduler.AddFunc(m.cfg.Schedule.Cron, func() {
				slog.Info("[Cron] Triggering auto geo update...")
				_ = updater.UpdateGeo(m.cfg.Paths.BoxDir, m.cfg.Core.BinName)
			})
		}
		// Tambahkan job update subscription
		if m.cfg.Schedule.UpdateSubscription {
			_, _ = m.scheduler.AddFunc(m.cfg.Schedule.Cron, func() {
				slog.Info("[Cron] Triggering auto subscription update...")
				_ = updater.UpdateSubscription(m.cfg)
			})
		}
		m.scheduler.Start()
		slog.Info("Cron scheduler started", "cron", m.cfg.Schedule.Cron)
	}

	// Watcher goroutine untuk crash recovery
	go m.watchProcess(logFile)

	return nil
}

// Stop menghentikan proses proxy core secara graceful
func (m *Manager) Stop() error {
	slog.Info("Stopping proxy core...")

	// Stop Cron Scheduler jika berjalan
	if m.scheduler != nil {
		m.scheduler.Stop()
		m.scheduler = nil
	}

	pidPath := filepath.Join(m.cfg.Paths.RunDir, "box.pid")

	// 1. Jika kita memiliki control instance (same process)
	if m.running && m.cmd != nil {
		_ = os.Remove(pidPath)
		if m.cancelFunc != nil {
			m.cancelFunc()
		}
		_ = m.cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() {
			done <- m.cmd.Wait()
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = m.cmd.Process.Kill()
		}
		m.running = false
		m.cmd = nil
		slog.Info("Proxy core stopped successfully")
		_ = platform.UpdateModulePropDescription("zengobox", fmt.Sprintf("🛑 %s proxy is OFFLINE", m.cfg.Core.BinName))
		return nil
	}

	// 2. Fallback: Hentikan proses via file PID (different CLI process)
	data, err := os.ReadFile(pidPath)
	if err == nil {
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err == nil && pid > 0 {
			_ = os.Remove(pidPath)
			process, err := os.FindProcess(pid)
			if err == nil {
				// Kirim SIGTERM
				_ = process.Signal(syscall.SIGTERM)
				// Tunggu sebentar lalu pastikan ter-kill
				time.Sleep(1 * time.Second)
				_ = process.Signal(syscall.Signal(0))
				if err == nil {
					// Jika masih bernafas, SIGKILL
					_ = process.Signal(syscall.SIGKILL)
				}
			}
		}
	}

	m.running = false
	slog.Info("Proxy core stopped successfully")
	_ = platform.UpdateModulePropDescription("zengobox", fmt.Sprintf("🛑 %s proxy is OFFLINE", m.cfg.Core.BinName))
	return nil
}

// Status memeriksa apakah proses berjalan dan mengembalikan info
func (m *Manager) Status() (bool, int) {
	// 1. Cek memory state jika kita berada di instance proses yang sama
	if m.running && m.cmd != nil && m.cmd.Process != nil {
		err := m.cmd.Process.Signal(syscall.Signal(0))
		if err == nil {
			return true, m.cmd.Process.Pid
		}
	}

	// 2. Fallback check dari file PID (berguna saat dipanggil dari CLI baru)
	pidPath := filepath.Join(m.cfg.Paths.RunDir, "box.pid")
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return false, 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return false, 0
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0
	}

	// Cek status via signal 0
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true, pid
	}

	return false, 0
}

func (m *Manager) prepareXClash() {
	if m.cfg.Core.ClashOption == "" {
		m.cfg.Core.ClashOption = "mihomo"
	}
	binDir := m.cfg.Paths.BinDir
	clashPath := filepath.Join(binDir, "clash")
	xclashDir := filepath.Join(binDir, "xclash")
	targetPath := filepath.Join(xclashDir, m.cfg.Core.ClashOption)

	// Cek apakah target ada
	if _, err := os.Stat(targetPath); err == nil {
		// Cek link saat ini
		currentLink, err := os.Readlink(clashPath)
		if err != nil || currentLink != targetPath {
			_ = os.Remove(clashPath)
			if err := os.Symlink(targetPath, clashPath); err != nil {
				slog.Error("Failed to symlink xclash", "error", err)
			} else {
				slog.Info("xclash symlink created", "target", targetPath)
			}
		}
	} else {
		slog.Warn("xclash target not found", "target", targetPath)
	}
}

func (m *Manager) watchProcess(logFile *os.File) {
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	err := m.cmd.Wait()
	m.running = false

	if err != nil {
		slog.Warn("Proxy core process exited with error", "error", err)
		_ = platform.UpdateModulePropDescription("zengobox", fmt.Sprintf("💥 %s crashed unexpectedly!", m.cfg.Core.BinName))
		// TODO: Implementasi crash recovery / auto restart dengan exponential backoff
	} else {
		slog.Info("Proxy core process exited cleanly")
	}
}

func parseUserGroup(ug string) (uint32, uint32, error) {
	parts := strings.Split(ug, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid format")
	}

	uStr := parts[0]
	gStr := parts[1]

	var uid, gid uint32

	// Parse UID
	if uStr == "root" {
		uid = 0
	} else {
		id, err := strconv.ParseUint(uStr, 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse uid: %w", err)
		}
		uid = uint32(id)
	}

	// Parse GID
	if gStr == "net_admin" {
		gid = 3005
	} else {
		id, err := strconv.ParseUint(gStr, 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse gid: %w", err)
		}
		gid = uint32(id)
	}

	return uid, gid, nil
}
