package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/zauremazhikovayandex/url/cmd/config"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func generateShortID(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:n], nil
}

func PostHandler(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	originalURL := strings.TrimSpace(string(body))
	id, err := generateShortID(8)
	if err != nil || len(body) == 0 {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = os.Setenv(id, originalURL)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	shortURL := fmt.Sprintf("%s/%s", config.AppConfig.BaseURL, id)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(shortURL))
	if err != nil {
		log.Println(err)
	}

}

func GetHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	originalURL := os.Getenv(id)
	if originalURL == "" {
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

func Router() chi.Router {
	r := chi.NewRouter()

	r.Post("/", PostHandler)
	r.Get("/{id}", GetHandler)
	return r
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	config.InitConfig()

	fmt.Println("Running server on", config.AppConfig.ServerAddr)
	return http.ListenAndServe(config.AppConfig.ServerAddr, Router())
}
