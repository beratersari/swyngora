# Swyngora

AI-powered cryptocurrency (and stock) analysis platform — market data, analytics, and an assistant that can explain what the numbers mean.

**Current version:** see [`VERSION`](VERSION) (`0.1.0` development).

## What’s in the repo now

| Package | Purpose |
|---|---|
| [`backend/`](backend/) | Go HTTP API — Binance candles & 24h volume, supply metadata |
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
curl -s 'http://localhost:8080/api/v1/market/supply?asset=BTC' | jq .
```

### Simple frontend (manual testing)

```bash
cd simple-frontend
python3 -m http.server 5173
# open http://localhost:5173 — set API base to http://localhost:8080 if needed
```

## Data sources

| Data | Source | Notes |
|---|---|---|
| Candlesticks | Binance public REST | No API key for market data |
| 24h volume / ticker | Binance public REST | Base + quote volume |
| Circulating / total / max supply | CoinGecko free public API | Binance market APIs do not expose supply |

See [docs/features/market-data.md](docs/features/market-data.md) and [docs/adr/0001-binance-and-coingecko-market-sources.md](docs/adr/0001-binance-and-coingecko-market-sources.md).

## API contract

OpenAPI: [`backend/api/openapi/openapi.yaml`](backend/api/openapi/openapi.yaml)

## Git workflow

Integration branch: **`develop`**. Features: `feature/*` → MR into `develop`. Full rules in [`AGENTS.md`](AGENTS.md).

## License

TBD.
