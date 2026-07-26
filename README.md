# Swyngora

AI-powered cryptocurrency (and stock) analysis platform — market data, analytics, and an assistant that can explain what the numbers mean.

**Current version:** see [`VERSION`](VERSION) (`0.1.0` development).

## What’s in the repo now

| Package | Purpose |
|---|---|
| [`backend/`](backend/) | Go HTTP API — multi-exchange market data, supply, indicators, watchlist |
| [`simple-frontend/`](simple-frontend/) | Static test UI for the API (not the product app) |
| [`frontend/`](frontend/) | Reserved for the production web UI |
| [`docs/`](docs/) | Feature notes and ADRs |
| [`AGENTS.md`](AGENTS.md) | Team & coding-agent conventions |

## Quick start

### Prerequisites

- Go 1.22+ (tested with Go 1.26)
- Any static file server for the simple frontend (e.g. Python 3)

### Backend

```bash
cd backend
go test ./...
go run ./cmd/server
# → http://localhost:8080
```

Example:

```bash
curl -s 'http://localhost:8080/api/v1/market/candles?symbol=BTCUSDT&interval=1h&limit=3' | jq .
curl -s 'http://localhost:8080/api/v1/market/ticker/24h?symbol=BTCUSDT' | jq .
curl -s 'http://localhost:8080/api/v1/market/spot?quote=USDT&sort=quoteVolume&limit=5' | jq .
curl -s 'http://localhost:8080/api/v1/market/supply?asset=BTC' | jq .
```

### Simple frontend (manual testing)

```bash
cd simple-frontend
python3 -m http.server 5173
# open http://localhost:5173 — set API base to http://localhost:8080 if needed
```

### Telegram bot (integrated in backend)

Optional transport in the same process as the HTTP API (no separate binary).

```bash
cd backend
cp .env.example .env   # set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID
go run ./cmd/server    # :8080 + Telegram long-poll when token is set
```

After editing tokens in `.env`, **restart the server**. See [`docs/features/telegram-bot.md`](docs/features/telegram-bot.md) and `backend/README.md`.

## Data sources

| Data | Source | Notes |
|---|---|---|
| Candlesticks | Binance public Spot REST | No API key for market data |
| Spot market list / search / sort | Binance public Spot REST | exchangeInfo + 24h tickers |
| 24h volume / ticker | Binance public Spot REST | Base + quote volume |
| Circulating / total / max supply | Binance marketing symbol list | Daily snapshot @ 03:00 UTC (cache-only requests); max null when undefined |

See [docs/features/market-data.md](docs/features/market-data.md) and [docs/adr/0001-binance-and-coingecko-market-sources.md](docs/adr/0001-binance-and-coingecko-market-sources.md).

## API contract

OpenAPI: [`backend/api/openapi/openapi.yaml`](backend/api/openapi/openapi.yaml)

## Git workflow

Integration branch: **`develop`**. Features: `feature/*` → MR into `develop`. Full rules in [`AGENTS.md`](AGENTS.md).

**Remotes:** `origin` = GitLab (primary team host); `beratersari` = private GitHub mirror (`https://github.com/beratersari/swyngora`). Publish shared branches to **both** (`git pushboth <ref>` — see AGENTS.md §3.8).

## License

TBD.
