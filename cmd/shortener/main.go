package main

import (
	"fmt"
	"github.com/zauremazhikovayandex/url/internal/app"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"net/http"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	config.InitConfig()
	storage.InitStorage()
	logger.New("info")

	fmt.Println("Running server on", config.AppConfig.ServerAddr)
	return http.ListenAndServe(config.AppConfig.ServerAddr, app.Router())
}
