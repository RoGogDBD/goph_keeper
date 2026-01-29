package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"

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

	router := chi.NewRouter()
	router.Post("/register", authHandler.Register)
	router.Post("/login", authHandler.Login)
	router.With(httpapi.AuthMiddleware(jwtSvc)).Get("/me", authHandler.Me)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
