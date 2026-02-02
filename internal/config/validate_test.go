package config

import "testing"

func TestServerConfigValidate(t *testing.T) {
	t.Parallel()

	valid := ServerConfig{
		Host: "0.0.0.0",
		Port: 8080,
		Storage: StorageConfig{
			Type: "postgres",
			DSN:  "postgresql://user:pass@localhost:5432/db?sslmode=disable",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		JWTKey: "secret",
	}

	tests := []struct {
		name    string
		mutate  func(cfg ServerConfig) ServerConfig
		wantErr bool
	}{
		{
			name: "valid",
			mutate: func(cfg ServerConfig) ServerConfig {
				return cfg
			},
			wantErr: false,
		},
		{
			name: "empty host",
			mutate: func(cfg ServerConfig) ServerConfig {
				cfg.Host = ""
				return cfg
			},
			wantErr: true,
		},
		{
			name: "bad port",
			mutate: func(cfg ServerConfig) ServerConfig {
				cfg.Port = 70000
				return cfg
			},
			wantErr: true,
		},
		{
			name: "unsupported storage",
			mutate: func(cfg ServerConfig) ServerConfig {
				cfg.Storage.Type = "bad"
				return cfg
			},
			wantErr: true,
		},
		{
			name: "postgres without dsn",
			mutate: func(cfg ServerConfig) ServerConfig {
				cfg.Storage.Type = "postgres"
				cfg.Storage.DSN = ""
				return cfg
			},
			wantErr: true,
		},
		{
			name: "empty log level",
			mutate: func(cfg ServerConfig) ServerConfig {
				cfg.Log.Level = ""
				return cfg
			},
			wantErr: true,
		},
		{
			name: "bad log format",
			mutate: func(cfg ServerConfig) ServerConfig {
				cfg.Log.Format = "bad"
				return cfg
			},
			wantErr: true,
		},
		{
			name: "empty jwt key",
			mutate: func(cfg ServerConfig) ServerConfig {
				cfg.JWTKey = ""
				return cfg
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := tt.mutate(valid)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestClientConfigValidate(t *testing.T) {
	t.Parallel()

	valid := ClientConfig{
		ServerURL: "http://127.0.0.1:8080",
		DataDir:   ".gophkeeper",
		TLS:       false,
		TLSCA:     "",
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}

	tests := []struct {
		name    string
		mutate  func(cfg ClientConfig) ClientConfig
		wantErr bool
	}{
		{
			name: "valid",
			mutate: func(cfg ClientConfig) ClientConfig {
				return cfg
			},
			wantErr: false,
		},
		{
			name: "empty server url",
			mutate: func(cfg ClientConfig) ClientConfig {
				cfg.ServerURL = ""
				return cfg
			},
			wantErr: true,
		},
		{
			name: "empty data dir",
			mutate: func(cfg ClientConfig) ClientConfig {
				cfg.DataDir = ""
				return cfg
			},
			wantErr: true,
		},
		{
			name: "bad log format",
			mutate: func(cfg ClientConfig) ClientConfig {
				cfg.Log.Format = "bad"
				return cfg
			},
			wantErr: true,
		},
		{
			name: "empty log level",
			mutate: func(cfg ClientConfig) ClientConfig {
				cfg.Log.Level = ""
				return cfg
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := tt.mutate(valid)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
