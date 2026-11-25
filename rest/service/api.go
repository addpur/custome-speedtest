package service

import (
	"time"

	restdomain "github.com/addpur/custome-speedtest/rest/domain"
	"github.com/addpur/custome-speedtest/speedtest/domain"
	stservice "github.com/addpur/custome-speedtest/speedtest/service"
)

type APIService struct {
	cfg      domain.Config
	engine   *stservice.Engine
	registry *stservice.Registry
	store    domain.SpeedtestStore
}

func NewAPIService(cfg domain.Config, engine *stservice.Engine, registry *stservice.Registry, store domain.SpeedtestStore) *APIService {
	return &APIService{
		cfg:      cfg,
		engine:   engine,
		registry: registry,
		store:    store,
	}
}

func (s *APIService) Engine() *stservice.Engine {
	return s.engine
}

func (s *APIService) GetConfig() *restdomain.ConfigResponse {
	return &restdomain.ConfigResponse{
		ListenAddr:      s.cfg.ListenAddr,
		BufferSize:      s.cfg.BufferSize,
		DefaultDuration: int(s.cfg.Duration / time.Second),
		DefaultParallel: s.cfg.Parallel,
		DefaultOverhead: s.cfg.Overhead,
	}
}

func (s *APIService) GetServers(keywords []string) []restdomain.ServerResponse {
	servers := s.registry.GetServers(keywords)
	result := make([]restdomain.ServerResponse, 0, len(servers))
	for _, srv := range servers {
		result = append(result, restdomain.ServerResponse{
			URL:      srv.URL,
			Keywords: srv.Keywords,
		})
	}
	return result
}

func (s *APIService) GetResults() []restdomain.ResultResponse {
	results := s.store.Latest(50)
	result := make([]restdomain.ResultResponse, 0, len(results))
	for _, r := range results {
		result = append(result, restdomain.ResultResponse{
			ID:              r.ID,
			Type:            string(r.Type),
			Server:          r.Server,
			Bytes:           r.Bytes,
			Mbps:            r.Mbps,
			Parallel:        r.Parallel,
			OverheadApplied: r.OverheadApplied,
			Timestamp:       r.Timestamp,
		})
	}
	return result
}

func (s *APIService) AddResult(res restdomain.ResultResponse) {
	s.store.Add(domain.Result{
		ID:              res.ID,
		Type:            domain.TestType(res.Type),
		Server:          res.Server,
		Bytes:           res.Bytes,
		Mbps:            res.Mbps,
		Parallel:        res.Parallel,
		OverheadApplied: res.OverheadApplied,
		Timestamp:       res.Timestamp,
	})
}

func (s *APIService) GetStress() *restdomain.StressResponse {
	return &restdomain.StressResponse{
		DownloadMbps: 0,
		UploadMbps:   0,
		TotalMbps:    0,
		Duration:     0,
	}
}
