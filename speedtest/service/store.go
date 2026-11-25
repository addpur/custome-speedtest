package service

import (
	"sync"

	"github.com/addpur/custome-speedtest/speedtest/domain"
)

type Store struct {
	results []domain.Result
	mu      sync.RWMutex
}

func NewStore() domain.SpeedtestStore {
	return &Store{}
}

func (s *Store) Add(r domain.Result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results = append(s.results, r)
}

func (s *Store) Latest(n int) []domain.Result {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if n >= len(s.results) {
		return s.results
	}
	return s.results[len(s.results)-n:]
}

func (s *Store) All() []domain.Result {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.results
}
