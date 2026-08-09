# Swyngora

AI-powered cryptocurrency (and stock) analysis platform — market data, analytics, and an assistant that can explain what the numbers mean.

**Current version:** see [`VERSION`](VERSION) (`0.1.0` development).

## What’s in the repo now

| Package | Purpose |
|---|---|
| [`backend/`](backend/) | Go HTTP API — multi-exchange market data, supply, indicators, watchlist; **MCP** server (`cmd/mcp`) |
| [`ai/`](ai/) | Python multi-agent assistant (LangGraph orchestrator + market/web/X/analyst specialists) |
| [`simple-frontend/`](simple-frontend/) | Static test UI for the API (not the product app) |
| [`frontend/`](frontend/) | Production web UI (Ant Design + Lightweight Charts) |
| [`mobile/`](mobile/) | React Native client — **Chrome via react-native-web** (`npm run web`); no Expo |
| [`project-management/`](project-management/) | Local epics, tasks, board (frontend + mobile) |
| [`docs/`](docs/) | Feature notes, ADRs, design docs, GitLab PM defs |
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

### Mobile (React Native — Chrome)

Web-first scaffold under `mobile/` — **no Expo**, uses **react-native-web**.

```bash
cd mobile
npm install
npm run web
# open http://localhost:5180 in Chrome (or http://<WSL-IP>:5180 from Windows)
```

Optional: start the Go backend so Home can show API health.

See [`mobile/README.md`](mobile/README.md) and [`docs/design/mobile-project-initialization.md`](docs/design/mobile-project-initialization.md).

### Simple frontend (manual testing)

```bash
cd simple-frontend
python3 -m http.server 5173
# open http://localhost:5173 — set API base to http://localhost:8080 if needed
```

### AI assistant

```bash
# terminal 1 — API + MCP on the same process (:8080 REST, :8080/mcp)
cd backend && go run ./cmd/server

# terminal 2 — multi-agent CLI (Ollama default)
cd ai
uv sync && source .venv/bin/activate    # requires uv ≥ 0.12
export AI_LLM_PROVIDER=ollama
export SWYNGORA_API_URL=http://localhost:8080
swyngora-ai "What is BTC price and RSI on binance?"
```

See [`ai/README.md`](ai/README.md) and [`docs/features/ai-assistant.md`](docs/features/ai-assistant.md). LLM providers: **Ollama** or **Grok (xAI)** only.

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

## Product frontend (design phase)

| Doc | Purpose |
|---|---|
| [`docs/design/frontend-system-design.md`](docs/design/frontend-system-design.md) | Frontend architecture |
| [`docs/design/frontend-project-initialization.md`](docs/design/frontend-project-initialization.md) | **Epic A (first):** project init |
| [`docs/features/multi-exchange-spot-markets.md`](docs/features/multi-exchange-spot-markets.md) | **Epic B:** multi-exchange spot markets |
| [`docs/pm/frontend-epics-and-issues.md`](docs/pm/frontend-epics-and-issues.md) | GitLab epic/issue definitions |
| [`docs/pm/create-gitlab-epics.sh`](docs/pm/create-gitlab-epics.sh) | Create epics/issues via GitLab API |
| [`docs/pm/gitlab-mcp-setup.md`](docs/pm/gitlab-mcp-setup.md) | Configure GitLab project-management MCP in Grok |
| [`project-management/`](project-management/) | Local task board (INIT/MKT tasks) |
| [`project-management/decisions/001-antd-and-lightweight-charts.md`](project-management/decisions/001-antd-and-lightweight-charts.md) | UI kit + charts decision |
| [`docs/design/frontend-design-system.md`](docs/design/frontend-design-system.md) | Color, type, Text, Skeleton, isLoading |

Work order: **initialize `frontend/` toolchain** → then multi-exchange spot UI. Folder skeleton already lives under `frontend/src/`.

## API contract

OpenAPI: [`backend/api/openapi/openapi.yaml`](backend/api/openapi/openapi.yaml)

## Git workflow

Integration branch: **`develop`**. Features: `feature/*` → MR into `develop`. Full rules in [`AGENTS.md`](AGENTS.md).

**Remotes:** `origin` = GitLab (primary team host); `beratersari` = private GitHub mirror (`https://github.com/beratersari/swyngora`). Publish shared branches to **both** (`git pushboth <ref>` — see AGENTS.md §3.8).

## License

TBD.
