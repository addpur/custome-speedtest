package service

import (
	"context"
	"time"

	"github.com/addpur/custome-speedtest/speedtest/domain"
)

// NewScheduler creates a new Scheduler instance
func NewScheduler(cfg domain.Config, engine *Engine, registry *Registry, store domain.SpeedtestStore) domain.Scheduler {
	return &Scheduler{
		cfg:      cfg,
		engine:   engine,
		registry: registry,
		store:    store,
	}
}

type Scheduler struct {
	cfg      domain.Config
	engine   *Engine
	registry *Registry
	store    domain.SpeedtestStore
}

func (s *Scheduler) Start(ctx context.Context) {
	if s.cfg.AutoRun {
		go s.runAuto(ctx)
	}
	if s.cfg.AutoAfter > 0 {
		go s.runAfter(ctx)
	}
}

func (s *Scheduler) runAuto(ctx context.Context) {
	servers := s.registry.GetServers(s.cfg.Keywords)
	if len(servers) == 0 {
		return
	}
	for _, server := range servers {
		s.runTest(ctx, server)
	}
}

func (s *Scheduler) runAfter(ctx context.Context) {
	select {
	case <-time.After(s.cfg.AutoAfter):
		s.runAuto(ctx)
	case <-ctx.Done():
	}
}

func (s *Scheduler) runTest(ctx context.Context, server domain.ServerInfo) {
	duration := s.cfg.Duration
	if s.cfg.StressDuration > 0 {
		duration = s.cfg.StressDuration
	}

	go func() {
		bytes, elapsed, err := s.engine.RunDownload(nil, duration)
		if err == nil {
			result := s.engine.CreateResult(domain.Download, server.URL, bytes, elapsed, s.cfg.Parallel, s.cfg.Overhead)
			s.store.Add(result)
		}
	}()

	go func() {
		bytes, elapsed, err := s.engine.RunUpload(nil, duration)
		if err == nil {
			result := s.engine.CreateResult(domain.Upload, server.URL, bytes, elapsed, s.cfg.Parallel, s.cfg.Overhead)
			s.store.Add(result)
		}
	}()
}
