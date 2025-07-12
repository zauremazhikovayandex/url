package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/zauremazhikovayandex/url/internal/app"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	//Init Config
	config.InitConfig()
	storage.InitStorage()

	//Init Logger
	logger.New("info")

	addr := config.AppConfig.ServerAddr
	fmt.Println("Running server on", addr)

	// Create server
	srv := &http.Server{
		Addr:    addr,
		Handler: app.Router(),
	}

	// Gracefully shutdown
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		log.Println("Shutting down server...")

		if config.AppConfig.UseFileStorage == "Y" {
			// Save to file
			filePath := config.AppConfig.FileStorage
			if err := storage.Store.ShutdownSaveToFile(filePath); err != nil {
				log.Printf("Failed to save store: %v", err)
			} else {
				log.Printf("Store saved to: %s", filePath)
			}
		}

		// Shutdown server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
