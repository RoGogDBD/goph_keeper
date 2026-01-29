# goph_keeper
GophKeeper представляет собой клиент-серверную систему, позволяющую пользователю надёжно и безопасно хранить логины, пароли, бинарные данные и прочую приватную информацию.

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
