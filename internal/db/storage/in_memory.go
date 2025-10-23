// Package storage предоставляет простое in-memory и файловое хранилище ссылок.
package storage

import (
	"encoding/json"
	"github.com/zauremazhikovayandex/url/internal/config"
	"log"
	"os"
	"sync"
)

// Store — глобальное in-memory хранилище, инициализируемое при старте.
var Store *Storage

// Storage представляет потокобезопасное хранилище ключ→значение.
type Storage struct {
	data map[string]string
	mu   sync.RWMutex
}

// InitStorage инициализирует глобальное хранилище и загружает данные из файла (если указан путь).
func InitStorage() {
	Store = &Storage{data: make(map[string]string)}

	filePath := config.AppConfig.FileStorage
	if err := Store.LoadFromFile(filePath); err != nil {
		log.Printf("Failed to load store from file: %v", err)
	}
}

// Set сохраняет значение по ключу.
func (s *Storage) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// Get возвращает значение по ключу.
func (s *Storage) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// Delete удаляет значение по ключу.
func (s *Storage) Delete(key string) {
	delete(s.data, key)
}

// ShutdownSaveToFile сохраняет данные хранилища в файл перед остановкой.
func (s *Storage) ShutdownSaveToFile(filename string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := os.MkdirAll(getDir(filename), 0755); err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(s.data)
}

// LoadFromFile загружает данные хранилища из файла.
func (s *Storage) LoadFromFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		// если файл не найден — это не ошибка
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	var loaded map[string]string
	if err := json.NewDecoder(f).Decode(&loaded); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range loaded {
		s.data[k] = v
	}
	return nil
}

// getDir - Получение директории хранилища
func getDir(path string) string {
	if idx := len(path) - len("/url_history.json"); idx > 0 {
		return path[:idx]
	}
	return "."
}
