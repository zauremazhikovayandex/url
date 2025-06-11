package config

import (
	"flag"
)

var AppConfig *Config

type Config struct {
	ServerAddr string
	BaseURL    string
}

func initURL() (string, string) {
	serverAddr := ""
	baseURL := ""
	defaultServerAddr := ":8080"
	defaultBaseURL := "http://localhost:8080"

	// флаги
	flag.StringVar(&serverAddr, "a", "", "port to run server")
	flag.StringVar(&baseURL, "b", "", "base URL for short links")
	flag.Parse()

	if serverAddr == "" {
		serverAddr = defaultServerAddr
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
