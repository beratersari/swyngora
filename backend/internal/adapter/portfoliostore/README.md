# Portfolio store

SQLite persistence for paper-trading portfolios, spot positions, **tax lots** (FIFO/LIFO), trades, pending orders
(limit buy/sell, stop-loss), recurring buys, and **isolated margin** positions/orders/trades
(long/short, leverage, liquidation, SL/TP). Includes **reservations**, **partial fills**,
**in-place amend** (open GTC limit/stop, CAS on remaining+trigger), **bulk cancel-all**
(open + pending bracket exits, one market or all), and
status-gated cancel/fill so open state survives restarts. **Allocation baskets**
(`allocation_baskets` / `allocation_targets`) store named target mixes; rebalance is not a worker. **Risk limits** (`risk_limits`) store optional daily-loss and max-weight caps plus a UTC day equity snapshot.

```bash
go test ./internal/adapter/portfoliostore/ -count=1
```