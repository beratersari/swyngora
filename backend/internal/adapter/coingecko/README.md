# CoinGecko adapter

Public `/api/v3/coins/markets` used **only** as a supply/mcap fallback for delisted (or soon-delisted) assets missing from the Binance marketing snapshot.

Live prices, listings, and the default supply cache stay on Binance / venue adapters (ADR 0001). No API key.

```bash
cd backend
go test ./internal/adapter/coingecko/ -count=1
```
