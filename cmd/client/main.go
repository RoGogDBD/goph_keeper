package main

import (
	"flag"
	"fmt"
	"os"

	"goph_keeper/internal/config"
	"goph_keeper/internal/version"
)

func main() {
	// проверка на аргумент "version" для быстрой печати версии
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version.Info())
		return
	}

	var fo config.ClientFlagOverrides
	var showVersion bool
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	config.RegisterClientFlags(fs, &fo)
	fs.BoolVar(&showVersion, "version", false, "print version and exit")
	_ = fs.Parse(os.Args[1:])

	if showVersion {
		fmt.Println(version.Info())
		return
	}

	cfg, err := config.LoadClientConfig(fo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "config validation error:", err)
		os.Exit(1)
	}

	fmt.Printf("client config: server_url=%s data_dir=%s tls=%v log=%s/%s\n",
		cfg.ServerURL, cfg.DataDir, cfg.TLS, cfg.Log.Level, cfg.Log.Format)

	// TODO: initialize client and start CLI
}
