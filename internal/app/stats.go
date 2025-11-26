package app

import (
	"encoding/json"
	"errors"
	"github.com/zauremazhikovayandex/url/internal/services"
	"net"
	"net/http"
	"strings"
)

// GetInternalStats - возвращает агрегированную статистику сервиса
func (h *Handler) GetInternalStats(w http.ResponseWriter, r *http.Request) {
	ipStr := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	ip := net.ParseIP(ipStr)

	urls, users, err := h.shortener.Stats(r.Context(), ip)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	type stats struct {
		URLs  int `json:"urls"`
		Users int `json:"users"`
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats{URLs: urls, Users: users})
}
