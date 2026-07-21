# GophKeeper

Клиент-серверный менеджер паролей. Позволяет безопасно хранить логины/пароли, текстовые заметки, данные банковских карт и произвольные бинарные файлы с синхронизацией между устройствами.

---

## Содержание

- [Требования](#требования)
- [Быстрый старт через Docker](#быстрый-старт-через-docker)
- [Запуск сервера вручную](#запуск-сервера-вручную)
- [Сборка клиента](#сборка-клиента)
- [Использование CLI](#использование-cli)
- [Архитектура](#архитектура)
- [Переменные окружения сервера](#переменные-окружения-сервера)
- [Разработка и тесты](#разработка-и-тесты)

---

## Требования

| Инструмент | Версия |
| --- | --- |
| Go | ≥ 1.25 |
| PostgreSQL | ≥ 15 |
| Docker + Docker Compose | любая актуальная |
| make | опционально |

---

## Быстрый старт через Docker

Самый простой способ запустить сервер вместе с базой данных:

```bash
# 1. Клонировать репозиторий
git clone https://github.com/timofeevav/gophkeeper.git
cd gophkeeper

# 2. Поднять PostgreSQL + сервер
docker compose up --build -d

# 3. Убедиться что сервер запустился (-k — сервер использует
#    самоподписанный сертификат, если TLS_CERT_FILE/TLS_KEY_FILE не заданы)
curl -k https://localhost:8080/api/v1/auth/login \
  -d '{"login":"test","password":"test"}' \
  -H "Content-Type: application/json"
# ответ: {"error":"invalid credentials"} — значит сервер работает
```

> Сервер всегда работает по HTTPS: если пара сертификат/ключ не задана,
> при старте генерируется самоподписанный сертификат. Для клиента в этом
> случае нужен флаг `--insecure`.

Остановить:

```bash
docker compose down
```

> **Важно:** JWT_SECRET в `docker-compose.yml` задан в виде примера. Перед продакшн-развёртыванием обязательно замените его на случайную строку длиной ≥ 32 символов.

---

## Запуск сервера вручную

### 1. Подготовить PostgreSQL

```bash
psql -U postgres -c "CREATE USER gophkeeper WITH PASSWORD 'gophkeeper';"
psql -U postgres -c "CREATE DATABASE gophkeeper OWNER gophkeeper;"
```

### 2. Применить миграции

```bash
# Установить утилиту migrate (если ещё не установлена)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Применить миграции
migrate -path migrations \
        -database "postgres://gophkeeper:gophkeeper@localhost:5432/gophkeeper?sslmode=disable" \
        up
```

### 3. Создать файл конфигурации

Скопируйте `.env.example` в `.env` и заполните значения:

```bash
cp .env.example .env
```

```env
SERVER_ADDRESS=:8080
DATABASE_URI=postgres://gophkeeper:gophkeeper@localhost:5432/gophkeeper?sslmode=disable
JWT_SECRET=замените-на-случайную-строку-длиной-32-символа
LOG_LEVEL=info
```

### 4. Собрать и запустить сервер

```bash
go build -o bin/server ./cmd/server
./bin/server
```

Или через Make:

```bash
make build-server
./bin/server
```

---

## Сборка клиента

### Для текущей платформы

```bash
make build-client
# бинарник: bin/gophkeeper (или bin/gophkeeper.exe на Windows)
```

### Для всех платформ сразу

```bash
make build-client-all
```

Результат в папке `bin/`:

| Файл | Платформа |
| --- | --- |
| `gophkeeper-linux-amd64` | Linux x86-64 |
| `gophkeeper-windows-amd64.exe` | Windows x86-64 |
| `gophkeeper-darwin-amd64` | macOS Intel |
| `gophkeeper-darwin-arm64` | macOS Apple Silicon |

### Версия в бинарнике

При сборке через `make build-client` в бинарник автоматически встраиваются версия, дата сборки и хеш коммита:

```bash
./bin/gophkeeper version
# Version:    1.2.0
# Build Date: 2026-06-29T10:00:00Z
# Commit:     abc1234
```

---

## Использование CLI

### Глобальные флаги

```
--server   string   Адрес сервера (по умолчанию: https://localhost:8080)
--insecure          Отключить проверку TLS-сертификата (для самоподписанных сертификатов)
--config   string   Путь к файлу конфигурации клиента
```

Конфигурация сохраняется в `~/.gophkeeper/config.yaml` после первого входа.

---

### Регистрация и вход

```bash
# Зарегистрировать нового пользователя (логин — флагом или интерактивно,
# пароль — всегда интерактивно, чтобы не оставался в истории шелла)
./bin/gophkeeper register --insecure --login myuser

# Войти в существующий аккаунт
./bin/gophkeeper login --insecure --login myuser

# Завершить сессию (удалить токен)
./bin/gophkeeper logout
```

После успешного входа токен сохраняется локально — последующие команды не требуют повторной аутентификации.

---

### Работа с секретами

#### Список всех секретов

```bash
./bin/gophkeeper secret list

# С фильтром по типу
./bin/gophkeeper secret list --type login_password
./bin/gophkeeper secret list --type card
./bin/gophkeeper secret list --type text
./bin/gophkeeper secret list --type binary
```

Пример вывода:
```
ID                                    Тип               Название
------------------------------------------------------------------------
7c9e6679-7425-40de-944b-e07fc1f90ae7  login_password    GitHub
3a1b2c3d-1234-5678-abcd-ef0123456789  card              Sberbank Visa
```

#### Добавить секрет

```bash
./bin/gophkeeper secret add <тип>
```

Поддерживаемые типы:

**`login_password`** — логин и пароль:
```bash
./bin/gophkeeper secret add login_password
# Название: GitHub
# Метаданные (опционально): рабочий аккаунт
# Логин: mylogin
# Пароль: ••••••••
# Мастер-пароль: ••••••••
# Секрет создан: 7c9e6679-...
```

**`text`** — произвольный текст (заметка, одноразовый код и т.д.):
```bash
./bin/gophkeeper secret add text
# Название: Wi-Fi пароль
# Метаданные (опционально): домашняя сеть
# Текст: SuperSecretWifi123
# Мастер-пароль: ••••••••
```

**`card`** — данные банковской карты:
```bash
./bin/gophkeeper secret add card
# Название: Sberbank Visa
# Метаданные (опционально): зарплатная карта
# Номер карты: 4276 1234 5678 9012
# Держатель: IVAN IVANOV
# Срок (MM/YY): 12/27
# CVV: •••
# Мастер-пароль: ••••••••
```

**`binary`** — произвольный файл:
```bash
./bin/gophkeeper secret add binary
# Название: SSH ключ
# Метаданные (опционально): сервер prod
# Путь к файлу: /home/user/.ssh/id_rsa
# Мастер-пароль: ••••••••
```

#### Получить секрет по ID

```bash
./bin/gophkeeper secret get 7c9e6679-7425-40de-944b-e07fc1f90ae7
# Мастер-пароль: ••••••••
# ID:       7c9e6679-7425-40de-944b-e07fc1f90ae7
# Тип:      login_password
# Название: GitHub
# Мета:     рабочий аккаунт
# Данные:   {"login":"mylogin","password":"mypassword"}
```

#### Удалить секрет

```bash
./bin/gophkeeper secret delete 7c9e6679-7425-40de-944b-e07fc1f90ae7
# Секрет 7c9e6679-... удалён
```

---

### Синхронизация

Синхронизация происходит автоматически при каждом обращении к серверу. Принудительная синхронизация:

```bash
./bin/gophkeeper sync
# Синхронизация завершена: обновлено 3, конфликтов 0
```

Если обнаружены конфликты версий (одни и те же данные изменились и на сервере, и локально), команда сообщит об этом:

```
Синхронизация завершена: обновлено 1, конфликтов 2
Конфликты обнаружены. Используйте 'secret get <id>' для просмотра серверной версии.
```

---

## Мастер-пароль

GophKeeper использует **сквозное шифрование (E2E)**:

- Данные шифруются **на клиенте** до отправки на сервер алгоритмом **AES-256-GCM**
- Ключ шифрования выводится из мастер-пароля через **PBKDF2** (100 000 итераций, SHA-256)
- Сервер хранит только зашифрованный blob — даже администратор сервера **не может** прочитать ваши данные
- Мастер-пароль **нигде не хранится** и не передаётся на сервер

> Если вы забудете мастер-пароль, восстановить данные будет невозможно.

---

## Архитектура

```
┌─────────────────┐   HTTPS/JSON   ┌──────────────────────┐
│   CLI Client    │◄──────────────►│   HTTP Server        │
│                 │                │                      │
│  ┌───────────┐  │                │  /api/v1/auth        │
│  │  AES-256  │  │                │  /api/v1/secrets     │
│  │   GCM     │  │                │  /api/v1/sync        │
│  └───────────┘  │                │                      │
│  ┌───────────┐  │                │  ┌────────────────┐  │
│  │  bbolt DB │  │                │  │  PostgreSQL    │  │
│  │  (cache)  │  │                │  └────────────────┘  │
│  └───────────┘  │                └──────────────────────┘
└─────────────────┘
```

### Серверные слои

```
handler/     — HTTP-обработчики, валидация входных данных
service/     — бизнес-логика (Auth, Secret, Sync)
repository/  — работа с PostgreSQL через pgx/v5
middleware/  — JWT-авторизация, логирование
crypto/      — bcrypt (пароли), JWT HS256
```

### API эндпоинты

| Метод | Путь | Описание | Авторизация |
| --- | --- | --- | --- |
| POST | `/api/v1/auth/register` | Регистрация | — |
| POST | `/api/v1/auth/login` | Вход | — |
| POST | `/api/v1/secrets` | Создать секрет | JWT |
| GET | `/api/v1/secrets` | Список секретов | JWT |
| GET | `/api/v1/secrets/{id}` | Получить секрет | JWT |
| PUT | `/api/v1/secrets/{id}` | Обновить секрет | JWT |
| DELETE | `/api/v1/secrets/{id}` | Удалить секрет | JWT |
| POST | `/api/v1/sync` | Синхронизация | JWT |

---

## Переменные окружения сервера

| Переменная | Обязательная | По умолчанию | Описание |
| --- | :---: | --- | --- |
| `DATABASE_URI` | ✅ | — | DSN для подключения к PostgreSQL |
| `JWT_SECRET` | ✅ | — | Секрет для подписи JWT (≥ 32 символа) |
| `SERVER_ADDRESS` | — | `:8080` | Адрес и порт сервера |
| `TLS_CERT_FILE` | — | — | Путь к TLS-сертификату |
| `TLS_KEY_FILE` | — | — | Путь к TLS-ключу |
| `LOG_LEVEL` | — | `info` | Уровень логирования: debug/info/warn/error |

### TLS

Сервер работает **только по HTTPS**. Если `TLS_CERT_FILE`/`TLS_KEY_FILE`
не заданы, при старте автоматически генерируется самоподписанный сертификат —
в этом случае запускайте клиент с флагом `--insecure`.

Для продакшна задайте собственную пару сертификат/ключ:

```env
TLS_CERT_FILE=/etc/ssl/certs/server.crt
TLS_KEY_FILE=/etc/ssl/private/server.key
```

---

## Разработка и тесты

### Запуск тестов

```bash
# Все тесты
make test

# С отчётом о покрытии
make test-coverage
# открыть coverage.html в браузере для просмотра детального отчёта
```

### Покрытие по пакетам

| Пакет | Покрытие |
| --- | --- |
| `server/middleware` | 100% |
| `server/service` | ~85% |
| `client/storage` | ~87% |
| `server/crypto` | ~81% |
| `client/crypto` | ~82% |
| `server/handler` | ~71% |

### Линтер

```bash
# Установить golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Запустить
make lint
```

### Команды Make

| Команда | Описание |
| --- | --- |
| `make build` | Собрать сервер и клиент |
| `make build-server` | Собрать только сервер |
| `make build-client` | Собрать клиент для текущей платформы |
| `make build-client-all` | Собрать клиент для всех платформ |
| `make test` | Запустить тесты с флагом `-race` |
| `make test-coverage` | Тесты + HTML-отчёт о покрытии |
| `make lint` | Запустить линтер |
| `make docker-build` | Собрать Docker-образ сервера |
| `make migrate-up` | Применить миграции БД |
| `make migrate-down` | Откатить миграции БД |
| `make clean` | Удалить артефакты сборки |

### Структура проекта

```
GophKeeper/
├── cmd/
│   ├── server/main.go          # точка входа сервера
│   └── client/main.go          # точка входа CLI
├── internal/
│   ├── server/
│   │   ├── config/             # конфигурация сервера
│   │   ├── crypto/             # bcrypt + JWT
│   │   ├── handler/            # HTTP-обработчики
│   │   ├── middleware/         # auth, logger
│   │   ├── model/              # доменные модели
│   │   ├── repository/         # интерфейсы репозиториев
│   │   │   └── postgres/       # реализация на PostgreSQL
│   │   └── service/            # бизнес-логика
│   └── client/
│       ├── api/                # HTTP-клиент
│       ├── cmd/                # Cobra-команды
│       ├── config/             # конфигурация клиента
│       ├── crypto/             # AES-256-GCM + PBKDF2
│       ├── storage/            # локальный кэш (bbolt)
│       ├── sync/               # синхронизация
│       └── version/            # версия и дата сборки
├── migrations/                 # SQL-миграции
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── .env.example
```