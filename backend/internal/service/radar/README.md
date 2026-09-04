# Radar service

Records Setup Radar appearances and fills what price did after
15m / 1h / 4h so the desk can show how useful past hits were.

## Layout

- `service.go` — persist scan hits, review, fill due afters
- `worker.go` — background scan (15m) and fill (1m)
- `service_test.go` — insert/dedupe and after-fill

## How to test

```bash
cd backend
go test ./internal/service/radar/ ./internal/adapter/radarstore/ ./internal/domain/ -run Radar -count=1
```

## Config

`RADAR_DB_PATH`, `RADAR_SCAN_INTERVAL` (15m), `RADAR_FILL_INTERVAL` (1m).

## Dependencies

`RadarCasePort` (SQLite) and market `ScanForensics` / ticker / candles.
