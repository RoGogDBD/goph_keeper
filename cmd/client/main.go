package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"goph_keeper/internal/client/api"
	"goph_keeper/internal/client/crypto"
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
	defer func() {
		if cerr := local.Close(); cerr != nil {
			_ = cerr
		}
	}()
	ctx := context.Background()

	switch rest[0] {
	case "register":
		if err := handleRegister(ctx, cli, rest[1:]); err != nil {
			return err
		}
	case "login":
		if err := handleLogin(ctx, cli, rest[1:]); err != nil {
			return err
		}
	case "add":
		if err := handleAdd(ctx, local, cryptoSvc, rest[1:]); err != nil {
			return err
		}
	case "list":
		if err := handleList(ctx, local); err != nil {
			return err
		}
	case "get":
		if err := handleGet(ctx, local, cryptoSvc, rest[1:]); err != nil {
			return err
		}
	case "update":
		if err := handleUpdate(ctx, local, cryptoSvc, rest[1:]); err != nil {
			return err
		}
	case "delete":
		if err := handleDelete(ctx, local, rest[1:]); err != nil {
			return err
		}
	case "sync":
		if err := handleSync(ctx, cli, local, cfg.DataDir); err != nil {
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

func handleRegister(ctx context.Context, cli *api.Client, args []string) error {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	var email, password string
	fs.StringVar(&email, "email", "", "user email")
	fs.StringVar(&password, "password", "", "user password")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse register flags: %w", err)
	}

	if email == "" || password == "" {
		return errors.New("email and password are required")
	}

	if err := cli.Register(ctx, api.RegisterRequest{Email: email, Password: password}); err != nil {
		return fmt.Errorf("register error: %w", err)
	}
	printLine("registered")
	return nil
}

func handleLogin(ctx context.Context, cli *api.Client, args []string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	var email, password string
	fs.StringVar(&email, "email", "", "user email")
	fs.StringVar(&password, "password", "", "user password")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse login flags: %w", err)
	}

	if email == "" || password == "" {
		return errors.New("email and password are required")
	}

	if err := cli.Login(ctx, api.LoginRequest{Email: email, Password: password}); err != nil {
		return fmt.Errorf("login error: %w", err)
	}
	printLine("logged in")
	return nil
}

func handleAdd(ctx context.Context, local *store.LocalStore, cryptoSvc *crypto.Crypto, args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	var typ, payload, meta string
	fs.StringVar(&typ, "type", "", "secret type")
	fs.StringVar(&payload, "payload", "", "secret payload")
	fs.StringVar(&meta, "meta", "", "meta key=value pairs separated by comma")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse add flags: %w", err)
	}

	if typ == "" || payload == "" {
		return errors.New("type and payload are required")
	}

	now := time.Now().UTC()
	item := store.Item{
		ID:        newID(),
		Type:      typ,
		Payload:   []byte(payload),
		Meta:      store.ParseMeta(meta),
		Deleted:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if cryptoSvc == nil {
		return errors.New("master password is required")
	}
	encPayload, err := cryptoSvc.Encrypt(item.Payload)
	if err != nil {
		return fmt.Errorf("encrypt error: %w", err)
	}
	item.Payload = encPayload

	if err := local.Upsert(ctx, item, true); err != nil {
		return fmt.Errorf("local save error: %w", err)
	}
	printLine(item.ID)
	return nil
}

func handleList(ctx context.Context, local *store.LocalStore) error {
	list, err := local.List(ctx, false)
	if err != nil {
		return fmt.Errorf("list error: %w", err)
	}
	for _, s := range list {
		printFmt("%s %s %s\n", s.ID, s.Type, s.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	}
	return nil
}

func handleGet(ctx context.Context, local *store.LocalStore, cryptoSvc *crypto.Crypto, args []string) error {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	var id string
	fs.StringVar(&id, "id", "", "secret id")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse get flags: %w", err)
	}

	if id == "" {
		return errors.New("id is required")
	}

	secret, err := local.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("get error: %w", err)
	}
	if cryptoSvc == nil {
		return errors.New("master password is required")
	}
	if len(secret.Payload) > 0 {
		dec, err := cryptoSvc.Decrypt(secret.Payload)
		if err != nil {
			return fmt.Errorf("decrypt error: %w", err)
		}
		secret.Payload = dec
	}
	printFmt("id=%s type=%s payload=%s meta=%v updated_at=%s\n",
		secret.ID, secret.Type, strings.TrimSpace(string(secret.Payload)), secret.Meta, secret.UpdatedAt.Format(time.RFC3339))
	return nil
}

func handleUpdate(ctx context.Context, local *store.LocalStore, cryptoSvc *crypto.Crypto, args []string) error {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	var id, typ, payload, meta string
	fs.StringVar(&id, "id", "", "secret id")
	fs.StringVar(&typ, "type", "", "secret type")
	fs.StringVar(&payload, "payload", "", "secret payload")
	fs.StringVar(&meta, "meta", "", "meta key=value pairs separated by comma")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse update flags: %w", err)
	}

	if id == "" || typ == "" || payload == "" {
		return errors.New("id, type and payload are required")
	}

	if cryptoSvc == nil {
		return errors.New("master password is required")
	}

	createdAt := time.Now().UTC()
	if existing, err := local.Get(ctx, id); err == nil {
		createdAt = existing.CreatedAt
	}

	item := store.Item{
		ID:        id,
		Type:      typ,
		Payload:   []byte(payload),
		Meta:      store.ParseMeta(meta),
		Deleted:   false,
		CreatedAt: createdAt,
		UpdatedAt: time.Now().UTC(),
	}
	enc, err := cryptoSvc.Encrypt(item.Payload)
	if err != nil {
		return fmt.Errorf("encrypt error: %w", err)
	}
	item.Payload = enc
	if err := local.Upsert(ctx, item, true); err != nil {
		return fmt.Errorf("local update error: %w", err)
	}
	printLine(item.ID)
	return nil
}

func handleDelete(ctx context.Context, local *store.LocalStore, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	var id string
	fs.StringVar(&id, "id", "", "secret id")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse delete flags: %w", err)
	}

	if id == "" {
		return errors.New("id is required")
	}

	createdAt := time.Now().UTC()
	itemType := "deleted"
	var payload []byte
	if existing, err := local.Get(ctx, id); err == nil {
		createdAt = existing.CreatedAt
		itemType = existing.Type
		payload = existing.Payload
	}

	item := store.Item{
		ID:        id,
		Type:      itemType,
		Payload:   payload,
		Deleted:   true,
		CreatedAt: createdAt,
		UpdatedAt: time.Now().UTC(),
	}
	if err := local.Upsert(ctx, item, true); err != nil {
		return fmt.Errorf("local delete error: %w", err)
	}
	printLine("deleted")
	return nil
}

func handleSync(ctx context.Context, cli *api.Client, local *store.LocalStore, dataDir string) error {
	syncStore := store.NewSyncStore(dataDir)
	since, err := syncStore.Load()
	if err != nil && !errors.Is(err, store.ErrSyncNotFound) {
		return fmt.Errorf("sync state error: %w", err)
	}

	dirty, err := local.ListDirty(ctx)
	if err != nil {
		return fmt.Errorf("local dirty error: %w", err)
	}
	if len(dirty) > 0 {
		_, err = cli.SyncPushEncrypted(ctx, toSyncItems(dirty))
		if err != nil {
			return fmt.Errorf("sync push error: %w", err)
		}
		var ids []string
		for _, item := range dirty {
			ids = append(ids, item.ID)
		}
		if err := local.MarkClean(ctx, ids); err != nil {
			return fmt.Errorf("mark clean error: %w", err)
		}
	}

	items, err := cli.SyncPullEncrypted(ctx, since)
	if err != nil {
		return fmt.Errorf("sync pull error: %w", err)
	}

	if err := local.ApplyRemote(ctx, fromSyncItems(items)); err != nil {
		return fmt.Errorf("apply remote error: %w", err)
	}

	for _, item := range items {
		status := "active"
		if item.Deleted {
			status = "deleted"
		}
		printFmt("%s %s %s %s\n", item.ID, item.Type, status, item.UpdatedAt.Format(time.RFC3339))
	}

	if err := syncStore.Save(time.Now().UTC()); err != nil {
		return fmt.Errorf("sync save error: %w", err)
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

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func toSyncItems(items []store.Item) []api.SyncItem {
	out := make([]api.SyncItem, 0, len(items))
	for _, it := range items {
		out = append(out, api.SyncItem{
			ID:        it.ID,
			Type:      it.Type,
			Payload:   it.Payload,
			Meta:      it.Meta,
			Deleted:   it.Deleted,
			CreatedAt: it.CreatedAt,
			UpdatedAt: it.UpdatedAt,
		})
	}
	return out
}

func fromSyncItems(items []api.SyncItem) []store.Item {
	out := make([]store.Item, 0, len(items))
	for _, it := range items {
		out = append(out, store.Item{
			ID:        it.ID,
			Type:      it.Type,
			Payload:   it.Payload,
			Meta:      it.Meta,
			Deleted:   it.Deleted,
			CreatedAt: it.CreatedAt,
			UpdatedAt: it.UpdatedAt,
		})
	}
	return out
}
