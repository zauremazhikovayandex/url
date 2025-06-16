package config

var AppConfig *Config

type Config struct {
	ServerAddr string
	BaseURL    string
}

func initURL() (string, string) {
	serverAddr, baseURL := ":8080", "http://localhost:8080"
	return serverAddr, baseURL
}

func InitConfig() {

	serverAddr, baseURL := initURL()

	AppConfig = &Config{
		ServerAddr: serverAddr,
		BaseURL:    baseURL,
	}
}
