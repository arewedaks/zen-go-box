package core

import (
	"log/slog"

	"github.com/arewedaks/zen-go-box/internal/config"
	"github.com/arewedaks/zen-go-box/internal/updater"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron *cron.Cron
	cfg  *config.Config
}

func NewScheduler(cfg *config.Config) *Scheduler {
	// Use standard cron (minute hour dom month dow)
	c := cron.New()
	return &Scheduler{
		cron: c,
		cfg:  cfg,
	}
}

func (s *Scheduler) Start() {
	if !s.cfg.Schedule.Enabled {
		return
	}

	cronExpr := s.cfg.Schedule.Cron
	if cronExpr == "" {
		cronExpr = "0 0,6,12,18 * * *" // default every 6 hours
	}

	slog.Info("Smart Scheduler: Starting cron daemon", "expression", cronExpr)

	_, err := s.cron.AddFunc(cronExpr, func() {
		slog.Info("Smart Scheduler: Cron triggered. Executing scheduled tasks...")

		if s.cfg.Schedule.UpdateGeo {
			slog.Info("Smart Scheduler: Updating Geo databases...")
			_ = updater.UpdateGeo(s.cfg.Paths.BoxDir, s.cfg.Core.BinName)
		}

		if s.cfg.Schedule.UpdateSubscription {
			slog.Info("Smart Scheduler: Updating Subscriptions...")
			_ = updater.UpdateSubscription(s.cfg)
		}
	})

	if err != nil {
		slog.Error("Smart Scheduler: Failed to parse cron expression", "error", err)
		return
	}

	s.cron.Start()
}

func (s *Scheduler) Stop() {
	if s.cfg.Schedule.Enabled && s.cron != nil {
		slog.Info("Smart Scheduler: Stopping cron daemon...")
		s.cron.Stop()
	}
}
