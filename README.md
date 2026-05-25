# Task Manager (Concurrent Task Engine)

RESTful API для управления задачами с асинхронной обработкой, конкурентными воркерами и Docker контейнеризацией.

## 📋 Оглавление

- [Возможности](#-возможности)
- [Технологии](#-технологии)
- [Быстрый старт](#-быстрый-старт)
- [API Эндпоинты](#-api-эндпоинты)
- [Архитектура](#-архитектура)
- [Конкурентность](#-конкурентность)
- [Docker](#-docker)
- [Тестирование](#-тестирование)
- [Структура проекта](#-структура-проекта)

## 🚀 Возможности

- ✅ **JWT аутентификация** — безопасный доступ к API
- ✅ **CRUD операции** — полное управление задачами
- ✅ **Worker pool** — параллельная обработка задач (5 воркеров) на основе buffered channels
- ✅ **Batch processing** — группировка операций для оптимизации записи в БД
- ✅ **Deadline checker** — фоновый мониторинг просроченных задач
- ✅ **Graceful shutdown** — корректное завершение всех горутин через context.WithCancel и sync.WaitGroup
- ✅ **Потокобезопасность** — использование sync.RWMutex для доступа к общему состоянию (подтверждено тестами с флагом -race)
- ✅ **Метрики воркеров** — отслеживание выполненных/неудачных задач
- ✅ **Docker support** — оптимизированный multi-stage образ

## 🛠 Технологии

| Компонент | Технология |
|-----------|------------|
| Язык | Go 1.21 |
| База данных | PostgreSQL 15 |
| Аутентификация | JWT |
| Конфигурация | Viper |
| Логирование | slog |
| Миграции | golang-migrate |
| Контейнеризация | Docker + Docker Compose |
| Тестирование | go test с флагом -race |

## 🚀 Быстрый старт

### Требования

- Go 1.21+
- Docker и Docker Compose
- PostgreSQL 15 (или Docker)

### Запуск через Docker (рекомендуется)

```bash
# 1. Клонируйте репозиторий
git clone https://github.com/yourusername/task-manager.git
cd task-manager

# 2. Запустите все сервисы
docker-compose up -d

# 3. Проверьте что всё работает
curl http://localhost:8080/health
```

### Локальный запуск

```bash
# 1. Установите зависимости
go mod download

# 2. Настройте переменные окружения
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=taskmanager
export JWT_SECRET=your-secret-key

# 3. Запустите миграции
go run cmd/migrate/main.go up

# 4. Запустите сервер
go run cmd/api/main.go
```

## 📡 API Эндпоинты

### Аутентификация

| Метод | Endpoint | Описание |
|-------|----------|----------|
| POST | `/api/auth/register` | Регистрация пользователя |
| POST | `/api/auth/login` | Вход (возвращает JWT токен) |

### Задачи

| Метод | Endpoint | Описание | Auth |
|-------|----------|----------|------|
| GET | `/api/tasks` | Список задач | ✅ |
| POST | `/api/tasks` | Создать задачу | ✅ |
| GET | `/api/tasks/:id` | Получить задачу | ✅ |
| PUT | `/api/tasks/:id` | Обновить задачу | ✅ |
| DELETE | `/api/tasks/:id` | Удалить задачу | ✅ |
| POST | `/api/tasks/batch` | Пакетное создание | ✅ |
| POST | `/api/tasks/export` | Экспорт задач | ✅ |

### Системные

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/health` | Проверка здоровья сервиса |
| GET | `/api/metrics` | Метрики worker pool |

## 🏗 Архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                         HTTP Server                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   Auth      │  │    Task     │  │   Batch Handler     │  │
│  │   Handler   │  │   Handler   │  │                     │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
│         │                │                     │             │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────────▼──────────┐  │
│  │   Auth      │  │    Task     │  │  Notification Svc   │  │
│  │   Service   │  │   Service   │  │                     │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
│         │                │                     │             │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────────▼──────────┐  │
│  │    User     │  │    Task     │  │   Worker Pool       │  │
│  │ Repository  │  │ Repository  │  │  (5 workers)        │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
│         │                │                     │             │
└─────────┼────────────────┼─────────────────────┼────────────┘
          │                │                     │
          ▼                ▼                     ▼
    ┌─────────────────────────────────────────────────────┐
    │                    PostgreSQL                        │
    └─────────────────────────────────────────────────────┘
```

### Компоненты

1. **TaskWorkerPool** — пул из 5 воркеров для обработки тяжелых задач (экспорт, анализ, пакетное обновление)
   - Buffered channels для контроля нагрузки (буфер = workers × 20)
   - Graceful shutdown через context.WithCancel
   - Сбор метрик выполнения

2. **BatchProcessor** — буферизация задач для пакетной записи в БД
   - Размер батча: 100 задач
   - Буфер: 1000 задач
   - Авто-сброс каждые 5 секунд

3. **DeadlineChecker** — фоновая проверка просроченных задач
   - Интервал проверки: 1 минута
   - Отправка уведомлений пользователям

## ⚡ Конкурентность

### Worker Pool на buffered channels

```go
// Создание пула с буфером workers*20
jobQueue := make(chan queue.Job, workers*20)
resultQueue := make(chan queue.Result, workers*20)
```

### Потокобезопасность через sync.RWMutex

```go
// Защита метрик
metrics.mu.RLock()
defer metrics.mu.RUnlock()
```

### Graceful Shutdown

```go
// Использование context для отмены
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Ожидание завершения горутин
var wg sync.WaitGroup
wg.Wait()
```

## 🐳 Docker

### Multi-stage сборка

```dockerfile
# Stage 1: Build
FROM golang:1.21-alpine AS builder
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/api

# Stage 2: Runtime
FROM alpine:3.19
COPY --from=builder /app/main .
USER appuser
CMD ["./main"]
```

### Запуск с Docker Compose

```bash
docker-compose up -d
```

Сервисы:
- `api` — основное приложение (порт 8080)
- `postgres` — база данных (порт 5432)

## 🧪 Тестирование

### Запуск всех тестов

```bash
go test ./... -v
```

### Запуск с проверкой race conditions

```bash
go test ./... -race -v
```

### Покрытие кода

```bash
go test ./... -cover
```

Текущее покрытие ядра менеджера (internal/worker): **>80%**

### Edge cases покрытые тестами

- ✅ Отмена задач через context
- ✅ Таймауты операций
- ✅ Переполнение очереди (buffer full)
- ✅ Concurrent access к метрикам
- ✅ Graceful shutdown

## 📁 Структура проекта

```
task-manager/
├── cmd/
│   └── api/
│       └── main.go              # Точка входа
├── internal/
│   ├── config/                  # Конфигурация приложения
│   ├── handler/                 # HTTP handlers
│   │   └── middleware/          # JWT middleware
│   ├── models/                  # Модели данных
│   ├── queue/                   # Типы для очередей
│   ├── repository/              # Работа с БД
│   ├── service/                 # Бизнес-логика
│   └── worker/                  # Воркеры и процессоры
│       ├── task_worker_pool.go  # Пул воркеров
│       ├── batch_processor.go   # Пакетная обработка
│       └── deadline_checker.go  # Проверка дедлайнов
├── pkg/
│   ├── jwt/                     # JWT утилиты
│   └── logger/                  # Логгер
├── migrations/                  # SQL миграции
├── docker-compose.yml           # Docker Compose конфигурация
├── Dockerfile                   # Multi-stage Dockerfile
├── go.mod                       # Go модуль
└── README.md                    # Документация
```

## 🔧 Конфигурация

Переменные окружения:

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `DB_HOST` | Хост PostgreSQL | `localhost` |
| `DB_PORT` | Порт PostgreSQL | `5432` |
| `DB_USER` | Пользователь БД | `postgres` |
| `DB_PASSWORD` | Пароль БД | `postgres` |
| `DB_NAME` | Имя базы данных | `taskmanager` |
| `DB_SSLMODE` | SSL режим | `disable` |
| `JWT_SECRET` | Секретный ключ JWT | `secret` |
| `JWT_EXPIRATION` | Время жизни токена (часы) | `24` |
| `SERVER_PORT` | Порт сервера | `8080` |

## 📝 Лицензия

MIT License
