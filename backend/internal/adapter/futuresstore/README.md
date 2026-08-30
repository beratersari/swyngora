# Futures history store

SQLite persistence for Binance USD-M and Bybit linear **open interest**,
**funding**, **long/short ratio**, **liquidation** events, and **taker
buy/sell bars** used for CVD.

Duplicates are rejected by primary key:

- snapshots: `(metric, exchange, symbol, sampled_at_ms, predicted)`
- liquidations: `(exchange, symbol, side, time_ms, price, quantity)`

A failed insert for one venue does not affect the other. Rows survive process
restarts. Default file: `data/futures.db`.

`liquidation_coverage` also stores last print, last socket payload, last save
time, and disconnect gaps so a restart does not look like live coverage.

```bash
go test ./internal/adapter/futuresstore/ -count=1
```
