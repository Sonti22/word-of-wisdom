# Word of Wisdom – Proof-of-Work TCP Service

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

TCP-сервер, защищённый от DDoS с помощью **Proof-of-Work (hashcash)**. Клиенты должны решить вычислительную задачу перед получением цитаты из "Word of Wisdom".

## 🎬 Демонстрация

```bash
$ docker-compose up --build

# Server output:
wow-server | {"level":"info","msg":"server started","addr":":8080","bits":20}
wow-server | {"level":"info","msg":"connection accepted","remote":"172.21.0.3:39120"}
wow-server | {"level":"info","msg":"solution verified","nonce":"1379033"}
wow-server | {"level":"info","msg":"quote sent"}

# Client output:
wow-client | {"level":"info","msg":"connecting to server","addr":"server:8080"}
wow-client | {"level":"info","msg":"received challenge","bits":20,"expires_in":60}
wow-client | {"level":"info","msg":"PoW solved","nonce":"1379033","duration_ms":297}
wow-client | 
wow-client | ╔════════════════════════════════════════════════════════════════════╗
wow-client | ║  Word of Wisdom                                                    ║
wow-client | ╠════════════════════════════════════════════════════════════════════╣
wow-client | ║                                                                    ║
wow-client | ║  The only true wisdom is in knowing you know nothing. – Socrates   ║
wow-client | ║                                                                    ║
wow-client | ╠════════════════════════════════════════════════════════════════════╣
wow-client | ║  PoW solved in: 297ms (bits: 20)                                   ║
wow-client | ╚════════════════════════════════════════════════════════════════════╝
```

**Время решения PoW:** ~300ms при bits=20 ⚡

---

## 🎯 Обоснование выбора PoW-алгоритма

### Почему Hashcash?

**Hashcash** (SHA-256 с ведущими нулевыми битами) выбран по следующим причинам:

1. **Простота реализации и верификации**
   - Проверка решения: `O(1)` — одно вычисление SHA-256
   - Нет сложных криптографических протоколов
   - Только стандартная библиотека Go (`crypto/sha256`)

2. **Настраиваемая сложность**
   - Динамическая регулировка `bits` (16–24) под нагрузку
   - bits=20 → ~1–5 сек на современном CPU
   - bits=24 → ~30–60 сек (защита от ботнетов)

3. **Детерминированность**
   - Результат зависит только от `challenge + nonce`
   - Легко воспроизвести в тестах

4. **Без внешних зависимостей**
   - Требование проекта: только stdlib
   - Hashcash реализуется на 100 строках кода

5. **Проверенный временем**
   - Используется в Bitcoin, Hashcash email stamps
   - Известные векторы атак давно закрыты

---

## 🛡️ Защита от DDoS и Replay-атак

### 1. **Anti-DDoS меры**

| Механизм | Реализация | Эффект |
|----------|------------|--------|
| **PoW Challenge** | SHA-256 с N ведущими нулями | Атакующий тратит CPU на каждый запрос |
| **Connection Deadline** | 30 сек на решение + отправку | Медленные атаки (slowloris) отсекаются |
| **Read/Write Timeouts** | 5 сек на чтение/запись | Защита от зависших соединений |
| **1 попытка на соединение** | После верификации → close | Нельзя переиспользовать challenge |
| **Graceful Shutdown** | SIGTERM/SIGINT → дождаться завершения | Корректное закрытие при перезагрузке |

### 2. **Anti-Replay меры**

| Параметр | Защита | Проверка на сервере |
|----------|--------|---------------------|
| **`salt`** | Уникальный рандом (16 байт) | Каждый challenge уникален |
| **`ts`** (timestamp) | Unix time генерации | `ts <= now <= ts + expires_in` |
| **`expires_in`** | TTL в секундах (60–300) | Старые решения отвергаются |
| **`resource`** | Идентификатор цели (`quote`) | Нельзя переиспользовать для других API |
| **`bits`** | Сложность встроена в challenge | Клиент не может снизить сложность |

**Пример challenge:**
```json
{
  "ver": "1",
  "alg": "sha256",
  "bits": 20,
  "ts": 1697462400,
  "expires_in": 60,
  "resource": "quote",
  "salt": "fd93f40dd926686fa2281b7e41fb8cdf"
}
```

**Невозможно:**
- ✅ Переиспользовать старый `nonce` (изменится `salt` и `ts`)
- ✅ Подделать сложность (встроена в challenge)
- ✅ Использовать решение для другого ресурса (проверка `resource`)
- ✅ Отправить решение через 10 минут (проверка `expires_in`)

### 3. **Rate Limiting (реализовано!)**

**Включено через ENV:**
```bash
WOW_RATE_LIMIT=10  # max 10 requests/sec per IP (0 = disabled)
```

**Механизм:** Token Bucket Algorithm
- Каждый IP получает свой "бакет" токенов
- Токены восполняются со скоростью `rate` в секунду
- Burst capacity = 2x rate (допускает всплески)
- Автоматическая очистка старых записей (каждые 5 минут)

**Эффект:**
- Атакующий не может отправить > N req/sec
- Легитимные всплески разрешены (burst)
- Memory-efficient (in-memory map, автоочистка)

### 4. **Adaptive Difficulty (реализовано!)**

**Включено через ENV:**
```bash
WOW_ADAPTIVE_BITS=true  # динамическая сложность (default: false)
```

**Механизм:** Difficulty увеличивается с количеством попыток
| Попытки | Сложность | Время решения |
|---------|-----------|---------------|
| 1-2     | bits      | ~1 сек        |
| 3-5     | bits+1    | ~2 сек        |
| 6-10    | bits+2    | ~4 сек        |
| 11-20   | bits+3    | ~8 сек        |
| 21+     | bits+4    | ~16 сек       |

**Эффект:**
- Повторные запросы от одного IP → экспоненциально дороже
- Ботнеты вынуждены тратить больше CPU
- Легитимные пользователи получают базовую сложность

**Логи при спаме:**
```json
{"level":"info","msg":"adaptive difficulty increased","base_bits":20,"new_bits":22,"attempts":7}
{"level":"warn","msg":"rate limited","remote":"192.168.1.1:1234","attempts":15}
```

### 5. **Production Extensions (опционально)**

Для дальнейшего усиления:
- **Connection pool limit**: max M одновременных соединений
- **Challenge cache**: хранить использованные `salt` в Redis (TTL = `expires_in`)
- **IP whitelist**: исключить trusted IPs из rate limit
- **DDoS mitigation**: интеграция с CloudFlare/AWS Shield

---

## 🏗️ Архитектура

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLIENT                                   │
│  ┌──────────────┐  ┌────────────────┐  ┌──────────────────┐    │
│  │   TCP Conn   │→ │  PoW Solver    │→ │  Quote Display   │    │
│  │   Handler    │  │  (8 goroutines)│  │  (Pretty Print)  │    │
│  └──────────────┘  └────────────────┘  └──────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                              ↕ TCP (JSON newline-delimited)
┌─────────────────────────────────────────────────────────────────┐
│                         SERVER                                   │
│  ┌──────────────┐  ┌────────────────┐  ┌──────────────────┐    │
│  │   TCP Conn   │→ │  PoW Verifier  │→ │  Quote Provider  │    │
│  │   Listener   │  │  (Challenge +  │  │  (crypto/rand)   │    │
│  │   (:8080)    │  │   Salt + TTL)  │  │                  │    │
│  └──────────────┘  └────────────────┘  └──────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

**Компоненты:**
- **Server** (`cmd/server`): слушает TCP, выдаёт challenge, проверяет PoW, возвращает цитату.
- **Client** (`cmd/client`): подключается, решает PoW, получает цитату.
- **PoW** (`internal/pow`): генерация/решение/верификация hashcash-challenge.
- **Quotes** (`internal/quotes`): пул мудрых цитат.
- **Server utils** (`internal/server`): протокол TCP, хелперы.

---

## 📡 Протокол взаимодействия

```
CLIENT                                    SERVER
  │                                          │
  │────────── 1. TCP Connect ───────────────>│
  │                                          │
  │<──────── 2. Challenge (JSON) ────────────│
  │          {ver, alg, bits, ts,            │
  │           expires_in, resource, salt}    │
  │                                          │
  │─── 3. Solve PoW (find nonce) ───        │
  │    (SHA-256 brute-force,                 │
  │     8 goroutines, ~300ms)                │
  │                                          │
  │──────── 4. Solution (JSON) ──────────────>│
  │          {nonce, digest_hex}             │
  │                                          │
  │                                   ─── 5. Verify PoW ───
  │                                       (LeadingZeroBits,
  │                                        timestamp, resource)
  │                                          │
  │<───────── 6. Quote (JSON) ───────────────│
  │          {quote: "Wisdom..."}            │
  │          OR {error: "..."}               │
  │                                          │
  │────────── 7. Close Connection ───────────X
```

**Детали протокола:**
1. **Client → Server**: открывает соединение.
2. **Server → Client**: отправляет JSON с `challenge` (ver, alg, bits, ts, expires_in, resource, salt).
3. **Client**: решает PoW (находит nonce, где `SHA-256(challenge:nonce)` имеет ≥ bits ведущих нулей).
4. **Client → Server**: отправляет JSON с `solution` (nonce, digest_hex).
5. **Server**: проверяет временное окно, resource, и количество ведущих нулей.
6. **Server → Client**: отправляет JSON с `quote` или `error`.
7. **Connection closed**: одна попытка на соединение.

## ⚙️ Конфигурация (ENV)

| Переменная           | Описание                                      | Default |
|----------------------|-----------------------------------------------|---------|
| `WOW_ADDR`           | Адрес сервера (host:port)                     | `:8080` |
| `WOW_BITS`           | Базовая сложность PoW (ведущие нули бит)     | `20`    |
| `WOW_EXPIRES`        | TTL challenge (сек)                           | `300`   |
| `WOW_RATE_LIMIT`     | Rate limit (req/sec per IP, 0=disabled)       | `0`     |
| `WOW_ADAPTIVE_BITS`  | Динамическая сложность (true/false)           | `false` |

**Пример с защитой:**
```bash
# Production setup with rate limiting + adaptive difficulty
export WOW_ADDR=:8080
export WOW_BITS=22
export WOW_EXPIRES=60
export WOW_RATE_LIMIT=10        # max 10 req/sec per IP
export WOW_ADAPTIVE_BITS=true   # increase difficulty on spam

./bin/server
```

---

## ⚡ Быстрый старт (5 минут)

### Вариант 1: Docker Compose (рекомендуется)

```bash
# Клонировать и запустить
git clone https://github.com/<your-username>/word-of-wisdom.git
cd word-of-wisdom
docker-compose up --build

# Результат: клиент получает цитату за ~300ms
```

**Готово!** Сервер запустится на `:8080`, клиент подключится, решит PoW и выведет цитату.

---

### Вариант 2: Native Go (без Docker)

**Требования:** Go 1.22+

```bash
# 1. Клонировать и установить зависимости (нет внешних deps)
git clone https://github.com/<your-username>/word-of-wisdom.git
cd word-of-wisdom

# 2. Собрать бинарники
go build -o bin/server ./cmd/server
go build -o bin/client ./cmd/client

# 3. Запустить сервер (терминал 1)
export WOW_ADDR=:8080
export WOW_BITS=20
./bin/server

# 4. Запустить клиент (терминал 2)
export WOW_ADDR=127.0.0.1:8080
./bin/client
```

---

### Вариант 3: Makefile / PowerShell

#### Linux/macOS (Makefile):
```bash
# Показать все команды
make help

# Полный цикл: форматирование, линтинг, тесты, сборка
make all

# Только сборка
make build

# Запуск сервера
make run-server

# Запуск клиента (в другом терминале)
make run-client

# Тесты
make test
make test-short         # Только unit-тесты (быстро)
make test-integration   # Только интеграционные

# Качество кода
make fmt                # Форматирование
make lint               # Линтинг
make check              # fmt + lint + test

# Docker
make up                 # Запуск через docker-compose
make down               # Остановка
make logs               # Логи
```

### Windows (PowerShell):
```powershell
# Обновить PATH (если Go не найден)
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")

# Показать все команды
.\make.ps1 help

# Сборка
.\make.ps1 build

# Запуск
.\make.ps1 run-server
.\make.ps1 run-client

# Тесты
.\make.ps1 test

# Docker
.\make.ps1 up
.\make.ps1 down
```

> **Note для Windows:** Если PowerShell блокирует скрипты, используйте:
> ```powershell
> Set-ExecutionPolicy -ExecutionPolicy Bypass -Scope Process
> ```
> Или запускайте команды Go напрямую: `go build`, `go test`, `docker-compose up`.

---

## 🐳 Docker

### Quick Start

```bash
# Запуск server + client через docker-compose
docker-compose up --build
```

### Раздельная сборка

```bash
# Собрать образ сервера
docker build -f Dockerfile.server -t wow-server:latest .

# Собрать образ клиента
docker build -f Dockerfile.client -t wow-client:latest .

# Запустить сервер
docker run -d \
  --name wow-server \
  -p 8080:8080 \
  -e WOW_BITS=20 \
  wow-server:latest

# Запустить клиент
docker run --rm \
  --name wow-client \
  --link wow-server:server \
  -e WOW_ADDR=server:8080 \
  wow-client:latest
```

### Кастомная конфигурация

```bash
# Запуск с низкой сложностью (быстрые тесты)
WOW_BITS=16 docker-compose up --build

# Запуск с высокой сложностью (production)
WOW_BITS=24 docker-compose up --build

# Запуск с rate limiting + adaptive difficulty
WOW_RATE_LIMIT=10 WOW_ADAPTIVE_BITS=true docker-compose up --build
```

---

## 🧪 Тестирование

### Unit-тесты

```bash
# Все тесты с race detection
go test -race -v ./...

# Только unit (быстро)
go test -short ./...

# С покрытием
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Статистика тестов:**
- `internal/pow`: 10 unit-тестов + 2 бенчмарка
- `internal/quotes`: 9 unit-тестов (concurrency, distribution)
- `internal/ratelimit`: 9 unit-тестов + 2 бенчмарка (token bucket, adaptive)
- `tests/integration_test.go`: 5 e2e-тестов (success/error/timeout)

### Интеграционные тесты

```bash
# Linux/macOS
./scripts/run_local.sh

# Windows
.\scripts\run_local.ps1

# Негативные сценарии (garbage data, invalid nonce, timeout)
./scripts/test_negative.sh       # Linux/macOS
.\scripts\test_negative.ps1      # Windows

# Rate limit тесты (rapid requests → rate limit + adaptive bits)
./scripts/test_ratelimit.sh      # Linux/macOS
.\scripts\test_ratelimit.ps1     # Windows
```

**Ожидаемые результаты rate-limit теста:**
```json
{"level":"info","msg":"connection accepted","attempts":1}
{"level":"info","msg":"connection accepted","attempts":2}
...
{"level":"info","msg":"adaptive difficulty increased","base_bits":18,"new_bits":20,"attempts":6}
{"level":"warn","msg":"rate limited","attempts":11}
```

### Race Detection

```bash
# Проверка на race conditions
go test -race ./...

# Запуск сервера с race detector
go run -race ./cmd/server
```

---

## 📊 Метрики производительности

| Параметр | Значение |
|----------|----------|
| **Время решения (bits=20)** | ~300ms (8 goroutines) |
| **Время решения (bits=24)** | ~30–60 сек |
| **Проверка решения** | <1ms (одно SHA-256) |
| **Размер challenge** | ~200 байт (JSON) |
| **Размер solution** | ~150 байт (JSON) |
| **Размер quote** | ~100–200 байт (JSON) |
| **Memory (server)** | ~5 МБ (RSS) |
| **Docker image (server)** | ~7 МБ (scratch) |
| **Docker image (client)** | ~7 МБ (alpine) |

---

## 📝 Структура проекта

```
word-of-wisdom/
├── cmd/
│   ├── server/main.go          # TCP server entry point
│   └── client/main.go          # CLI client entry point
├── internal/
│   ├── pow/
│   │   ├── pow.go              # Hashcash: Generate/Solve/Verify
│   │   └── pow_test.go         # 10 unit tests + benchmarks
│   ├── quotes/
│   │   ├── quotes.go           # 10 wisdom quotes
│   │   └── quotes_test.go      # 9 tests (concurrency, distribution)
│   ├── ratelimit/
│   │   ├── ratelimit.go        # Token bucket + adaptive difficulty
│   │   └── ratelimit_test.go   # 9 unit tests + benchmarks
│   └── server/
│       └── tcp.go              # Protocol handlers + JSON helpers
├── tests/
│   └── integration_test.go     # 5 e2e tests (success/error/timeout)
├── scripts/
│   ├── run_local.sh/.ps1       # Local runner (bash + PowerShell)
│   ├── test_negative.sh/.ps1   # Negative scenarios
│   └── test_ratelimit.sh/.ps1  # Rate limit + adaptive difficulty tests
├── Dockerfile.server           # Multi-stage (scratch)
├── Dockerfile.client           # Multi-stage (scratch)
├── Dockerfile.client.alpine    # Multi-stage (alpine for compose)
├── docker-compose.yml          # Orchestration (server + client)
├── Makefile                    # 20+ targets (build/test/docker/ci)
├── make.ps1                    # PowerShell equivalent
├── go.mod                      # Go 1.22, no external deps
├── .gitignore                  # Go + Docker + IDE
├── .dockerignore               # Build optimization
└── README.md                   # This file
```

---

## 🔧 Требования

- **Go:** 1.22+
- **Docker:** 20.10+ (опционально)
- **Платформы:** Linux, macOS, Windows

**Зависимости:** только стандартная библиотека Go (no external deps)

---

## 📚 Дополнительная информация

### Настройка сложности

| Bits | Среднее время | Use Case |
|------|---------------|----------|
| 16   | ~10–50ms      | Локальные тесты |
| 20   | ~1–5 сек      | Development |
| 22   | ~5–15 сек     | Production (умеренная нагрузка) |
| 24   | ~30–60 сек    | Production (высокая защита) |

### Troubleshooting

**Проблема:** `go: command not found` (Windows)
```powershell
# Решение: обновить PATH
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
```

**Проблема:** `bind: address already in use`
```bash
# Linux/macOS
lsof -ti:8080 | xargs kill -9

# Windows
netstat -ano | findstr :8080
taskkill /PID <PID> /F
```

**Проблема:** PowerShell блокирует скрипты
```powershell
Set-ExecutionPolicy -ExecutionPolicy Bypass -Scope Process
```

---

## 🤝 Contributing

Pull requests welcome! Убедитесь, что:
- Все тесты проходят: `go test -race ./...`
- Код отформатирован: `gofmt -w .`
- Нет race conditions: `go test -race ./...`

---

## 📄 Лицензия

MIT

