# goph_keeper
GophKeeper — клиент-серверная система для безопасного хранения приватных данных пользователя: логинов/паролей, текстовых заметок, бинарных данных и банковских карт. Клиент умеет работать как через CLI, так и через TUI (команда `ui`). Сервер хранит данные в Postgres.

## Быстрый запуск (рекомендуется)
Полный стек через Docker (Postgres + сервер + nginx):

```bash
make run
```

Клиент (в отдельном терминале):

```bash
make run-client -- --config ./client.yaml ui
```

Пример `client.yaml`:

```yaml
server_url: "http://127.0.0.1:8080"
data_dir: ".gophkeeper"
tls: false
tls_ca: ""
log:
  level: "info"
  format: "text"
```

## Локальный запуск без Docker
Нужен доступный Postgres. Затем:

```bash
export GOPHKEEPER_SERVER_STORAGE=postgres
export GOPHKEEPER_SERVER_DSN="postgresql://user:pass@127.0.0.1:5432/gophkeeper?sslmode=disable"
export GOPHKEEPER_SERVER_JWT_KEY="change-me"
make run-server
```

Клиент:

```bash
make run-client -- --server-url http://127.0.0.1:8080 ui
```

## Запуск и конфигурация
Приоритет параметров: флаги CLI → переменные окружения → конфиг‑файл → значения по умолчанию.

### Сервер
Пример YAML:

```yaml
host: "0.0.0.0"
port: 8080
storage:
  type: "memory"
  dsn: ""
log:
  level: "info"
  format: "text"
jwt_key: "change-me"
tls:
  enabled: false
  cert_file: ""
  key_file: ""
```

Запуск:

```bash
./server --config ./server.yaml
```

Переменные окружения:
- `GOPHKEEPER_SERVER_HOST`
- `GOPHKEEPER_SERVER_PORT`
- `GOPHKEEPER_SERVER_STORAGE`
- `GOPHKEEPER_SERVER_DSN`
- `GOPHKEEPER_SERVER_JWT_KEY`
- `GOPHKEEPER_SERVER_TLS`
- `GOPHKEEPER_SERVER_TLS_CERT`
- `GOPHKEEPER_SERVER_TLS_KEY`
- `GOPHKEEPER_LOG_LEVEL`
- `GOPHKEEPER_LOG_FORMAT`

### Клиент
Пример YAML:

```yaml
server_url: "http://127.0.0.1:8080"
data_dir: ".gophkeeper"
tls: false
tls_ca: ""
log:
  level: "info"
  format: "text"
```

Запуск:

```bash
./client --config ./client.yaml
./client --version
./client version
```

Переменные окружения:
- `GOPHKEEPER_CLIENT_SERVER_URL`
- `GOPHKEEPER_CLIENT_DATA_DIR`
- `GOPHKEEPER_CLIENT_TLS`
- `GOPHKEEPER_CLIENT_TLS_CA`
- `GOPHKEEPER_LOG_LEVEL`
- `GOPHKEEPER_LOG_FORMAT`

## TLS
Чтобы включить HTTPS на сервере:

```bash
export GOPHKEEPER_SERVER_TLS=true
export GOPHKEEPER_SERVER_TLS_CERT=/path/to/cert.pem
export GOPHKEEPER_SERVER_TLS_KEY=/path/to/key.pem
make run-server
```

Клиент с TLS и CA:

```bash
make run-client -- --tls true --server-url https://127.0.0.1:8080 --tls-ca /path/to/ca.pem ui
```

## Swagger / OpenAPI
Спецификация и UI:

```bash
# OpenAPI YAML
curl http://127.0.0.1:8080/swagger/openapi.yaml

# Swagger UI
open http://127.0.0.1:8080/swagger/
```

## Сборка и запуск

Сборка бинарников:

```bash
make build
```

Сборка отдельно:

```bash
make build-server
make build-client
```

Запуск без сборки:

```bash
make run-server
make run-client
```

## Security

Проверка уязвимостей через govulncheck:

```bash
make security
```
