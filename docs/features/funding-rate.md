# Futures funding rate

## Goal

Show the **current (predicted next) funding rate** and **recent settled history**
for a coin on Binance USD-M and Bybit linear perpetual, next to open interest,
so a user or the AI can ask “are longs paying to hold BTC?”

## Behavior

- `GET /api/v1/market/funding-rate?symbol=BTCUSDT`
  - `exchange=all` (default) returns **each venue separately** (rates are not averaged)
  - `exchange=binance` or `bybit` hoists `current` + `history` to the top level
  - `limit` settled prints (default 12, max 30)
- `GET /api/v1/market/open-interest` includes the same payload on `funding`
- `current` is the **predicted next** payment (not yet settled)
- `history` is settled payments, **newest first**
- `rate` is decimal (`0.0001` = 0.01%); `ratePct` is percent
- `payer` is who pays at settlement: `long` (positive rate) or `short` (negative)
- `avgLast3` is the mean of the last three settlements (~24h at 8h)
- Durable predicted + settled samples are also stored in SQLite. Query them with
  `GET /api/v1/market/futures-history?metric=funding`. See [`futures-history.md`](futures-history.md).

Sources (public REST, no API key):

- Binance: `GET /fapi/v1/premiumIndex` + `GET /fapi/v1/fundingRate`
- Bybit: `GET /v5/market/tickers?category=linear` + `GET /v5/market/funding/history`

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/funding.go` |
| Adapters | `adapter/binance/funding.go`, `adapter/bybit/funding.go` |
| Service | `GetFundingRate` (also attached by `GetOpenInterest`) |
| HTTP | `GET /api/v1/market/funding-rate` |
| MCP / AI | `get_funding_rate` (also on `get_open_interest.funding`) |
| Telegram | `/funding <symbol>` and a Funding block on `/oi` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ ./internal/transport/http/handler/ ./internal/transport/mcp/
curl "http://localhost:8080/api/v1/market/funding-rate?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/open-interest?symbol=BTCUSDT"
```

## Limits

- Not Coinbase. Not COIN-M / inverse.
- Combined `all` does not invent a blended rate.
- Interval defaults to 8h when the venue does not publish one.
- Informational only — not financial advice.
