# Portfolio store

SQLite persistence for paper-trading portfolios, positions, trades, and pending orders
(limit buy/sell, stop-loss). Open orders survive restarts; fill/cancel are status-gated.

```bash
go test ./internal/adapter/portfoliostore/ -count=1
```