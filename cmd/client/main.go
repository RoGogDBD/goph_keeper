package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"goph_keeper/internal/client"
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
	root := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	config.RegisterClientFlags(root, &fo)
	root.BoolVar(&showVersion, "version", false, "print version and exit")
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

	store := client.NewTokenStore(cfg.DataDir)
	cli := client.New(cfg.ServerURL, store)
	ctx := context.Background()

	switch args[0] {
	case "register":
		handleRegister(ctx, cli, args[1:])
	case "login":
		handleLogin(ctx, cli, args[1:])
	case "add":
		handleAdd(ctx, cli, args[1:])
	case "list":
		handleList(ctx, cli)
	case "get":
		handleGet(ctx, cli, args[1:])
	case "update":
		handleUpdate(ctx, cli, args[1:])
	case "delete":
		handleDelete(ctx, cli, args[1:])
	default:
		printClientUsage()
		os.Exit(1)
	}
}

func handleRegister(ctx context.Context, cli *client.Client, args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	var email, password string
	fs.StringVar(&email, "email", "", "user email")
	fs.StringVar(&password, "password", "", "user password")
	_ = fs.Parse(args)

	if email == "" || password == "" {
		fmt.Fprintln(os.Stderr, "email and password are required")
		os.Exit(1)
	}

	if err := cli.Register(ctx, client.RegisterRequest{Email: email, Password: password}); err != nil {
		fmt.Fprintln(os.Stderr, "register error:", err)
		os.Exit(1)
	}
	fmt.Println("registered")
}

func handleLogin(ctx context.Context, cli *client.Client, args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	var email, password string
	fs.StringVar(&email, "email", "", "user email")
	fs.StringVar(&password, "password", "", "user password")
	_ = fs.Parse(args)

	if email == "" || password == "" {
		fmt.Fprintln(os.Stderr, "email and password are required")
		os.Exit(1)
	}

	if err := cli.Login(ctx, client.LoginRequest{Email: email, Password: password}); err != nil {
		fmt.Fprintln(os.Stderr, "login error:", err)
		os.Exit(1)
	}
	fmt.Println("logged in")
}

func handleAdd(ctx context.Context, cli *client.Client, args []string) {
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

	resp, err := cli.CreateSecret(ctx, client.SecretRequest{
		Type:    typ,
		Payload: []byte(payload),
		Meta:    client.ParseMeta(meta),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "create error:", err)
		os.Exit(1)
	}
	fmt.Println(resp.ID)
}

func handleList(ctx context.Context, cli *client.Client) {
	list, err := cli.ListSecrets(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list error:", err)
		os.Exit(1)
	}
	for _, s := range list {
		fmt.Printf("%s %s %s\n", s.ID, s.Type, s.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	}
}

func handleGet(ctx context.Context, cli *client.Client, args []string) {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	var id string
	fs.StringVar(&id, "id", "", "secret id")
	_ = fs.Parse(args)

	if id == "" {
		fmt.Fprintln(os.Stderr, "id is required")
		os.Exit(1)
	}

	secret, err := cli.GetSecret(ctx, id)
	if err != nil {
		fmt.Fprintln(os.Stderr, "get error:", err)
		os.Exit(1)
	}
	fmt.Printf("id=%s type=%s payload=%s meta=%v updated_at=%s\n",
		secret.ID, secret.Type, strings.TrimSpace(string(secret.Payload)), secret.Meta, secret.UpdatedAt.Format(time.RFC3339))
}

func handleUpdate(ctx context.Context, cli *client.Client, args []string) {
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

	resp, err := cli.UpdateSecret(ctx, id, client.SecretRequest{
		Type:    typ,
		Payload: []byte(payload),
		Meta:    client.ParseMeta(meta),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "update error:", err)
		os.Exit(1)
	}
	fmt.Println(resp.ID)
}

func handleDelete(ctx context.Context, cli *client.Client, args []string) {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	var id string
	fs.StringVar(&id, "id", "", "secret id")
	_ = fs.Parse(args)

	if id == "" {
		fmt.Fprintln(os.Stderr, "id is required")
		os.Exit(1)
	}

	if err := cli.DeleteSecret(ctx, id); err != nil {
		fmt.Fprintln(os.Stderr, "delete error:", err)
		os.Exit(1)
	}
	fmt.Println("deleted")
}

func printClientUsage() {
	fmt.Println("Usage:")
	fmt.Println("  client [--config path] <command> [flags]")
	fmt.Println("Commands:")
	fmt.Println("  register --email --password")
	fmt.Println("  login    --email --password")
	fmt.Println("  add      --type --payload [--meta k=v,...]")
	fmt.Println("  list")
	fmt.Println("  get      --id")
	fmt.Println("  update   --id --type --payload [--meta k=v,...]")
	fmt.Println("  delete   --id")
	fmt.Println("  version")
}
