package config

import (
	"flag"
	"os"
)

var AppConfig *Config

type Config struct {
	ServerAddr string
	BaseURL    string
}

func initURL() (string, string) {
	serverAddr, baseURL := "", ""
	defaultServerAddr, defaultBaseURL := ":8080", "http://localhost:8080"

	// флаги
	flag.StringVar(&serverAddr, "a", "", "port to run server")
	flag.StringVar(&baseURL, "b", "", "base URL for short links")
	flag.Parse()

	// ENV переменные
	envAddr := os.Getenv("SERVER_ADDRESS")
	envBase := os.Getenv("BASE_URL")

	// приоритет: env > флаг > default
	if envAddr != "" {
		serverAddr = envAddr
	}
	if serverAddr == "" {
		serverAddr = defaultServerAddr
	}

	if envBase != "" {
		baseURL = envBase
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return serverAddr, baseURL
}

func InitConfig() {

	serverAddr, baseURL := initURL()

	AppConfig = &Config{
		ServerAddr: serverAddr,
		BaseURL:    baseURL,
	}
}
