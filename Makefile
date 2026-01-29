SHELL := /bin/sh

APP_SERVER := gophkeeper-server
APP_CLIENT := gophkeeper-client

BUILD_DIR := bin

VERSION ?= dev
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)

LDFLAGS := -ldflags "-X goph_keeper/internal/version.BuildVersion=$(VERSION) -X goph_keeper/internal/version.BuildDate=$(DATE) -X goph_keeper/internal/version.BuildCommit=$(COMMIT)"

.PHONY: build build-server build-client run-server run-client clean test vet lint docker-up docker-down

build: build-server build-client

build-server:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_SERVER) ./cmd/server

build-client:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_CLIENT) ./cmd/client

run-server:
	go run $(LDFLAGS) ./cmd/server

run-client:
	go run $(LDFLAGS) ./cmd/client

clean:
	@rm -rf $(BUILD_DIR)

test:
	go test ./...

vet:
	go vet ./...

lint:
	@echo "No linter configured"

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v
