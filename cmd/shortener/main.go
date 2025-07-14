package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/zauremazhikovayandex/url/internal/app"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/services"
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

	//Init File Storage
	storage.InitStorage()

	//Prepare DB
	instance, err := postgres.SQLInstance()
	if err != nil {
		fmt.Println("DB prepare issues", err)
	} else {
		postgres.PrepareDB(instance)
	}

	//Init Logger
	logger.New("info")

	// Create server
	addr := config.AppConfig.ServerAddr
	fmt.Println("Running server on", addr)
	urlService := &services.PostgresURLService{}
	srv := &http.Server{
		Addr:    addr,
		Handler: app.InitHandlers(urlService),
	}

	// Gracefully shutdown
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		log.Println("Shutting down server...")

		// Save to file
		filePath := config.AppConfig.FileStorage
		if err := storage.Store.ShutdownSaveToFile(filePath); err != nil {
			log.Printf("Failed to save store: %v", err)
		} else {
			log.Printf("Store saved to: %s", filePath)
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
