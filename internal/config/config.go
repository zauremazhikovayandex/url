package config

import (
	"flag"
	"fmt"
	"os"
)

var AppConfig *Config

type Config struct {
	ServerAddr     string
	BaseURL        string
	UseFileStorage string
	FileStorage    string
	PGConfig       *PostgresConfig
}

type PostgresConfig struct {
	DBConnection string
	DBTimeout    int
}

func InitConfig() {
	// Парсим флаги во временные переменные
	serverAddrFlag := flag.String("a", "", "port to run server")
	baseURLFlag := flag.String("b", "", "base URL for short links")
	fileStorageFlag := flag.String("f", "", "file storage")
	dbConnectionFlag := flag.String("d", "", "postgres connection")
	flag.Parse()

	// Устанавливаем значения по умолчанию
	serverAddr := ":8080"
	baseURL := "http://localhost:8080"
	useFileStorage := "N"
	fileStorage := "url_history.json"
	dbConnection := "host=localhost port=5432 user=postgres password=postgres dbname=aviato sslmode=disable"

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
	if *dbConnectionFlag != "" {
		dbConnection = *dbConnectionFlag
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
	if env := os.Getenv("USE_FILE_STORAGE"); env != "" {
		useFileStorage = env
	}

	AppConfig = &Config{
		ServerAddr:     serverAddr,
		BaseURL:        baseURL,
		FileStorage:    fileStorage,
		UseFileStorage: useFileStorage,
		PGConfig: &PostgresConfig{
			DBConnection: dbConnection,
			DBTimeout:    5,
		},
	}

	if useFileStorage == "Y" {
		fmt.Println("💾 Using file storage path:", AppConfig.FileStorage) // отладка
	}
}
