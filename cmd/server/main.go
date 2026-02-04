package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"goph_keeper/internal/auth"
	"goph_keeper/internal/config"
	"goph_keeper/internal/handlers"
	"goph_keeper/internal/storage"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	if err := run(os.Args[1:], logger); err != nil {
		if !errors.Is(err, errUsage) {
			logger.Printf("%v", err)
		}
		os.Exit(1)
	}
}

var errUsage = errors.New("usage")

var (
	initStores        = storage.InitStores
	listenAndServe    = http.ListenAndServe
	listenAndServeTLS = http.ListenAndServeTLS
)

func run(args []string, logger *log.Logger) error {
	var fo config.ServerFlagOverrides
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	config.RegisterServerFlags(fs, &fo)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	cfg, err := config.LoadServerConfig(fo)
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation error: %w", err)
	}

	jwtSvc, err := auth.NewJWTService(cfg.JWTKey)
	if err != nil {
		return fmt.Errorf("jwt init error: %w", err)
	}

	migrationsPath, err := migrationsDir()
	if err != nil {
		return fmt.Errorf("migrations path error: %w", err)
	}
	stores, cleanup, err := initStores(cfg, migrationsPath)
	if err != nil {
		return fmt.Errorf("storage init error: %w", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	router, err := handlers.NewRouter(stores.Users, stores.Secrets, jwtSvc)
	if err != nil {
		return fmt.Errorf("router init error: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	logger.Printf("server listening on %s", addr)
	if cfg.TLS.Enabled {
		return listenAndServeTLS(addr, cfg.TLS.CertFile, cfg.TLS.KeyFile, router)
	}
	return listenAndServe(addr, router)
}

func migrationsDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, "migrations"), nil
}
