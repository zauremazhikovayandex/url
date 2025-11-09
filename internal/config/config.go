// Package config - Работа с конфигурацией приложения
package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// AppConfig — глобальная конфигурация приложения.
var (
	AppConfig *Config
	once      sync.Once
)

// Config описывает параметры HTTP-сервера, хранилища и авторизации.
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
	EnableHTTPS    bool
}

// PostgresConfig описывает параметры подключения к PostgreSQL.
type PostgresConfig struct {
	DBConnection string
	DBTimeout    int
}

func boolEnv(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// InitConfig инициализирует конфигурацию из флагов и переменных окружения.
func InitConfig() {
	once.Do(func() {
		// Парсим флаги во временные переменные
		serverAddrFlag := flag.String("a", "", "port to run server")
		baseURLFlag := flag.String("b", "", "base URL for short links")
		fileStorageFlag := flag.String("f", "", "file storage")
		dbConnectionFlag := flag.String("d", "", "postgres connection")
		enableHTTPSFlag := flag.Bool("s", false, "enable HTTPS")
		flag.Parse()

		// значения по умолчанию
		serverAddr := ":8080"
		baseURL := "http://localhost:8080"
		fileStorage := ""
		dbConnection := ""
		enableHTTPS := false

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
		if *enableHTTPSFlag {
			enableHTTPS = true
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
		if boolEnv("ENABLE_HTTPS") {
			enableHTTPS = true
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
			EnableHTTPS:   enableHTTPS,
		}

		fmt.Println("Storage type:", storageType)
		if storageType == "File" {
			fmt.Println("💾 Using file storage path:", AppConfig.FileStorage)
		}
		if AppConfig.EnableHTTPS {
			fmt.Println("🔐 HTTPS: enabled")
		} else {
			fmt.Println("🔓 HTTPS: disabled")
		}
	})
}
