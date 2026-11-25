package domain

import (
    "context"
)

// StressService interface for running stress tests
type StressService interface {
    StartStress(ctx context.Context, req *RunRequest) error
}

// APIService interface for REST API endpoints
type APIService interface {
    GetConfig() *ConfigResponse
    GetServers(keywords []string) []ServerResponse
    GetResults() []ResultResponse
    GetStress() *StressResponse
}
