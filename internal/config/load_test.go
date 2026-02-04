package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadServerConfigEnvAndFlags(t *testing.T) {
	t.Setenv("GOPHKEEPER_SERVER_HOST", "127.0.0.1")
	t.Setenv("GOPHKEEPER_SERVER_PORT", "9090")
	t.Setenv("GOPHKEEPER_SERVER_STORAGE", "postgres")
	t.Setenv("GOPHKEEPER_SERVER_DSN", "dsn")
	t.Setenv("GOPHKEEPER_SERVER_JWT_KEY", "key")
	t.Setenv("GOPHKEEPER_LOG_LEVEL", "debug")
	t.Setenv("GOPHKEEPER_LOG_FORMAT", "json")
	t.Setenv("GOPHKEEPER_SERVER_TLS", "true")
	t.Setenv("GOPHKEEPER_SERVER_TLS_CERT", "/tmp/cert")
	t.Setenv("GOPHKEEPER_SERVER_TLS_KEY", "/tmp/key")

	cfg, err := LoadServerConfig(ServerFlagOverrides{})
	if err != nil {
		t.Fatalf("LoadServerConfig: %v", err)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != 9090 {
		t.Fatalf("env not applied")
	}
	if !cfg.TLS.Enabled || cfg.TLS.CertFile == "" || cfg.TLS.KeyFile == "" {
		t.Fatalf("tls env not applied")
	}

	override := ServerFlagOverrides{
		Host:   "0.0.0.0",
		Port:   8081,
		JWTKey: "override",
	}
	cfg, err = LoadServerConfig(override)
	if err != nil {
		t.Fatalf("LoadServerConfig override: %v", err)
	}
	if cfg.Host != "0.0.0.0" || cfg.Port != 8081 || cfg.JWTKey != "override" {
		t.Fatalf("flags not applied")
	}
}

func TestLoadClientConfigEnvAndFlags(t *testing.T) {
	t.Setenv("GOPHKEEPER_CLIENT_SERVER_URL", "http://localhost:9000")
	t.Setenv("GOPHKEEPER_CLIENT_DATA_DIR", "data")
	t.Setenv("GOPHKEEPER_CLIENT_TLS", "true")
	t.Setenv("GOPHKEEPER_CLIENT_TLS_CA", "/tmp/ca")
	t.Setenv("GOPHKEEPER_LOG_LEVEL", "debug")
	t.Setenv("GOPHKEEPER_LOG_FORMAT", "json")

	cfg, err := LoadClientConfig(ClientFlagOverrides{})
	if err != nil {
		t.Fatalf("LoadClientConfig: %v", err)
	}
	if cfg.ServerURL != "http://localhost:9000" || cfg.DataDir != "data" || !cfg.TLS {
		t.Fatalf("env not applied")
	}

	override := ClientFlagOverrides{
		ServerURL: "http://override",
		DataDir:   "override",
		TLS:       "false",
	}
	cfg, err = LoadClientConfig(override)
	if err != nil {
		t.Fatalf("LoadClientConfig override: %v", err)
	}
	if cfg.ServerURL != "http://override" || cfg.DataDir != "override" || cfg.TLS {
		t.Fatalf("flags not applied")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.yaml")
	data := []byte(`
host: "0.0.0.0"
port: 8080
storage:
  type: "postgres"
  dsn: "dsn"
log:
  level: "info"
  format: "text"
jwt_key: "key"
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := LoadServerConfig(ServerFlagOverrides{ConfigPath: path})
	if err != nil {
		t.Fatalf("LoadServerConfig: %v", err)
	}
	if cfg.Storage.Type != "postgres" || cfg.JWTKey != "key" {
		t.Fatalf("file not applied")
	}
}

func TestRegisterFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	var sfo ServerFlagOverrides
	RegisterServerFlags(fs, &sfo)
	if fs.Lookup("host") == nil || fs.Lookup("jwt-key") == nil || fs.Lookup("tls") == nil {
		t.Fatalf("server flags missing")
	}

	fs = flag.NewFlagSet("test", flag.ContinueOnError)
	var cfo ClientFlagOverrides
	RegisterClientFlags(fs, &cfo)
	if fs.Lookup("server-url") == nil || fs.Lookup("tls") == nil {
		t.Fatalf("client flags missing")
	}
}
