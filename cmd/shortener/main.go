package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/zauremazhikovayandex/url/internal/app"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/services"
	"log"
	"math/big"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

// printRealOrDefault выводит "<label>: <value|N/A>"
func printRealOrDefault(label, value string) {
	if value == "" {
		value = "N/A"
	}
	fmt.Printf("%s: %s\n", label, value)
}

// printBuildInfo печатает сведения о сборке в stdout
func printBuildInfo() {
	printRealOrDefault("Build version", buildVersion)
	printRealOrDefault("Build date", buildDate)
	printRealOrDefault("Build commit", buildCommit)
}

// makeSelfSignedTLSConfig генерирует самоподписанный сертификат (RSA 2048)
// для "localhost" и 127.0.0.1 на 1 год.
func makeSelfSignedTLSConfig() (*tls.Config, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "url-shortener self-signed",
			Organization: []string{"url-shortener"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}, nil
}

func run() error {
	// Печать сведений о сборке
	printBuildInfo()

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

	// PPROF (оставляем на 6060, без TLS)
	go func() {
		log.Println("pprof on http://127.0.0.1:6060/debug/pprof/")
		if err := http.ListenAndServe("127.0.0.1:6060", nil); err != nil {
			log.Println("pprof server error:", err)
		}
	}()

	// Стартуем HTTP или HTTPS в зависимости от конфигурации
	if config.AppConfig.EnableHTTPS {
		tcfg, err := makeSelfSignedTLSConfig()
		if err != nil {
			return fmt.Errorf("tls config err: %w", err)
		}
		ln, err := tls.Listen("tcp", addr, tcfg)
		if err != nil {
			return fmt.Errorf("tls listen err: %w", err)
		}
		log.Println("HTTPS enabled (self-signed cert).")
		// Serve на готовом TLS-listener
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("https server error: %w", err)
		}
	} else {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}
	}

	return nil
}
