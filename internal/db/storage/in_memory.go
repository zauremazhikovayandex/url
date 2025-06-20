package storage

import "sync"

var Store *Storage

type Storage struct {
	data map[string]string
	mu   sync.RWMutex
}

func InitStorage() {
	Store = &Storage{data: make(map[string]string)}
}

func (s *Storage) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *Storage) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

func (s *Storage) Delete(key string) {
	delete(s.data, key)
}
