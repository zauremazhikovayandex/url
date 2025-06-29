package config

import (
	"flag"
	"os"
)

var AppConfig *Config

type Config struct {
	ServerAddr  string
	BaseURL     string
	FileStorage string
}

func initURL() (string, string, string) {
	serverAddr, baseURL, fileStorage := "", "", ""
	defaultServerAddr, defaultBaseURL := ":8080", "http://localhost:8080"
	defaultFileStorage := "/Users/zauremazhikova/GolandProjects/practicum/storage/url_history.json"

	// флаги
	flag.StringVar(&serverAddr, "a", "", "port to run server")
	flag.StringVar(&baseURL, "b", "", "base URL for short links")
	flag.StringVar(&fileStorage, "f", "", "file storage")
	flag.Parse()

	// ENV переменные
	envAddr := os.Getenv("SERVER_ADDRESS")
	envBase := os.Getenv("BASE_URL")
	envFileStorage := os.Getenv("FILE_STORAGE")

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

	if envFileStorage != "" {
		fileStorage = envFileStorage
	}
	if fileStorage == "" {
		fileStorage = defaultFileStorage
	}

	return serverAddr, baseURL, fileStorage
}

func InitConfig() {

	serverAddr, baseURL, fileStorage := initURL()

	AppConfig = &Config{
		ServerAddr:  serverAddr,
		BaseURL:     baseURL,
		FileStorage: fileStorage,
	}
}
