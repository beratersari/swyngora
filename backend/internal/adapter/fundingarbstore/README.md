# fundingarbstore

SQLite persistence for funding-arb watches and open/closed signals.

A watch with `symbol=*` follows the funding-arb scan (one open signal per
coin). Schema v2 unique-indexes open signals on `(watch_id, symbol)`.
`UpdateWatch` writes min profit and other settings; `status` is `active`,
`paused`, or `expired`. Optional `expires_at` ends the follow.

## Layout

- `sqlite.go` — watches + signals (FK cascade on watch delete)
- `sqlite_test.go` — create / list / signal open-close / purge

## How to test

```bash
cd backend
go test ./internal/adapter/fundingarbstore/ ./internal/service/fundingarb/ -count=1
```

## Config

`FUNDING_ARB_DB_PATH` (default `data/fundingarb.db`).

## Dependencies

Domain `FundingArbWatchPort`. The market service evaluates quotes; the alert service delivers webhooks.
