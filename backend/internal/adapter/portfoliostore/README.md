# Portfolio store

SQLite persistence for paper-trading portfolios, spot positions, trades, pending orders
(limit buy/sell, stop-loss), recurring buys, and **isolated margin** positions/orders/trades
(long/short, leverage, liquidation, SL/TP). Includes **reservations**, **partial fills**, and
status-gated cancel/fill so open state survives restarts.

```bash
go test ./internal/adapter/portfoliostore/ -count=1
```