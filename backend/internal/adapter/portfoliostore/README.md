# Portfolio store

SQLite persistence for paper-trading portfolios, positions, trades, and pending orders
(limit buy/sell, stop-loss) including **reservations**, **remaining/filled quantity**, and
**partial fills**. Open orders and locks survive restarts; fill/cancel/reject are status-gated
and release unused reservations on cancel/reject.

```bash
go test ./internal/adapter/portfoliostore/ -count=1
```