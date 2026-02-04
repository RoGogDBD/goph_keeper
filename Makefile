SHELL := /bin/sh

APP_SERVER := gophkeeper-server
APP_CLIENT := gophkeeper-client

BUILD_DIR := bin

VERSION ?= dev
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)

LDFLAGS := -ldflags "-X goph_keeper/internal/version.BuildVersion=$(VERSION) -X goph_keeper/internal/version.BuildDate=$(DATE) -X goph_keeper/internal/version.BuildCommit=$(COMMIT)"

.PHONY: help build build-server build-client run run-server run-client clean clean-data test vet lint docker-up docker-down security coverage

help: ## Показать доступные цели make
	@awk 'BEGIN {FS=":.*##"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: build-server build-client ## Собрать сервер и клиент

build-server: ## Собрать бинарник сервера
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_SERVER) ./cmd/server

build-client: ## Собрать бинарник клиента
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_CLIENT) ./cmd/client

run: docker-up ## Запустить полный стек (db + server + nginx) через docker compose

run-server: ## Запустить сервер без сборки
	go run $(LDFLAGS) ./cmd/server

run-client: ## Запустить клиент без сборки
	go run $(LDFLAGS) ./cmd/client

clean: ## Удалить собранные бинарники
	@rm -rf $(BUILD_DIR)

clean-data: ## Удалить локальные данные клиента
	@rm -rf .gophkeeper

test: ## Запустить тесты
	go test ./...

vet: ## Запустить go vet
	go vet ./...

lint: ## Запустить базовые линтеры (golangci-lint)
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run ./...

security: ## Запустить govulncheck
	@command -v govulncheck >/dev/null 2>&1 || { echo "govulncheck not found. Install: go install golang.org/x/vuln/cmd/govulncheck@latest"; exit 1; }
	govulncheck ./...

coverage: ## Запустить тесты с покрытием
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

docker-up: ## Запустить docker compose стек
	docker compose up --build -d

docker-down: ## Остановить docker compose стек
	docker compose down -v
