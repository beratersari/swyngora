# Feature: Telegram bot

## Problem / goal

Use Swyngora market data from Telegram with a small command set (**no AI**): prices, spot lists, lowest mcap, supply, RSI/EMA, and a personal watchlist.

## Architecture

The bot is an **optional transport** inside the Go backend process (not a separate binary):

```text
Telegram long-poll → transport/telegram → service/market + service/watchlist → adapters
HTTP API           → transport/http     → (same services)
```

Enabled when `TELEGRAM_BOT_TOKEN` is set. Empty token → API-only server.

### Allowlist (fail closed)

| Config | Behavior |
|--------|----------|
| `TELEGRAM_CHAT_ID` and/or `TELEGRAM_ALLOWED_CHAT_IDS` | Only those chats may use the bot |
| Neither set, and `TELEGRAM_ALLOW_ALL` unset/false | **Bot does not start** (logged error) |
| `TELEGRAM_ALLOW_ALL=true` with empty allowlist | Public bot (any chat) — opt-in only |

## Commands

See `backend/README.md` (includes `/lowmcap` and `/lowmcap all`).

Free-text messages are **not** treated as `/price` (use explicit commands).

## Where the code lives

| Area | Path |
|------|------|
| Long-poll runner | `backend/internal/transport/telegram/bot.go` |
| Commands | `backend/internal/transport/telegram/commands.go` |
| Telegram HTTP API | `backend/internal/transport/telegram/client.go` |
| Formatters | `backend/internal/transport/telegram/format.go` |
| Wiring | `backend/cmd/server/main.go` |
| Config | `backend/internal/platform/config` |

## How to run

```bash
cd backend
cp .env.example .env   # set TELEGRAM_BOT_TOKEN + TELEGRAM_CHAT_ID
go run ./cmd/server    # starts HTTP :8080 + Telegram poller
```

## After changing tokens

**Restart the backend process** (Ctrl+C, then `go run ./cmd/server`). Env is loaded only at startup (`.env` / `backend/.env`).

## Tests

```bash
cd backend && go test ./internal/transport/telegram/...
```

## Limitations

- No alerts / push (v1)
- No AI
- Watchlist is in-memory (lost on process restart)
- Single process should run long-poll for a given bot token
