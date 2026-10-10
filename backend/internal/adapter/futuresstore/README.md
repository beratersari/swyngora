# Futures history store

SQLite persistence for Binance USD-M, Bybit linear, and OKX USDT-swap **open interest**,
**funding**, **long/short ratio** (including Binance top-trader size/account
shares), **liquidation** events, and **taker buy/sell bars** used for CVD.
Hyperliquid rows in the same file are sampled open interest and hourly funding.
There is no Hyperliquid long/short, tape, or liquidation feed.
A taker minute also stores whale and retail notional when each print's size
was known. `size_known` stays 0 for older minutes, and a later total-only
write does not erase a sized minute.

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
