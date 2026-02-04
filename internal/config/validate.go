package config

import (
	"errors"
	"fmt"
	"strings"
)

// Validate validates server configuration values.
func (c ServerConfig) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return errors.New("server host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("server port out of range: %d", c.Port)
	}
	switch strings.ToLower(strings.TrimSpace(c.Storage.Type)) {
	case "", "memory", "sqlite", "postgres":
	default:
		return fmt.Errorf("unsupported storage type: %s", c.Storage.Type)
	}
	if strings.ToLower(strings.TrimSpace(c.Storage.Type)) == "postgres" && strings.TrimSpace(c.Storage.DSN) == "" {
		return errors.New("storage dsn is required for postgres")
	}
	if strings.TrimSpace(c.Log.Level) == "" {
		return errors.New("log level is required")
	}
	switch strings.ToLower(strings.TrimSpace(c.Log.Format)) {
	case "", "text", "json":
	default:
		return fmt.Errorf("unsupported log format: %s", c.Log.Format)
	}
	if strings.TrimSpace(c.JWTKey) == "" {
		return errors.New("jwt_key is required")
	}
	if c.TLS.Enabled {
		if strings.TrimSpace(c.TLS.CertFile) == "" || strings.TrimSpace(c.TLS.KeyFile) == "" {
			return errors.New("tls cert and key are required when tls is enabled")
		}
	}
	return nil
}

// Validate validates client configuration values.
func (c ClientConfig) Validate() error {
	if strings.TrimSpace(c.ServerURL) == "" {
		return errors.New("server_url is required")
	}
	if strings.TrimSpace(c.DataDir) == "" {
		return errors.New("data_dir is required")
	}
	switch strings.ToLower(strings.TrimSpace(c.Log.Format)) {
	case "", "text", "json":
	default:
		return fmt.Errorf("unsupported log format: %s", c.Log.Format)
	}
	if strings.TrimSpace(c.Log.Level) == "" {
		return errors.New("log level is required")
	}
	return nil
}
