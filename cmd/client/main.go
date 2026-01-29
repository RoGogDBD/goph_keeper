package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
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
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version.Info())
		return
	}

	var fo config.ClientFlagOverrides
	var showVersion bool
	var masterPassword string
	root := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	config.RegisterClientFlags(root, &fo)
	root.BoolVar(&showVersion, "version", false, "print version and exit")
	root.StringVar(&masterPassword, "master-password", "", "master password for encryption")
	_ = root.Parse(os.Args[1:])

	if showVersion {
		fmt.Println(version.Info())
		return
	}

	args := root.Args()
	if len(args) == 0 {
		printClientUsage()
		os.Exit(1)
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

	tokenStore := store.NewTokenStore(cfg.DataDir)
	cryptoSvc, err := buildCrypto(masterPassword, cfg.DataDir, args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "crypto error:", err)
		os.Exit(1)
	}
	cli := api.New(cfg.ServerURL, tokenStore, cryptoSvc)
	local, err := store.NewLocalStore(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "local store error:", err)
		os.Exit(1)
	}
	defer local.Close()
	ctx := context.Background()

	switch args[0] {
	case "register":
		handleRegister(ctx, cli, args[1:])
	case "login":
		handleLogin(ctx, cli, args[1:])
	case "add":
		handleAdd(ctx, local, cryptoSvc, args[1:])
	case "list":
		handleList(ctx, local)
	case "get":
		handleGet(ctx, local, cryptoSvc, args[1:])
	case "update":
		handleUpdate(ctx, local, cryptoSvc, args[1:])
	case "delete":
		handleDelete(ctx, local, args[1:])
	case "sync":
		handleSync(ctx, cli, local, cfg.DataDir)
	case "ui":
		if err := ui.RunUI(cfg.ServerURL, cfg.DataDir); err != nil {
			fmt.Fprintln(os.Stderr, "ui error:", err)
			os.Exit(1)
		}
	default:
		printClientUsage()
		os.Exit(1)
	}
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

func handleRegister(ctx context.Context, cli *api.Client, args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	var email, password string
	fs.StringVar(&email, "email", "", "user email")
	fs.StringVar(&password, "password", "", "user password")
	_ = fs.Parse(args)

	if email == "" || password == "" {
		fmt.Fprintln(os.Stderr, "email and password are required")
		os.Exit(1)
	}

	if err := cli.Register(ctx, api.RegisterRequest{Email: email, Password: password}); err != nil {
		fmt.Fprintln(os.Stderr, "register error:", err)
		os.Exit(1)
	}
	fmt.Println("registered")
}

func handleLogin(ctx context.Context, cli *api.Client, args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	var email, password string
	fs.StringVar(&email, "email", "", "user email")
	fs.StringVar(&password, "password", "", "user password")
	_ = fs.Parse(args)

	if email == "" || password == "" {
		fmt.Fprintln(os.Stderr, "email and password are required")
		os.Exit(1)
	}

	if err := cli.Login(ctx, api.LoginRequest{Email: email, Password: password}); err != nil {
		fmt.Fprintln(os.Stderr, "login error:", err)
		os.Exit(1)
	}
	fmt.Println("logged in")
}

func handleAdd(ctx context.Context, local *store.LocalStore, cryptoSvc *crypto.Crypto, args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	var typ, payload, meta string
	fs.StringVar(&typ, "type", "", "secret type")
	fs.StringVar(&payload, "payload", "", "secret payload")
	fs.StringVar(&meta, "meta", "", "meta key=value pairs separated by comma")
	_ = fs.Parse(args)

	if typ == "" || payload == "" {
		fmt.Fprintln(os.Stderr, "type and payload are required")
		os.Exit(1)
	}

	now := time.Now().UTC()
	item := api.SyncItem{
		ID:        newID(),
		Type:      typ,
		Payload:   []byte(payload),
		Meta:      store.ParseMeta(meta),
		Deleted:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if cryptoSvc == nil {
		fmt.Fprintln(os.Stderr, "master password is required")
		os.Exit(1)
	}
	encPayload, err := cryptoSvc.Encrypt(item.Payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "encrypt error:", err)
		os.Exit(1)
	}
	item.Payload = encPayload

	if err := local.Upsert(ctx, item, true); err != nil {
		fmt.Fprintln(os.Stderr, "local save error:", err)
		os.Exit(1)
	}
	fmt.Println(item.ID)
}

func handleList(ctx context.Context, local *store.LocalStore) {
	list, err := local.List(ctx, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list error:", err)
		os.Exit(1)
	}
	for _, s := range list {
		fmt.Printf("%s %s %s\n", s.ID, s.Type, s.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	}
}

func handleGet(ctx context.Context, local *store.LocalStore, cryptoSvc *crypto.Crypto, args []string) {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	var id string
	fs.StringVar(&id, "id", "", "secret id")
	_ = fs.Parse(args)

	if id == "" {
		fmt.Fprintln(os.Stderr, "id is required")
		os.Exit(1)
	}

	secret, err := local.Get(ctx, id)
	if err != nil {
		fmt.Fprintln(os.Stderr, "get error:", err)
		os.Exit(1)
	}
	if cryptoSvc == nil {
		fmt.Fprintln(os.Stderr, "master password is required")
		os.Exit(1)
	}
	if len(secret.Payload) > 0 {
		dec, err := cryptoSvc.Decrypt(secret.Payload)
		if err != nil {
			fmt.Fprintln(os.Stderr, "decrypt error:", err)
			os.Exit(1)
		}
		secret.Payload = dec
	}
	fmt.Printf("id=%s type=%s payload=%s meta=%v updated_at=%s\n",
		secret.ID, secret.Type, strings.TrimSpace(string(secret.Payload)), secret.Meta, secret.UpdatedAt.Format(time.RFC3339))
}

func handleUpdate(ctx context.Context, local *store.LocalStore, cryptoSvc *crypto.Crypto, args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	var id, typ, payload, meta string
	fs.StringVar(&id, "id", "", "secret id")
	fs.StringVar(&typ, "type", "", "secret type")
	fs.StringVar(&payload, "payload", "", "secret payload")
	fs.StringVar(&meta, "meta", "", "meta key=value pairs separated by comma")
	_ = fs.Parse(args)

	if id == "" || typ == "" || payload == "" {
		fmt.Fprintln(os.Stderr, "id, type and payload are required")
		os.Exit(1)
	}

	if cryptoSvc == nil {
		fmt.Fprintln(os.Stderr, "master password is required")
		os.Exit(1)
	}

	createdAt := time.Now().UTC()
	if existing, err := local.Get(ctx, id); err == nil {
		createdAt = existing.CreatedAt
	}

	item := api.SyncItem{
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
		fmt.Fprintln(os.Stderr, "encrypt error:", err)
		os.Exit(1)
	}
	item.Payload = enc
	if err := local.Upsert(ctx, item, true); err != nil {
		fmt.Fprintln(os.Stderr, "local update error:", err)
		os.Exit(1)
	}
	fmt.Println(item.ID)
}

func handleDelete(ctx context.Context, local *store.LocalStore, args []string) {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	var id string
	fs.StringVar(&id, "id", "", "secret id")
	_ = fs.Parse(args)

	if id == "" {
		fmt.Fprintln(os.Stderr, "id is required")
		os.Exit(1)
	}

	createdAt := time.Now().UTC()
	itemType := "deleted"
	payload := []byte{}
	if existing, err := local.Get(ctx, id); err == nil {
		createdAt = existing.CreatedAt
		itemType = existing.Type
		payload = existing.Payload
	}

	item := api.SyncItem{
		ID:        id,
		Type:      itemType,
		Payload:   payload,
		Deleted:   true,
		CreatedAt: createdAt,
		UpdatedAt: time.Now().UTC(),
	}
	if err := local.Upsert(ctx, item, true); err != nil {
		fmt.Fprintln(os.Stderr, "local delete error:", err)
		os.Exit(1)
	}
	fmt.Println("deleted")
}

func handleSync(ctx context.Context, cli *api.Client, local *store.LocalStore, dataDir string) {
	syncStore := store.NewSyncStore(dataDir)
	since, err := syncStore.Load()
	if err != nil && err != store.ErrSyncNotFound {
		fmt.Fprintln(os.Stderr, "sync state error:", err)
		os.Exit(1)
	}

	dirty, err := local.ListDirty(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "local dirty error:", err)
		os.Exit(1)
	}
	if len(dirty) > 0 {
		_, err = cli.SyncPushEncrypted(ctx, dirty)
		if err != nil {
			fmt.Fprintln(os.Stderr, "sync push error:", err)
			os.Exit(1)
		}
		var ids []string
		for _, item := range dirty {
			ids = append(ids, item.ID)
		}
		if err := local.MarkClean(ctx, ids); err != nil {
			fmt.Fprintln(os.Stderr, "mark clean error:", err)
			os.Exit(1)
		}
	}

	items, err := cli.SyncPullEncrypted(ctx, since)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync pull error:", err)
		os.Exit(1)
	}

	if err := local.ApplyRemote(ctx, items); err != nil {
		fmt.Fprintln(os.Stderr, "apply remote error:", err)
		os.Exit(1)
	}

	for _, item := range items {
		status := "active"
		if item.Deleted {
			status = "deleted"
		}
		fmt.Printf("%s %s %s %s\n", item.ID, item.Type, status, item.UpdatedAt.Format(time.RFC3339))
	}

	if err := syncStore.Save(time.Now().UTC()); err != nil {
		fmt.Fprintln(os.Stderr, "sync save error:", err)
		os.Exit(1)
	}
	fmt.Printf("synced %d items\n", len(items))
}

func printClientUsage() {
	fmt.Println("Usage:")
	fmt.Println("  client [--config path] [--master-password pwd] <command> [flags]")
	fmt.Println("Commands:")
	fmt.Println("  register --email --password")
	fmt.Println("  login    --email --password")
	fmt.Println("  add      --type --payload [--meta k=v,...]")
	fmt.Println("  list")
	fmt.Println("  get      --id")
	fmt.Println("  update   --id --type --payload [--meta k=v,...]")
	fmt.Println("  delete   --id")
	fmt.Println("  sync")
	fmt.Println("  ui")
	fmt.Println("  version")
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
