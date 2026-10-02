# adapter/okx

Public OKX v5 adapter for **USDT-margined SWAP** futures: open interest,
funding, live liquidations, liquidation history backfill, 5m quote-volume
candles for venue mix, and SWAP OHLC for the liquidation heatmap.

## Purpose

CoinGlass-style liquidation / OI / funding desks need more than Binance and
Bybit. This package is futures-only — it is **not** a spot listing venue
(`SupportedExchanges` stays unchanged).

No API key. Base `https://www.okx.com`, public WS
`wss://ws.okx.com:8443/ws/v5/public`.

## Layout

| File | Role |
|---|---|
| `client.go` | HTTP GET + `{code,data}` envelope |
| `symbol.go` | `BTCUSDT` ↔ `BTC-USDT-SWAP` |
| `instruments.go` | `ctVal` cache (contract size in base) |
| `openinterest.go` | current OI + 5m rubik history (`oiCcy` / `oiUsd`) |
| `funding.go` | predicted next rate + settled history |
| `liqhub.go` | `liquidation-orders` SWAP websocket |
| `liqhist.go` | REST `liquidation-orders?state=filled` backfill |
| `futures_volume.go` | 5m SWAP candles (`volCcyQuote`) for volume share |
| `candles.go` | SWAP OHLC (`/api/v5/market/candles`) for the liquidation heatmap. Daily bars use `1Dutc` |
| `futures_depth.go` | SWAP book (`/api/v5/market/books`); contract size × `ctVal` is base quantity |
| `trades.go` | Newest SWAP trades (`/api/v5/market/trades`) as taker prints |
| `basis.go` | SWAP last and mark versus the USDT index |

## Symbols

Internal symbol stays `BTCUSDT`. OKX instId is `BTC-USDT-SWAP`. Liquidation
`sz` is contracts; notional is `sz * ctVal * bkPx` (BTC-USDT-SWAP `ctVal` is
0.01). Side uses `posSide` (long/short closed).

## How to test

```bash
cd backend
go test ./internal/adapter/okx/
```
