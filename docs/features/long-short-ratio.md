# Futures long/short ratio

## Goal

Show the **current long/short account ratio** and **recent 5-minute history**
for a coin on Binance USD-M and Bybit linear perpetual, so a user or the AI
can ask “are more traders long or short BTC?”

## Behavior

- `GET /api/v1/market/long-short-ratio?symbol=BTCUSDT`
  - `exchange=all` (default) returns **each venue separately** (not averaged)
  - `exchange=binance` or `bybit` hoists `current` + `history` to the top level
  - `limit` 5m prints (default 24 ≈ 2 hours, max 100)
- `GET /api/v1/market/open-interest` includes the same payload on `longShort`
- This is the **share of accounts** that are long vs short, **not** position size
- `ratio` is long/short (1 = even). `bias` is `long` if ratio ≥ 1.05, `short` if ≤ 0.95
- `change` on each venue is the signed ratio move vs the previous 5m print

Sources (public REST, no API key):

- Binance: `GET /futures/data/globalLongShortAccountRatio?period=5m`
- Bybit: `GET /v5/market/account-ratio?category=linear&period=5min` (`buyRatio` / `sellRatio`)

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/longshort.go` |
| Adapters | `adapter/binance/longshort.go`, `adapter/bybit/longshort.go` |
| Service | `GetLongShortRatio` (also attached by `GetOpenInterest`) |
| HTTP | `GET /api/v1/market/long-short-ratio` |
| MCP / AI | `get_long_short_ratio` (also on `get_open_interest.longShort`) |
| Telegram | `/ls <symbol>` and a Long/short block on `/oi` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ ./internal/transport/http/handler/ ./internal/transport/mcp/
curl "http://localhost:8080/api/v1/market/long-short-ratio?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/open-interest?symbol=BTCUSDT"
```

## Limits

- Not Coinbase. Not COIN-M / inverse.
- Not Binance top-trader position ratio (account-count only, for cross-venue compare).
- Combined `all` does not invent a blended ratio.
- Informational only — not financial advice.
