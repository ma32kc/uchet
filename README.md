# Uchet — учёт семейных расходов

Сервис из тестового задания: Telegram-бот с ИИ-агентом (естественный язык) и веб-панель для главы семьи (без авторизации).

## Стек

- Go: REST API + Telegram bot + LLM agent (OpenAI-compatible tool calling)
- PostgreSQL 16
- React + Vite + TypeScript (панель)
- Docker Compose

## Быстрый старт

1. Скопируйте конфиг и заполните секреты:

```bash
cp .env.example .env
```

Нужны:
- `TELEGRAM_BOT_TOKEN` — от [@BotFather](https://t.me/BotFather)
- `LLM_API_KEY` (+ при необходимости `LLM_BASE_URL`, `LLM_MODEL`)

2. Запустите всё одной командой:

```bash
docker compose up --build
```

Сервисы:
- API: http://localhost:18080
- Панель: http://localhost:3000
- Postgres: `localhost:15432` (user/pass/db: `uchet`)

## Как пользоваться

### Telegram

Пишите боту в личку обычным текстом:

- «кофе 350»
- «такси 1200 вчера»
- «сколько потратил на еду в этом месяце»
- «покажи траты за неделю»
- «последнее было не 350, а 450»
- «удали такси»

Каждый пользователь видит и меняет только свои траты.

### Веб-панель

Откройте http://localhost:3000 — список всех трат семьи, фильтры по участнику / периоду / категории, создание, редактирование и удаление. Авторизация не требуется (demo).

## Конфигурация

| Переменная | Назначение |
|---|---|
| `DATABASE_URL` | Postgres DSN |
| `TELEGRAM_BOT_TOKEN` | токен бота |
| `LLM_BASE_URL` | OpenAI-compatible endpoint |
| `LLM_API_KEY` | ключ LLM |
| `LLM_MODEL` | модель |
| `APP_TIMEZONE` | таймзона для «сегодня/вчера» (по умолчанию `Europe/Moscow`) |
| `HTTP_ADDR` | адрес API |

## Локальная разработка

```bash
# Postgres из compose
docker compose up -d postgres

export DATABASE_URL=postgres://uchet:uchet@localhost:15432/uchet?sslmode=disable
make run-api          # :8080, миграции на старте
make run-bot          # нужен .env с токенами
make web-dev          # Vite на :5173 с proxy на API
make test
```

## Архитектура

```
Telegram → bot → LLM agent (tools) → ExpenseService → Postgres
Web panel → REST /api/v1 → ExpenseService → Postgres
```

В боте `user_id` всегда берётся из Telegram update; аргументы LLM не могут подменить владельца.

## Категории

еда, транспорт, жильё, развлечения, здоровье, покупки, другое.

## Соответствие ТЗ

| Требование | Где |
|---|---|
| NL CRUD в Telegram без кнопок | `internal/agent`, `internal/telegram` |
| Только свои траты в боте | `ExpenseService.*ForUser` |
| Панель без auth, все траты | `web/`, `GET/POST/PATCH/DELETE /api/v1/expenses` |
| Фильтры member/period/category | query API + UI |
| Модель Expense | `migrations/001_init.sql` |
| Go backend | `cmd/api`, `cmd/bot` |
| docker compose | `docker-compose.yml` |
| LLM через конфиг | `LLM_BASE_URL`, `LLM_API_KEY`, `LLM_MODEL` |

## Работа с ИИ-ассистентом

Разрабатывал с Cursor (Composer).

Что сгенерировал ассистент:
- скелет Go-модуля, миграции, store/service/REST
- Telegram long polling и LLM-агент с tool calling
- веб-панель на React + Vite, docker compose, черновик README
- unit-тесты (domain, service, api, agent)

Что правил сам после ревью:
- слои `handler → service → store` и жёсткую привязку `user_id` из Telegram (LLM не выбирает владельца)
- ответы бота по add/list/update/delete — из Go-форматтера tool, без пересказа модели (Groq иначе искажал суммы и даты)
- панель: Mantine, пагинация, маскот; date/number-поля Mantine заменил на нативные input из‑за крашей
- прогон сценариев из ТЗ, `.env` под Groq, чистку БД перед демо

ИИ закрыл boilerplate. Решения по безопасности данных, контракту tools и стабильности UX принимал и проверял сам.
