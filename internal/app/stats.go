// Package app содержит хендлеры
package app

import (
	"encoding/json"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
	"net"
	"net/http"
	"strings"
)

// GetInternalStats возвращает агрегированную статистику сервиса.
// Доступ разрешён только, если X-Real-IP входит в доверенную подсеть из конфигурации.
func (h *Handler) GetInternalStats(w http.ResponseWriter, r *http.Request) {
	// доступ запрещён, если trusted_subnet не задан/не распарсен
	if config.AppConfig.TrustedIPNet == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// по заданию берём только X-Real-IP
	ipStr := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	ip := net.ParseIP(ipStr)
	if ipStr == "" || ip == nil || !config.AppConfig.TrustedIPNet.Contains(ip) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	type stats struct {
		URLs  int `json:"urls"`
		Users int `json:"users"`
	}

	ctx := r.Context()

	if config.AppConfig.StorageType == "DB" {
		u, us, err := postgres.CountStats(ctx)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats{URLs: u, Users: us})
		return
	}

	// In-memory/file
	if storage.Store == nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	storage.Store.Mu.RLock()
	n := len(storage.Store.DataUnsafe()) // см. маленькую утилиту ниже, либо просто len(storage.Store.data)
	storage.Store.Mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats{URLs: n, Users: 0})
}
