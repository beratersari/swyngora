# Feature: Telegram bot

## Problem / goal

Use Swyngora market data **and the multi-agent AI** from Telegram: prices, spot lists, lowest mcap, supply, RSI/EMA, watchlist, and natural-language `/ask`.

## Architecture

```text
Telegram long-poll → transport/telegram
                       ├─ market / watchlist services (in-process)
                       └─ /ask → AI HTTP client → Python multi-agent (port 8090)
HTTP API           → same services + POST /api/v1/ai/chat + /mcp
```

One backend process (`cmd/server`) can auto-start the Python AI child when `AI_AUTOSTART=true`.

## Commands

| Command | Purpose |
|---------|---------|
| `/price`, `/spot`, `/lowmcap`, `/mcap`, `/rsi`, `/exchanges` | Market data |
| `/watch` … | Watchlist |
| **`/ask <question>`** | Multi-agent AI (market tools + web + X) |
| **`/ai <question>`** | Alias of `/ask` |

## AI setup

```bash
# 1) Install AI package once
cd ai && python3 -m venv .venv && source .venv/bin/activate && pip install -e .

# 2) backend/.env
TELEGRAM_BOT_TOKEN=...
TELEGRAM_CHAT_ID=...
AI_AUTOSTART=true
AI_PYTHON=/absolute/path/to/swyngora/ai/.venv/bin/python
AI_WORKDIR=../ai          # when running from backend/
AI_SERVICE_URL=http://127.0.0.1:8090
XAI_API_KEY=...           # if AI_LLM_PROVIDER=grok
# or Ollama running locally with AI_LLM_PROVIDER=ollama

# 3) Start backend (restarts AI child if configured)
cd backend && go run ./cmd/server
```

Then in Telegram:

```text
/ask What is BTC price and RSI on binance?
```

## Tests

```bash
cd backend && go test ./internal/transport/telegram/... ./internal/service/aiagent/...
```
