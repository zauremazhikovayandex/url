package main

import (
	"fmt"
	"github.com/zauremazhikovayandex/url/cmd/routers"
	"github.com/zauremazhikovayandex/url/internal/config"
	"net/http"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	config.InitConfig()

	fmt.Println("Running server on", config.AppConfig.ServerAddr)
	return http.ListenAndServe(config.AppConfig.ServerAddr, routers.Router())
}
