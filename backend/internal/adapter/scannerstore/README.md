# Scanner store

SQLite persistence for technical indicator scanner rules and match history.
Rules store selected `conditions` and `match_mode` (`all` / `any`).
`UpdateRule` persists enable/disable and parameter edits.
Results are unique on `(rule_id, exchange, symbol, market_data_key)` so the same
bar cannot produce a duplicate hit.

```bash
go test ./internal/adapter/scannerstore/ -count=1
```
