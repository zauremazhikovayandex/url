// Package config - Работа с конфигурацией приложения
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
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

// boolEnvPtr парсит строку из окружения в *bool; возвращает nil, если распарсить нельзя.
func boolEnvPtr(v string) *bool {
	s := strings.TrimSpace(strings.ToLower(v))
	switch s {
	case "1", "true", "t", "yes", "on":
		b := true
		return &b
	case "0", "false", "f", "no", "off":
		b := false
		return &b
	default:
		if b, err := strconv.ParseBool(v); err == nil {
			return &b
		}
		return nil
	}
}

// pickBool выбирает итоговое булево значение по приоритету флаг > env > файл > дефолт.
func pickBool(flag *boolFlag, envPtr *bool, filePtr *bool, def bool) bool {
	switch {
	case flag != nil && flag.set:
		return flag.value
	case envPtr != nil:
		return *envPtr
	case filePtr != nil:
		return *filePtr
	default:
		return def
	}
}

// jsonConfig — структура для чтения настроек из JSON-файла конфигурации.
type jsonConfig struct {
	ServerAddress *string `json:"server_address"`
	BaseURL       *string `json:"base_url"`
	FileStorage   *string `json:"file_storage_path"`
	DatabaseDSN   *string `json:"database_dsn"`
	EnableHTTPS   *bool   `json:"enable_https"`
}

// boolFlag — вспомогательный тип для булевых флагов с приоритетом "задан/не задан".
type boolFlag struct {
	set   bool
	value bool
}

// Set устанавливает значение булевого флага и помечает его как явно заданный.
func (b *boolFlag) Set(s string) error {
	b.set = true
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	b.value = v
	return nil
}

// String возвращает строковое представление текущего значения флага.
func (b *boolFlag) String() string {
	if b == nil {
		return "false"
	}
	return strconv.FormatBool(b.value)
}

// IsBoolFlag отмечает флаг как булев, позволяя писать -s без значения.
func (b *boolFlag) IsBoolFlag() bool { return true }

// InitConfig инициализирует конфигурацию из флагов и переменных окружения.
func InitConfig() {
	once.Do(func() {
		// флаги
		var cfgPath string
		var httpsFlag boolFlag

		serverAddrFlag := flag.String("a", "", "port to run server")
		baseURLFlag := flag.String("b", "", "base URL for short links")
		fileStorageFlag := flag.String("f", "", "file storage")
		dbConnFlag := flag.String("d", "", "postgres connection")

		// алиасы
		flag.StringVar(&cfgPath, "c", "", "path to config file (JSON)")
		flag.StringVar(&cfgPath, "config", "", "path to config file (JSON)")
		flag.Var(&httpsFlag, "s", "enable HTTPS (bool)")
		flag.Var(&httpsFlag, "https", "enable HTTPS (bool)")

		flag.Parse()

		// env
		if v := os.Getenv("CONFIG"); v != "" && cfgPath == "" {
			cfgPath = v
		}
		envServerAddr := os.Getenv("SERVER_ADDRESS")
		envBaseURL := os.Getenv("BASE_URL")
		envFilePath := os.Getenv("FILE_STORAGE_PATH")
		envDB := os.Getenv("DATABASE_DSN")
		var envHTTPS *bool
		if v, ok := os.LookupEnv("ENABLE_HTTPS"); ok {
			envHTTPS = boolEnvPtr(v)
		}

		// file
		var fileCfg jsonConfig
		if cfgPath != "" {
			if data, err := os.ReadFile(cfgPath); err == nil {
				_ = json.Unmarshal(data, &fileCfg) // мягко
			} else {
				fmt.Println("config: cannot read file:", err)
			}
		}

		// сбор финальных значений по приоритету
		// helpers
		pickStr := func(flagVal, envVal string, filePtr *string, def string) string {
			switch {
			case flagVal != "":
				return flagVal
			case envVal != "":
				return envVal
			case filePtr != nil && *filePtr != "":
				return *filePtr
			default:
				return def
			}
		}

		// defaults
		serverAddr := pickStr(*serverAddrFlag, envServerAddr, fileCfg.ServerAddress, ":8080")
		baseURL := pickStr(*baseURLFlag, envBaseURL, fileCfg.BaseURL, "http://localhost:8080")
		filePath := pickStr(*fileStorageFlag, envFilePath, fileCfg.FileStorage, "")
		dbConn := pickStr(*dbConnFlag, envDB, fileCfg.DatabaseDSN, "")
		enableTLS := pickBool(&httpsFlag, envHTTPS, fileCfg.EnableHTTPS, false)

		storageType := "Memory"
		if dbConn != "" {
			storageType = "DB"
		} else if filePath != "" {
			storageType = "File"
		}

		AppConfig = &Config{
			ServerAddr:  serverAddr,
			BaseURL:     baseURL,
			FileStorage: filePath,
			PGConfig: &PostgresConfig{
				DBConnection: dbConn,
				DBTimeout:    10,
			},
			StorageType:   storageType,
			JWTSecretKey:  "supersecretkey",
			JWTTokenExp:   time.Hour * 3,
			JWTCookieName: "auth_token",
			EnableHTTPS:   enableTLS,
		}

		fmt.Println("Storage type:", storageType)
		if storageType == "File" {
			fmt.Println("💾 Using file storage path:", AppConfig.FileStorage)
		}
	})
}
