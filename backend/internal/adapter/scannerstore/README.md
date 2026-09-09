# Scanner store

SQLite persistence for technical indicator scanner rules and match history.
Rules store selected `conditions` and `match_mode` (`all` / `any`).
`UpdateRule` persists enable/disable and parameter edits.
Backtests store `rule_json` (the rule or custom mix as queued) so the worker does not
re-read a later edit. `kind` is `scanner_rule` or `custom_profile`. `rule_id` is the
source id and is not a foreign key (custom profile ids are not scanner rules).
Results are unique on `(rule_id, exchange, symbol, market_data_key)` so the same
bar cannot produce a duplicate hit.

```bash
go test ./internal/adapter/scannerstore/ -count=1
```
