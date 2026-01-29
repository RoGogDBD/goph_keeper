package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

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
	router, err := handlers.NewRouter(store, jwtSvc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "router init error:", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
