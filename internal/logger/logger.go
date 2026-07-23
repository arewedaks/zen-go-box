package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	logFile *os.File
	mu      sync.Mutex
)

// InitLogger menginisialisasi structured logging global menggunakan slog
func InitLogger(logDir string, levelStr string, maxSizeStr string) error {
	mu.Lock()
	defer mu.Unlock()

	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Buat directory log jika belum ada
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log dir: %w", err)
	}

	logPath := filepath.Join(logDir, "runs.log")

	// Lakukan log rotation jika file terlalu besar
	rotateLogIfNeeded(logPath, parseSize(maxSizeStr))

	var err error
	logFile, err = os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	fileHandler := &plainTextHandler{w: logFile, opts: opts, mu: &mu}
	consoleHandler := &plainTextHandler{w: os.Stdout, opts: opts, mu: &mu}

	// Combine handlers
	multiHandler := &multiHandlerImpl{
		handlers: []slog.Handler{fileHandler, consoleHandler},
	}

	logger := slog.New(multiHandler)
	slog.SetDefault(logger)

	return nil
}

// CloseLogger menutup file log
func CloseLogger() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		logFile.Sync()
		logFile.Close()
		logFile = nil
	}
}

type plainTextHandler struct {
	w    io.Writer
	opts *slog.HandlerOptions
	mu   *sync.Mutex
}

func (h *plainTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *plainTextHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Enabled(ctx, r.Level) {
		return nil
	}

	timeStr := r.Time.Format("15:04:05")
	
	var attrs []string
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, fmt.Sprintf("%s: %v", a.Key, a.Value.Any()))
		return true
	})

	attrStr := ""
	if len(attrs) > 0 {
		attrStr = " | " + strings.Join(attrs, " | ")
	}

	// Format: 15:04:05 [INFO]: pesan | key: value
	msg := fmt.Sprintf("%s [%s]: %s%s\n", timeStr, r.Level.String(), r.Message, attrStr)

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write([]byte(msg))
	return err
}

func (h *plainTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *plainTextHandler) WithGroup(name string) slog.Handler { return h }


// multiHandlerImpl mem-forward records ke beberapa handlers sekaligus
type multiHandlerImpl struct {
	handlers []slog.Handler
}

func (m *multiHandlerImpl) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandlerImpl) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			_ = h.Handle(ctx, r)
		}
	}
	return nil
}

func (m *multiHandlerImpl) WithAttrs(attrs []slog.Attr) slog.Handler {
	return m
}

func (m *multiHandlerImpl) WithGroup(name string) slog.Handler {
	return m
}

func parseSize(sizeStr string) int64 {
	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))
	if sizeStr == "" {
		return 1024 * 1024 // 1MB default
	}

	var multiplier int64 = 1
	if strings.HasSuffix(sizeStr, "K") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "K")
	} else if strings.HasSuffix(sizeStr, "M") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "M")
	} else if strings.HasSuffix(sizeStr, "G") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "G")
	}

	size, err := strconv.ParseFloat(sizeStr, 64)
	if err != nil || size <= 0 {
		size = 1.0
	}
	return int64(size * float64(multiplier))
}

func rotateLogIfNeeded(path string, maxSize int64) {
	fi, err := os.Stat(path)
	if err != nil {
		return
	}

	if fi.Size() >= maxSize {
		oldPath := path + ".old"
		os.Remove(oldPath)
		os.Rename(path, oldPath)
	}
}

// Toast menampilkan notifikasi Android toast melalui cmd notification jika berjalan sebagai root
func Toast(message string) {
	slog.Info(fmt.Sprintf("Notification: %s", message))
	cmd := exec.Command("cmd", "notification", "post", "-I", "https://arewedaks.com", "-S", "bigtext", "-t", "ZenGoBox", "Zen", message)
	_ = cmd.Run()
}
