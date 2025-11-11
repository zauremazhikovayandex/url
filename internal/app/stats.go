// Package app содержит хендлеры
package app

import (
	"encoding/json"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
	"net"
	"net/http"
	"strings"
)

// GetInternalStats возвращает агрегированную статистику сервиса.
// Доступ разрешён только, если X-Real-IP входит в доверенную подсеть из конфигурации.
func (h *Handler) GetInternalStats(w http.ResponseWriter, r *http.Request) {
	// trusted_subnet обязан быть настроен; иначе 403
	if config.AppConfig.TrustedIPNet == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Берём IP из X-Real-IP (по условию задачи)
	ipStr := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if ipStr == "" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	ip := net.ParseIP(ipStr)
	if ip == nil || !config.AppConfig.TrustedIPNet.Contains(ip) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// подсчёты
	type stats struct {
		URLs  int `json:"urls"`
		Users int `json:"users"`
	}

	ctx := r.Context()
	if config.AppConfig.StorageType == "DB" {
		u, us, err := h.urlService.GetStats(ctx)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats{URLs: u, Users: us})
		return
	}

	// In-memory/file режим: считаем только URL'ы, пользователей посчитать нечем — 0.
	if storage.Store != nil {
		// thread-safe подсчёт
		var n int
		storage.Store.MuRLock(func() { n = len(storage.Store.Snapshot()) })
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats{URLs: n, Users: 0})
		return
	}

	http.Error(w, "server error", http.StatusInternalServerError)
}
