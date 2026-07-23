package network

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/arewedaks/zengobox/internal/config"
	"github.com/arewedaks/zengobox/internal/core"
)

type ModuleWatcher struct {
	watcher *fsnotify.Watcher
	cfg     *config.Config
	mgr     *core.Manager
	done    chan bool
}

func NewModuleWatcher(cfg *config.Config, mgr *core.Manager) (*ModuleWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &ModuleWatcher{
		watcher: watcher,
		cfg:     cfg,
		mgr:     mgr,
		done:    make(chan bool),
	}, nil
}

func (mw *ModuleWatcher) Start() {
	// Pantau directory modul untuk file "disable"
	watchDir := filepath.Join("/data/adb/modules", "box_for_root")
	if _, err := os.Stat(watchDir); err != nil {
		// Fallback dev
		watchDir = "."
	}

	slog.Info("Starting module status watcher", "path", watchDir)
	_ = mw.watcher.Add(watchDir)

	go func() {
		for {
			select {
			case event, ok := <-mw.watcher.Events:
				if !ok {
					return
				}

				// Cek file disable
				if filepath.Base(event.Name) == "disable" {
					if event.Has(fsnotify.Create) {
						slog.Info("Module 'disable' file created. Stopping service...")
						_ = mw.mgr.Stop()
					} else if event.Has(fsnotify.Remove) {
						slog.Info("Module 'disable' file removed. Starting service...")
						_ = mw.mgr.Start()
					}
				}
			case err, ok := <-mw.watcher.Errors:
				if !ok {
					return
				}
				slog.Error("Module watcher error", "error", err)
			case <-mw.done:
				return
			}
		}
	}()
}

func (mw *ModuleWatcher) Stop() {
	mw.done <- true
	mw.watcher.Close()
	slog.Info("Module status watcher stopped.")
}
