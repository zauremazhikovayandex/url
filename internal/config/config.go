package config

import (
	"flag"
	"fmt"
	"os"
)

var AppConfig *Config

type Config struct {
	ServerAddr  string
	BaseURL     string
	FileStorage string
}

func InitConfig() {
	// Парсим флаги во временные переменные
	serverAddrFlag := flag.String("a", "", "port to run server")
	baseURLFlag := flag.String("b", "", "base URL for short links")
	fileStorageFlag := flag.String("f", "", "file storage")
	flag.Parse()

	// Устанавливаем значения по умолчанию
	serverAddr := ":8080"
	baseURL := "http://localhost:8080"
	fileStorage := "url_history.json"

	// Переопределяем флагами
	if *serverAddrFlag != "" {
		serverAddr = *serverAddrFlag
	}
	if *baseURLFlag != "" {
		baseURL = *baseURLFlag
	}
	if *fileStorageFlag != "" {
		fileStorage = *fileStorageFlag
	}

	// Окружением (имеет самый высокий приоритет)
	if env := os.Getenv("SERVER_ADDRESS"); env != "" {
		serverAddr = env
	}
	if env := os.Getenv("BASE_URL"); env != "" {
		baseURL = env
	}
	if env := os.Getenv("FILE_STORAGE_PATH"); env != "" {
		fileStorage = env
	}

	AppConfig = &Config{
		ServerAddr:  serverAddr,
		BaseURL:     baseURL,
		FileStorage: fileStorage,
	}

	fmt.Println("💾 Using file storage path:", AppConfig.FileStorage) // отладка
}
