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
	serverAddrFlag := flag.String("a", "", "port to run server")
	baseURLFlag := flag.String("b", "", "base URL for short links")
	fileStorageFlag := flag.String("f", "", "file storage")
	flag.Parse()

	// ENV переменные
	envAddr := os.Getenv("SERVER_ADDRESS")
	envBase := os.Getenv("BASE_URL")
	envFileStorage := os.Getenv("FILE_STORAGE")

	// Итоговые значения (приоритет: ENV > FLAG > DEFAULT)
	serverAddr := ":8080"
	if *serverAddrFlag != "" {
		serverAddr = *serverAddrFlag
	}
	if envAddr != "" {
		serverAddr = envAddr
	}

	baseURL := "http://localhost:8080"
	if *baseURLFlag != "" {
		baseURL = *baseURLFlag
	}
	if envBase != "" {
		baseURL = envBase
	}

	fileStorage := "/Users/zauremazhikova/GolandProjects/practicum/storage/url_history.json"
	if *fileStorageFlag != "" {
		fileStorage = *fileStorageFlag
	}
	if envFileStorage != "" {
		fileStorage = envFileStorage
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
