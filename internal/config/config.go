package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type StorageConfig struct {
	Type string `json:"type" yaml:"type"`
	DSN  string `json:"dsn" yaml:"dsn"`
}

type LogConfig struct {
	Level  string `json:"level" yaml:"level"`
	Format string `json:"format" yaml:"format"`
}

type ServerConfig struct {
	Host    string        `json:"host" yaml:"host"`
	Port    int           `json:"port" yaml:"port"`
	Storage StorageConfig `json:"storage" yaml:"storage"`
	Log     LogConfig     `json:"log" yaml:"log"`
	JWTKey  string        `json:"jwt_key" yaml:"jwt_key"`
}

type ClientConfig struct {
	ServerURL string    `json:"server_url" yaml:"server_url"`
	DataDir   string    `json:"data_dir" yaml:"data_dir"`
	TLS       bool      `json:"tls" yaml:"tls"`
	TLSCA     string    `json:"tls_ca" yaml:"tls_ca"`
	Log       LogConfig `json:"log" yaml:"log"`
}

type ServerFlagOverrides struct {
	ConfigPath string
	Host       string
	Port       int
	Storage    string
	DSN        string
	LogLevel   string
	LogFormat  string
	JWTKey     string
}

type ClientFlagOverrides struct {
	ConfigPath string
	ServerURL  string
	DataDir    string
	TLS        string
	TLSCA      string
	LogLevel   string
	LogFormat  string
}

func RegisterServerFlags(fs *flag.FlagSet, fo *ServerFlagOverrides) {
	fs.StringVar(&fo.ConfigPath, "config", "", "path to config file (yaml or json)")
	fs.StringVar(&fo.Host, "host", "", "server host")
	fs.IntVar(&fo.Port, "port", 0, "server port")
	fs.StringVar(&fo.Storage, "storage", "", "storage type: memory|sqlite|postgres")
	fs.StringVar(&fo.DSN, "dsn", "", "storage DSN")
	fs.StringVar(&fo.LogLevel, "log-level", "", "log level")
	fs.StringVar(&fo.LogFormat, "log-format", "", "log format: text|json")
	fs.StringVar(&fo.JWTKey, "jwt-key", "", "JWT signing key")
}

func RegisterClientFlags(fs *flag.FlagSet, fo *ClientFlagOverrides) {
	fs.StringVar(&fo.ConfigPath, "config", "", "path to config file (yaml or json)")
	fs.StringVar(&fo.ServerURL, "server-url", "", "server base URL")
	fs.StringVar(&fo.DataDir, "data-dir", "", "client data directory")
	fs.StringVar(&fo.TLS, "tls", "", "enable TLS (true/false)")
	fs.StringVar(&fo.TLSCA, "tls-ca", "", "path to TLS CA bundle")
	fs.StringVar(&fo.LogLevel, "log-level", "", "log level")
	fs.StringVar(&fo.LogFormat, "log-format", "", "log format: text|json")
}

func LoadServerConfig(fo ServerFlagOverrides) (ServerConfig, error) {
	cfg := defaultServerConfig()

	configPath := firstNonEmpty(fo.ConfigPath, os.Getenv("GOPHKEEPER_CONFIG"))
	if configPath != "" {
		if err := loadConfigFile(configPath, &cfg); err != nil {
			return ServerConfig{}, err
		}
	}

	applyServerEnv(&cfg)
	applyServerFlags(&cfg, fo)

	return cfg, nil
}

func LoadClientConfig(fo ClientFlagOverrides) (ClientConfig, error) {
	cfg := defaultClientConfig()

	configPath := firstNonEmpty(fo.ConfigPath, os.Getenv("GOPHKEEPER_CONFIG"))
	if configPath != "" {
		if err := loadConfigFile(configPath, &cfg); err != nil {
			return ClientConfig{}, err
		}
	}

	applyClientEnv(&cfg)
	applyClientFlags(&cfg, fo)

	return cfg, nil
}

func defaultServerConfig() ServerConfig {
	return ServerConfig{
		Host: "0.0.0.0",
		Port: 8080,
		Storage: StorageConfig{
			Type: "memory",
			DSN:  "",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		JWTKey: "",
	}
}

func defaultClientConfig() ClientConfig {
	return ClientConfig{
		ServerURL: "http://127.0.0.1:8080",
		DataDir:   ".gophkeeper",
		TLS:       false,
		TLSCA:     "",
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func loadConfigFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, target); err != nil {
			return fmt.Errorf("parse json config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, target); err != nil {
			return fmt.Errorf("parse yaml config: %w", err)
		}
	default:
		return errors.New("unsupported config format: use .yaml, .yml, or .json")
	}

	return nil
}

func applyServerEnv(cfg *ServerConfig) {
	if v := os.Getenv("GOPHKEEPER_SERVER_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("GOPHKEEPER_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Port = port
		}
	}
	if v := os.Getenv("GOPHKEEPER_SERVER_STORAGE"); v != "" {
		cfg.Storage.Type = v
	}
	if v := os.Getenv("GOPHKEEPER_SERVER_DSN"); v != "" {
		cfg.Storage.DSN = v
	}
	if v := os.Getenv("GOPHKEEPER_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("GOPHKEEPER_LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}
	if v := os.Getenv("GOPHKEEPER_SERVER_JWT_KEY"); v != "" {
		cfg.JWTKey = v
	}
}

func applyClientEnv(cfg *ClientConfig) {
	if v := os.Getenv("GOPHKEEPER_CLIENT_SERVER_URL"); v != "" {
		cfg.ServerURL = v
	}
	if v := os.Getenv("GOPHKEEPER_CLIENT_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("GOPHKEEPER_CLIENT_TLS"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.TLS = parsed
		}
	}
	if v := os.Getenv("GOPHKEEPER_CLIENT_TLS_CA"); v != "" {
		cfg.TLSCA = v
	}
	if v := os.Getenv("GOPHKEEPER_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("GOPHKEEPER_LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}
}

func applyServerFlags(cfg *ServerConfig, fo ServerFlagOverrides) {
	if fo.Host != "" {
		cfg.Host = fo.Host
	}
	if fo.Port != 0 {
		cfg.Port = fo.Port
	}
	if fo.Storage != "" {
		cfg.Storage.Type = fo.Storage
	}
	if fo.DSN != "" {
		cfg.Storage.DSN = fo.DSN
	}
	if fo.LogLevel != "" {
		cfg.Log.Level = fo.LogLevel
	}
	if fo.LogFormat != "" {
		cfg.Log.Format = fo.LogFormat
	}
	if fo.JWTKey != "" {
		cfg.JWTKey = fo.JWTKey
	}
}

func applyClientFlags(cfg *ClientConfig, fo ClientFlagOverrides) {
	if fo.ServerURL != "" {
		cfg.ServerURL = fo.ServerURL
	}
	if fo.DataDir != "" {
		cfg.DataDir = fo.DataDir
	}
	if fo.TLS != "" {
		if parsed, err := strconv.ParseBool(fo.TLS); err == nil {
			cfg.TLS = parsed
		}
	}
	if fo.TLSCA != "" {
		cfg.TLSCA = fo.TLSCA
	}
	if fo.LogLevel != "" {
		cfg.Log.Level = fo.LogLevel
	}
	if fo.LogFormat != "" {
		cfg.Log.Format = fo.LogFormat
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
