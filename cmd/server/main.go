package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"goph_keeper/internal/auth"
	"goph_keeper/internal/config"
	"goph_keeper/internal/handlers"
	"goph_keeper/internal/storage"
)

func main() {
	var fo config.ServerFlagOverrides
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	config.RegisterServerFlags(fs, &fo)
	_ = fs.Parse(os.Args[1:])

	logger := log.New(os.Stdout, "", log.LstdFlags)

	cfg, err := config.LoadServerConfig(fo)
	if err != nil {
		logger.Printf("config error: %v", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		logger.Printf("config validation error: %v", err)
		os.Exit(1)
	}

	jwtSvc, err := auth.NewJWTService(cfg.JWTKey)
	if err != nil {
		logger.Printf("jwt init error: %v", err)
		os.Exit(1)
	}

	if strings.ToLower(strings.TrimSpace(cfg.Storage.Type)) != "postgres" {
		logger.Printf("unsupported storage type: %s", cfg.Storage.Type)
		os.Exit(1)
	}

	migrationsPath, err := migrationsDir()
	if err != nil {
		logger.Printf("migrations path error: %v", err)
		os.Exit(1)
	}
	stores, cleanup, err := storage.InitPostgresStores(cfg.Storage.DSN, migrationsPath)
	if err != nil {
		logger.Printf("storage init error: %v", err)
		os.Exit(1)
	}
	if cleanup != nil {
		defer cleanup()
	}

	router, err := handlers.NewRouter(stores.Users, stores.Secrets, jwtSvc)
	if err != nil {
		logger.Printf("router init error: %v", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	logger.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}

func migrationsDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, "migrations"), nil
}
