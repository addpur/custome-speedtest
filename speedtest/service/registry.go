package service

import (
	"sync"

	"github.com/addpur/custome-speedtest/speedtest/domain"
)

type Registry struct {
	servers []domain.ServerInfo
	mu      sync.RWMutex
}

func NewRegistry(cfg domain.Config) *Registry {
	var list []domain.ServerInfo
	for _, url := range cfg.Servers {
		list = append(list, domain.ServerInfo{URL: url})
	}
	return &Registry{servers: list}
}

func (r *Registry) GetServers(keywords []string) []domain.ServerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(keywords) == 0 {
		return r.servers
	}

	var filtered []domain.ServerInfo
	for _, s := range r.servers {
		for _, kw := range keywords {
			if contains(s.URL, kw) {
				filtered = append(filtered, s)
				break
			}
		}
	}
	if len(filtered) == 0 {
		return r.servers
	}
	return filtered
}

func (r *Registry) AddServer(url string, keywords []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.servers = append(r.servers, domain.ServerInfo{URL: url, Keywords: keywords})
}

func (r *Registry) All() []domain.ServerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.servers
}

func contains(s, kw string) bool {
	return len(kw) == 0 || len(s) >= len(kw) && substring(s, kw)
}

func substring(s, kw string) bool {
	for i := 0; i <= len(s)-len(kw); i++ {
		if s[i:i+len(kw)] == kw {
			return true
		}
	}
	return false
}
