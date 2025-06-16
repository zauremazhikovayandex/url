package config

import "flag"

var AppConfig *Config

type Config struct {
	ServerAddr string
	BaseURL    string
}

func initURL() (string, string) {
	serverAddr, baseURL := "", ""

	// флаги
	flag.StringVar(&serverAddr, "a", "", "port to run server")
	flag.StringVar(&baseURL, "b", "", "base URL for short links")
	flag.Parse()

	return serverAddr, baseURL
}

func InitConfig() {

	serverAddr, baseURL := initURL()

	AppConfig = &Config{
		ServerAddr: serverAddr,
		BaseURL:    baseURL,
	}
}
