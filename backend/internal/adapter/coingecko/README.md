# CoinGecko adapter

Public `/api/v3/coins/markets` is the supply/mcap fallback for delisted (or soon-delisted) assets missing from the Binance marketing snapshot.

Public `/api/v3/coins/{id}/ohlc` plus the markets last price are the CoinGecko path for **post-delist movement** when no other listed venue still trades the pair (`GET /api/v1/market/post-delist`).

Live prices, listings, and the default supply cache stay on Binance / venue adapters (ADR 0001). No API key.

```bash
cd backend
go test ./internal/adapter/coingecko/ -count=1
```
