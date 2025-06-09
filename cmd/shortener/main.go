package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/go-chi/chi/v5"
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

	shortURL := fmt.Sprintf("%s/%s", baseURL, id)
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

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(originalURL))
	if err != nil {
		log.Println(err)
	}

}

func Router() chi.Router {
	r := chi.NewRouter()

	r.Post("/", PostHandler)
	r.Get("/{id}", GetHandler)
	return r
}

var (
	serverAddr string
	baseURL    string
)

func parseFlags() {
	// флаги
	flag.StringVar(&serverAddr, "a", ":8080", "port to run server")
	flag.StringVar(&baseURL, "b", "http://localhost:8080", "base URL for short links")
	flag.Parse()
}

func main() {

	parseFlags()
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	fmt.Println("Running server on", serverAddr)
	return http.ListenAndServe(serverAddr, Router())
}
