# Futures open interest

## Goal

Show **current open interest** for a coin and how much OI **opened or closed**
over the last **5 minutes, 1 hour, 4 hours, and 24 hours** — as a USDT
**amount and a percent**, **per exchange** and combined — so a user or the AI
can ask “how much BTC OI was added in the last hour?”

## Behavior

- `GET /api/v1/market/open-interest?symbol=BTCUSDT`
  - `exchange=all` (default) sums **Binance USD-M**, **Bybit linear perpetual**, and **OKX USDT-swap**
  - `exchange=binance`, `bybit`, or `okx` for one venue
  - `exchange=hyperliquid` is that venue alone. `value` is the mark times coin size. The margin is USDC, lined up with the USDT name. It is not inside `exchange=all`
- Current reading:
  - `contracts` — venue-published outstanding size in the **base asset** (e.g. BTC)
  - `value` — USDT notional
- Each window includes past contracts/value, signed `change` / `changePct` /
  `changeValue` / `changeValuePct`, `direction` (`up` / `down` / `flat`),
  `flow` (`opened` / `closed` / `flat`), `openedValue` or `closedValue`
  (USDT notional added or closed), and `complete` (false when the historical
  sample is missing or older than the window slack)
- Combined `all` stays blank until every venue that was asked is in.
  A venue that failed is named in `missing` and is not counted as zero.
  A window that lacks a sample on any asked venue is blank even when the
  others have it. Per-venue rows still return that venue's current OI
  plus its own 1h / 4h / 24h windows
- How much of futures volume that OI change represents is
  `GET /api/v1/market/participation` (15m / 1h / 4h). See [`participation.md`](participation.md)
- `funding` is attached (predicted next rate + recent settlements, plus OI-weighted `pressure`). See [`funding-rate.md`](funding-rate.md)
- `longShort` is attached (account long/short ratio, plus Binance crowd-vs-top `gap` and top-vs-price `divergence`). See [`long-short-ratio.md`](long-short-ratio.md)
- Cached ~30s (`OPEN_INTEREST_CACHE_TTL`)

Sources (public REST, no API key):

- Binance: `GET /fapi/v1/openInterest` + mark price + `/futures/data/openInterestHist?period=5m`
- Bybit: `GET /v5/market/tickers?category=linear` + `/v5/market/open-interest` (5min, paged) + 5m klines for hist notional.
- OKX: `GET /api/v5/public/open-interest?instId=BTC-USDT-SWAP` (`oiCcy` / `oiUsd`) + rubik `open-interest-history` period `5m`.
  **Single-sided only:** Bybit still publishes bilateral `openInterest` (~2×) and unilateral `singleOpenInterest`. Current and history use `singleOpenInterest` / `singleOpenInterestValue`. If the single field is missing, the bilateral figure is halved.

Unlike liquidations, history comes from the exchange — 24h change is available
immediately, not only after this process has been running 24h.

Durable samples (5m, both venues) are also stored in SQLite. Query them with
`GET /api/v1/market/futures-history?metric=open_interest`. See [`futures-history.md`](futures-history.md).

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/open_interest.go` |
| Adapters | `adapter/binance/openinterest.go`, `adapter/bybit/openinterest.go`, `adapter/okx/openinterest.go` |
| Service | `GetOpenInterest` |
| HTTP | `GET /api/v1/market/open-interest` |
| MCP / AI | `get_open_interest` |
| Telegram | `/oi <symbol> [binance\|bybit\|all]` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ ./internal/transport/http/handler/ ./internal/transport/mcp/
curl "http://localhost:8080/api/v1/market/open-interest?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/open-interest?symbol=BTCUSDT&exchange=binance"
```

## Limits

- Not Coinbase. Not COIN-M / inverse.
- Contract units: Binance USD-M is already one-sided; Bybit linear is normalized to `singleOpenInterest` (not the 2× `openInterest` both-sides field).
- Combined totals sum those normalized figures; venues can still disagree slightly on methodology.
- Bybit historical USDT value is single-sided OI × 5m close (hist API has no notional field).
- Informational only — not financial advice.
