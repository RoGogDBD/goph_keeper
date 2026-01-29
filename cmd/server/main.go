package main

import (
	"flag"
	"fmt"
	"os"

	"goph_keeper/internal/config"
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

	fmt.Printf("server config: host=%s port=%d storage=%s dsn=%s log=%s/%s\n",
		cfg.Host, cfg.Port, cfg.Storage.Type, cfg.Storage.DSN, cfg.Log.Level, cfg.Log.Format)

	// TODO: initialize logger, storage, router, and start server
}
