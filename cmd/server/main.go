package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"goph_keeper/internal/auth"
	"goph_keeper/internal/config"
	"goph_keeper/internal/httpapi"
	"goph_keeper/internal/storage"
)

func main() {
	var fo config.ServerFlagOverrides
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	config.RegisterServerFlags(fs, &fo)
	_ = fs.Parse(os.Args[1:])

	cfg, err := config.LoadServerConfig(fo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "config validation error:", err)
		os.Exit(1)
	}

	jwtSvc, err := auth.NewJWTService(cfg.JWTKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, "jwt init error:", err)
		os.Exit(1)
	}

	store := storage.NewMemoryUserStore()
	authSvc := auth.NewService(store, jwtSvc, 24*time.Hour)
	authHandler := httpapi.NewAuthHandler(authSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("/register", authHandler.Register)
	mux.HandleFunc("/login", authHandler.Login)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
