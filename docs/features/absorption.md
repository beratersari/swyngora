# Absorption (large flow, little price)

## Goal

Sometimes a lot of **market sells** hit and price **barely drops**. The
same can happen with **market buys** and a price that will not rise.
Someone on the other side is **absorbing** that flow (passive bids or
asks taking the aggression).

Useful when:

- you see a dump in volume but the tape does not give way
- you want the **size** of the buy/sell flow next to the **price result**
- you want **which side** is absorbing and **how strong** that is

## Behavior

`GET /api/v1/market/absorption?symbol=BTCUSDT`

- `exchange=all` (default): **Binance** and **Bybit** separately plus `combined`
- Windows **15m / 1h / 4h / 24h**: `buyNotional`, `sellNotional`, `delta`,
  `priceFrom` / `priceTo` / `priceChangePct`
- `kind`: `bid` (bids absorbing market sells) or `ask` (asks absorbing
  market buys). Empty when price moved with the aggressive side or flow
  was mixed
- `absorber`: `buy` or `sell` (the passive side)
- `result`: `held` (price almost flat) or `pushed` (price moved against
  the aggressive side)
- `score` 0–100 and `grade` `weak` | `moderate` | `strong` | `extreme`
  from how one-sided the flow is and how little price delivered
- `current` / `episodes`: consecutive 5-minute bars of the same kind. A
  quiet gap starts a new run
- `venues` / `combined` are **futures**; `spotVenues` / `spotCombined`
  are **spot**

Futures uses Binance USD-M / Bybit linear taker volume. Spot uses Binance
5m kline taker-buy and Bybit spot trades. Combined uses the overlapping
time range.

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/absorption.go` |
| Service | `backend/internal/service/market/absorption.go` |
| HTTP | `GET /api/v1/market/absorption` |
| MCP / AI | `get_absorption` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/... -run Absorption
curl "http://localhost:8080/api/v1/market/absorption?symbol=BTCUSDT"
```
