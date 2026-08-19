# Order-book history store

SQLite persistence for regular **spot order-book** samples (grouped bids/asks,
spread, total liquidity, imbalance, and large walls).

Duplicates are rejected by primary key `(exchange, symbol, sampled_at_ms)`.
A failed insert for one venue does not affect the others. Rows survive process
restarts. Default file: `data/orderbook.db`.

```bash
go test ./internal/adapter/bookhiststore/ -count=1
```
