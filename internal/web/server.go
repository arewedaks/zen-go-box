package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	mux.HandleFunc("/api/config", s.handleConfig)

	s.srv = &http.Server{
		Addr:    "127.0.0.1:9999",
		Handler: mux,
	}

	go func() {
		slog.Info("Zashboard Web UI started on http://127.0.0.1:9999")
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
