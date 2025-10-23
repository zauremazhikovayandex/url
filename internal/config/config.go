package config

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	AppConfig *Config
	once      sync.Once
)

type Config struct {
	ServerAddr     string
	BaseURL        string
	UseFileStorage string
	FileStorage    string
	PGConfig       *PostgresConfig
	StorageType    string
	JWTSecretKey   string
	JWTTokenExp    time.Duration
	JWTCookieName  string
}

type PostgresConfig struct {
	DBConnection string
	DBTimeout    int
}

func InitConfig() {
	once.Do(func() {
		// Парсим флаги во временные переменные
		serverAddrFlag := flag.String("a", "", "port to run server")
		baseURLFlag := flag.String("b", "", "base URL for short links")
		fileStorageFlag := flag.String("f", "", "file storage")
		dbConnectionFlag := flag.String("d", "", "postgres connection")
		flag.Parse()

		// значения по умолчанию
		serverAddr := ":8080"
		baseURL := "http://localhost:8080"
		fileStorage := ""
		dbConnection := ""

		// флаги (если переданы)
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

		// окружение
		if env := os.Getenv("SERVER_ADDRESS"); env != "" {
			serverAddr = env
		}
		if env := os.Getenv("BASE_URL"); env != "" {
			baseURL = env
		}
		if env := os.Getenv("FILE_STORAGE_PATH"); env != "" {
			fileStorage = env
		}
		if env := os.Getenv("DATABASE_DSN"); env != "" {
			dbConnection = env
		}

		storageType := "Memory"
		if dbConnection != "" {
			storageType = "DB"
		} else if fileStorage != "" {
			storageType = "File"
		}

		AppConfig = &Config{
			ServerAddr:  serverAddr,
			BaseURL:     baseURL,
			FileStorage: fileStorage,
			PGConfig: &PostgresConfig{
				DBConnection: dbConnection,
				DBTimeout:    10,
			},
			StorageType:   storageType,
			JWTSecretKey:  "supersecretkey",
			JWTTokenExp:   time.Hour * 3,
			JWTCookieName: "auth_token",
		}

		fmt.Println("Storage type:", storageType)
		if storageType == "File" {
			fmt.Println("💾 Using file storage path:", AppConfig.FileStorage)
		}
	})
}
