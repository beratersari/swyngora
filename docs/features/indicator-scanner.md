# Technical indicator scanner

## Goal

Users define **RSI**, **EMA crossover**, and **volume increase** rules. The backend evaluates those rules against **symbols on the client's watchlist** on a schedule, saves matches to history, and **never stores the same hit twice** for the same rule, symbol, and market bar.

Informational only — not financial advice.

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/scanner/rules` | Create rule (`type`, `interval`, type-specific params) |
| `GET` | `/api/v1/scanner/rules` | List rules |
| `GET` | `/api/v1/scanner/rules/{id}` | Get one rule |
| `DELETE` | `/api/v1/scanner/rules/{id}` | Delete rule (+ cascaded results) |
| `GET` | `/api/v1/scanner/results` | Match history (`limit`, `offset`) |

Tenancy: same `clientId` / `X-Client-Id` model as watchlists.

## Rule types

| `type` | Params | Match condition |
|--------|--------|-----------------|
| `rsi` | `rsiPeriod` (default 14), `rsiCondition` (`above`/`below`), `rsiThreshold` | Latest RSI vs threshold |
| `ma_crossover` | `maFastPeriod`, `maSlowPeriod`, `maDirection` (`golden_cross`/`death_cross`) | EMA fast crosses slow on the latest bar |
| `volume_increase` | `volumeLookback` (default 20), `volumeMinRatio` (default 2) | Last bar volume ≥ ratio × average of prior N bars |

`interval` defaults to `1h` (any supported candle interval).

## Deduping

Each result has `marketDataKey` = candle **open time**. Unique constraint:

`(ruleId, exchange, symbol, marketDataKey)`

Re-running the scanner on the same closed bar does not create another row. A new bar can produce a new result.

## Background job

- Interval: `SCANNER_CHECK_INTERVAL` (default `60s`), also runs once on process start.
- Loads enabled rules → client's watchlist → candles → evaluate → insert if new.

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/scanner.go` |
| Store | `backend/internal/adapter/scannerstore` |
| Service | `backend/internal/service/scanner` |
| HTTP | `backend/internal/transport/http/handler/scanner.go` |
| MCP | `create_scanner_rule`, `list_scanner_rules`, `delete_scanner_rule`, `list_scanner_results` |

## Config

| Env | Default |
|-----|---------|
| `SCANNER_DB_PATH` | `data/scanner.db` |
| `SCANNER_CHECK_INTERVAL` | `60s` |

## Historical backtests

Run a saved rule over a symbol and date range to see past signals and what price did afterward.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/scanner/backtests` | Start job (`ruleId`, `symbol`, `rangeStart`, `rangeEnd`) |
| `GET` | `/api/v1/scanner/backtests` | List jobs |
| `GET` | `/api/v1/scanner/backtests/{id}` | Progress (`progressPct`, `signalCount`, `status`) |
| `POST` | `/api/v1/scanner/backtests/{id}/cancel` | Cancel pending/running job |
| `GET` | `/api/v1/scanner/backtests/{id}/signals` | Match dates + `return1d` / `return5d` / `return20d` (%) |

### Behavior
- Job runs in the **background** (`pending` → `running` → `completed` | `canceled` | `failed`).
- **No duplicate run** for the same client + rule + symbol + date range while status is pending/running/completed (returns existing job).
- After cancel/failed, a new job with the same fingerprint can be started.
- Each signal stores close at match and optional **calendar-day** forward returns (1 / 5 / 20 days) when future candles exist.
- Max range: 400 days. Progress fields: `processedBars`, `totalBars`, `progressPct`.

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/adapter/scannerstore/ ./internal/service/scanner/ ./internal/transport/http/handler/ -run "Scanner|Backtest|Forward" -count=1
```
