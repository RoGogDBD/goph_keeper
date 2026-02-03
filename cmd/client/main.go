package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"goph_keeper/internal/client/api"
	"goph_keeper/internal/client/crypto"
	clientservice "goph_keeper/internal/client/service"
	"goph_keeper/internal/client/store"
	"goph_keeper/internal/client/ui"
	"goph_keeper/internal/config"
	"goph_keeper/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if !errors.Is(err, errUsage) {
			printErrLine(err.Error())
		}
		os.Exit(1)
	}
}

var errUsage = errors.New("usage")

func run(args []string) error {
	if len(args) > 0 && args[0] == "version" {
		printLine(version.Info())
		return nil
	}

	var fo config.ClientFlagOverrides
	var showVersion bool
	var masterPassword string
	root := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	config.RegisterClientFlags(root, &fo)
	root.BoolVar(&showVersion, "version", false, "print version and exit")
	root.StringVar(&masterPassword, "master-password", "", "master password for encryption")
	if err := root.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if showVersion {
		printLine(version.Info())
		return nil
	}

	rest := root.Args()
	if len(rest) == 0 {
		printClientUsage()
		return errUsage
	}

	cfg, err := config.LoadClientConfig(fo)
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation error: %w", err)
	}

	tokenStore := store.NewTokenStore(cfg.DataDir)
	cryptoSvc, err := buildCrypto(masterPassword, cfg.DataDir, rest[0])
	if err != nil {
		return fmt.Errorf("crypto error: %w", err)
	}
	httpClient, err := buildHTTPClient(cfg)
	if err != nil {
		return fmt.Errorf("http client error: %w", err)
	}
	cli := api.New(cfg.ServerURL, tokenStore, cryptoSvc, api.WithHTTPClient(httpClient))
	local, err := store.NewLocalStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("local store error: %w", err)
	}
	syncStore := store.NewSyncStore(cfg.DataDir)
	svc := clientservice.New(cli, local, syncStore, cryptoSvc)
	defer func() {
		if cerr := local.Close(); cerr != nil {
			_ = cerr
		}
	}()
	ctx := context.Background()

	switch rest[0] {
	case "register":
		if err := handleRegister(ctx, svc, rest[1:]); err != nil {
			return err
		}
	case "login":
		if err := handleLogin(ctx, svc, rest[1:]); err != nil {
			return err
		}
	case "add":
		if err := handleAdd(ctx, svc, rest[1:]); err != nil {
			return err
		}
	case "list":
		if err := handleList(ctx, svc); err != nil {
			return err
		}
	case "get":
		if err := handleGet(ctx, svc, rest[1:]); err != nil {
			return err
		}
	case "update":
		if err := handleUpdate(ctx, svc, rest[1:]); err != nil {
			return err
		}
	case "delete":
		if err := handleDelete(ctx, svc, rest[1:]); err != nil {
			return err
		}
	case "sync":
		if err := handleSync(ctx, svc); err != nil {
			return err
		}
	case "ui":
		if err := ui.RunUIWithHTTPClient(cfg.ServerURL, cfg.DataDir, httpClient); err != nil {
			return fmt.Errorf("ui error: %w", err)
		}
	default:
		printClientUsage()
		return errUsage
	}
	return nil
}

func buildCrypto(masterPassword, dataDir, command string) (*crypto.Crypto, error) {
	needsCrypto := map[string]bool{
		"add":    true,
		"get":    true,
		"update": true,
	}

	if !needsCrypto[command] {
		return nil, nil
	}

	return crypto.NewCrypto(masterPassword, dataDir)
}

func buildHTTPClient(cfg config.ClientConfig) (*http.Client, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	if !cfg.TLS {
		return client, nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if strings.TrimSpace(cfg.TLSCA) != "" {
		caPEM, err := os.ReadFile(cfg.TLSCA)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("failed to parse tls_ca")
		}
		tlsConfig.RootCAs = pool
	}

	client.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	return client, nil
}

func handleRegister(ctx context.Context, svc *clientservice.Service, args []string) error {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	var email, password string
	fs.StringVar(&email, "email", "", "user email")
	fs.StringVar(&password, "password", "", "user password")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse register flags: %w", err)
	}

	if err := svc.Register(ctx, email, password); err != nil {
		return fmt.Errorf("register error: %w", err)
	}
	printLine("registered")
	return nil
}

func handleLogin(ctx context.Context, svc *clientservice.Service, args []string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	var email, password string
	fs.StringVar(&email, "email", "", "user email")
	fs.StringVar(&password, "password", "", "user password")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse login flags: %w", err)
	}

	if err := svc.Login(ctx, email, password); err != nil {
		return fmt.Errorf("login error: %w", err)
	}
	printLine("logged in")
	return nil
}

func handleAdd(ctx context.Context, svc *clientservice.Service, args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	var typ, payload, meta string
	fs.StringVar(&typ, "type", "", "secret type")
	fs.StringVar(&payload, "payload", "", "secret payload")
	fs.StringVar(&meta, "meta", "", "meta key=value pairs separated by comma")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse add flags: %w", err)
	}

	id, err := svc.Add(ctx, typ, payload, meta)
	if err != nil {
		return fmt.Errorf("add error: %w", err)
	}
	printLine(id)
	return nil
}

func handleList(ctx context.Context, svc *clientservice.Service) error {
	list, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("list error: %w", err)
	}
	for _, s := range list {
		printFmt("%s %s %s\n", s.ID, s.Type, s.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	}
	return nil
}

func handleGet(ctx context.Context, svc *clientservice.Service, args []string) error {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	var id string
	fs.StringVar(&id, "id", "", "secret id")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse get flags: %w", err)
	}

	secret, err := svc.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("get error: %w", err)
	}
	printFmt("id=%s type=%s payload=%s meta=%v updated_at=%s\n",
		secret.ID, secret.Type, strings.TrimSpace(string(secret.Payload)), secret.Meta, secret.UpdatedAt.Format(time.RFC3339))
	return nil
}

func handleUpdate(ctx context.Context, svc *clientservice.Service, args []string) error {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	var id, typ, payload, meta string
	fs.StringVar(&id, "id", "", "secret id")
	fs.StringVar(&typ, "type", "", "secret type")
	fs.StringVar(&payload, "payload", "", "secret payload")
	fs.StringVar(&meta, "meta", "", "meta key=value pairs separated by comma")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse update flags: %w", err)
	}

	id, err := svc.Update(ctx, id, typ, payload, meta)
	if err != nil {
		return fmt.Errorf("update error: %w", err)
	}
	printLine(id)
	return nil
}

func handleDelete(ctx context.Context, svc *clientservice.Service, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	var id string
	fs.StringVar(&id, "id", "", "secret id")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse delete flags: %w", err)
	}

	if err := svc.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete error: %w", err)
	}
	printLine("deleted")
	return nil
}

func handleSync(ctx context.Context, svc *clientservice.Service) error {
	items, err := svc.Sync(ctx)
	if err != nil {
		return fmt.Errorf("sync error: %w", err)
	}

	for _, item := range items {
		status := "active"
		if item.Deleted {
			status = "deleted"
		}
		printFmt("%s %s %s %s\n", item.ID, item.Type, status, item.UpdatedAt.Format(time.RFC3339))
	}
	printFmt("synced %d items\n", len(items))
	return nil
}

func printClientUsage() {
	printLine("Usage:")
	printLine("  client [--config path] [--master-password pwd] <command> [flags]")
	printLine("Commands:")
	printLine("  register --email --password")
	printLine("  login    --email --password")
	printLine("  add      --type --payload [--meta k=v,...]")
	printLine("  list")
	printLine("  get      --id")
	printLine("  update   --id --type --payload [--meta k=v,...]")
	printLine("  delete   --id")
	printLine("  sync")
	printLine("  ui")
	printLine("  version")
}

func printLine(line string) {
	if _, err := fmt.Fprintln(os.Stdout, line); err != nil {
		_ = err
	}
}

func printFmt(format string, args ...any) {
	if _, err := fmt.Fprintf(os.Stdout, format, args...); err != nil {
		_ = err
	}
}

func printErrLine(line string) {
	if _, err := fmt.Fprintln(os.Stderr, line); err != nil {
		_ = err
	}
}
