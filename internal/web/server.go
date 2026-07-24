package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/arewedaks/zen-go-box/internal/core"
)

//go:embed assets/*
var assetsFS embed.FS

type Server struct {
	mgr *core.Manager
	srv *http.Server
}

func StartServer(mgr *core.Manager) {
	s := &Server{
		mgr: mgr,
	}

	mux := http.NewServeMux()

	// Serve Static Files
	subFS, _ := fs.Sub(assetsFS, "assets")
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	// API Endpoints
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/start", s.handleStart)
	mux.HandleFunc("/api/stop", s.handleStop)
	mux.HandleFunc("/api/restart", s.handleRestart)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/setup", s.handleSetup)
	mux.HandleFunc("/api/setup_log", s.handleSetupLog)

	s.srv = &http.Server{
		Addr:    ":9999",
		Handler: mux,
	}

	go func() {
		slog.Info("Zashboard Web UI started on http://0.0.0.0:9999 (Available on LAN)")
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Zashboard server failed", "error", err)
		}
	}()
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	running, pid := s.mgr.Status() 
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"running": running,
		"core":    s.mgr.Config().Core.BinName,
		"pid":     pid,
		"needs_setup": s.mgr.Config().NeedsSetup,
	})
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Start in background to prevent blocking HTTP response
	go func() {
		if err := s.mgr.Start(); err != nil {
			slog.Error("Zashboard failed to start core", "error", err)
		}
	}()
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "starting"}`))
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if err := s.mgr.Stop(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "stopped"}`))
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	go func() {
		_ = s.mgr.Stop()
		exec.Command("sh", "/data/adb/modules/zengobox/service.sh").Start()
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "restarting"}`))
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Read the last 100 lines of runs.log
	logPath := filepath.Join(s.mgr.Config().Paths.LogDir, "runs.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"logs": []string{"No logs found or error reading log file."}})
		return
	}
	
	lines := strings.Split(string(content), "\n")
	// Get last 100 lines
	if len(lines) > 100 {
		lines = lines[len(lines)-100:]
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{"logs": lines})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.mgr.Config())
		return
	}

	if r.Method == http.MethodPost {
		var newCfg map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Because JSON structure matches YAML, we can convert to JSON, then Unmarshal to struct
		b, _ := json.Marshal(newCfg)
		
		cfg := s.mgr.Config()
		if err := json.Unmarshal(b, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Save to file (assume default path)
		configPath := filepath.Join(cfg.Paths.BoxDir, "zengobox.yaml")
		if err := cfg.Save(configPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "saved"}`))
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Core string `json:"core"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Tulis progress log ke temp file
	logFile := "/data/local/tmp/zengobox_setup.log"
	_ = os.Remove(logFile)
	
	// Eksekusi setup core di background
	cmd := exec.Command("/data/adb/zengobox/bin/zengobox", "setup", req.Core)
	outFile, _ := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY, 0644)
	cmd.Stdout = outFile
	cmd.Stderr = outFile
	
	err := cmd.Start()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSetupLog(w http.ResponseWriter, r *http.Request) {
	logFile := "/data/local/tmp/zengobox_setup.log"
	data, err := os.ReadFile(logFile)
	if err != nil {
		w.Write([]byte("Waiting for logs..."))
		return
	}
	
	// Just return the last 3000 bytes so it doesn't get too large
	if len(data) > 3000 {
		data = data[len(data)-3000:]
	}
	w.Write(data)
}
