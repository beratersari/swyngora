# Radar store

SQLite persistence for Setup Radar hits and after-window fills.

A new hit is stored only when that coin has no case in the last 4 hours
(one row per appearance). After-rows start empty and are filled when
each 15m / 1h / 4h horizon is due.

## Layout

- `sqlite.go` — cases + afters
- `sqlite_test.go` — insert-if-quiet, list, fill

## How to test

```bash
cd backend
go test ./internal/adapter/radarstore/ ./internal/service/radar/ ./internal/domain/ -run Radar -count=1
```

## Config

`RADAR_DB_PATH` (default `data/radar.db`).

## Dependencies

Domain `RadarCasePort`. The radar service records scan hits and fills
after-prices from candles.
