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
	Mu   sync.RWMutex
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
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.data[key] = value
}

// Get возвращает значение по ключу.
func (s *Storage) Get(key string) (string, bool) {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// Delete удаляет значение по ключу.
func (s *Storage) Delete(key string) {
	delete(s.data, key)
}

// ShutdownSaveToFile сохраняет данные хранилища в файл перед остановкой.
func (s *Storage) ShutdownSaveToFile(filename string) error {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

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

	s.Mu.Lock()
	defer s.Mu.Unlock()
	for k, v := range loaded {
		s.data[k] = v
	}
	return nil
}

// DataUnsafe возвращает ссылку на внутреннюю map (только под локом!).
func (s *Storage) DataUnsafe() map[string]string { return s.data }

// getDir - Получение директории хранилища
func getDir(path string) string {
	if idx := len(path) - len("/url_history.json"); idx > 0 {
		return path[:idx]
	}
	return "."
}

// MuRLock выполняет fn под RLock (удобная обёртка).
func (s *Storage) MuRLock(fn func()) {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	fn()
}

// Snapshot возвращает копию текущей мапы (для безопасного чтения размеров/итераций).
func (s *Storage) Snapshot() map[string]string {
	cp := make(map[string]string, len(s.data))
	for k, v := range s.data {
		cp[k] = v
	}
	return cp
}
