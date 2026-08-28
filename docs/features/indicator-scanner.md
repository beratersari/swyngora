# Technical indicator scanner

## Goal

Users define **RSI**, **EMA crossover**, and **volume increase** rules. The backend evaluates those rules against **symbols on the client's watchlist** on a schedule, saves matches to history, and **never stores the same hit twice** for the same rule, symbol, and market bar.

Informational only — not financial advice.

## Product web UI

Route **`/signals`** (nav: Signals) is the swing-signal desk:

| Tab | Behavior |
|-----|----------|
| Setups | Client-side confluence: ≥2 of EMA / RSI / volume on the same pair+interval in 24h. Grade A = 3/3. Same-bar overlap is flagged. |
| Hits | Raw scanner match history with the exact trigger bar time (UTC, seconds) and a jump to the coin chart. |
| Rules | Create, edit, enable/disable, or delete rules. Pick RSI, EMA crossover, and/or volume, then **all must match** or **one is enough**. One-click **4h swing stack** still adds the three expert long-side filters as separate rules. |
| Lab | Historical backtest on one symbol with 1/5/20d forward returns. |

Coin detail overlays scanner hits as chart markers (toggle next to pump markers). Scanner still evaluates **watchlist symbols only**.

### Swing engine (from crypto_analyzer, hardened)

Backend `GET /api/v1/market/swing` and `GET /api/v1/swing/setups` run a **closed-bar** 4h + 1d engine:

- Patterns: EMA 9/21 cross, EMA50/200 pullback, SuperTrend, Wilder RSI recovery, MACD, Wilder ADX, BB squeeze, volume breakout.
- Quality: BTC regime (longs blocked in bear except mean-reversion), multi-TF EMA alignment, ADX chop filter, min 24h quote volume, fresh event required, min R:R **1.8** for trigger.
- Stops: farther of structure swing-low − 0.25 ATR and entry − 1.5 ATR (capped at 2.5 ATR). TP at least 1.8R (or 2.5 ATR if larger).
- Math uses Wilder RSI/ADX/ATR and SMA-seeded EMA (the source Python scanner used SMA-of-last-N RSI and first-price EMA seed).

Informational only. MCP: `analyze_swing`, `scan_swing_setups`.

**Not financial advice.** Live checker may match a forming bar; treat hits as informational until you confirm on a closed candle.

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/scanner/rules` | Create rule (`conditions` + `matchMode`, or legacy `type`) |
| `GET` | `/api/v1/scanner/rules` | List rules |
| `GET` | `/api/v1/scanner/rules/{id}` | Get one rule |
| `PATCH` | `/api/v1/scanner/rules/{id}` | Enable/disable or edit interval, conditions, periods, thresholds |
| `DELETE` | `/api/v1/scanner/rules/{id}` | Delete rule (+ cascaded results) |
| `GET` | `/api/v1/scanner/results` | Match history (`limit`, `offset`) |

Tenancy: same `clientId` / `X-Client-Id` model as watchlists.

## Conditions

Select one or more of:

| Condition | Params | Match |
|-----------|--------|--------|
| `rsi` | `rsiPeriod` (default 14), `rsiCondition` (`above`/`below`), `rsiThreshold` | Latest RSI vs threshold |
| `ma_crossover` | `maFastPeriod`, `maSlowPeriod`, `maDirection` (`golden_cross`/`death_cross`) | EMA fast crosses slow on the latest bar |
| `volume_increase` | `volumeLookback` (default 20), `volumeMinRatio` (default 2) | Last bar volume ≥ ratio × average of prior N bars |

`matchMode`:

| Mode | Result |
|------|--------|
| `all` (default) | Every selected condition must be true on the same bar |
| `any` | One selected condition is enough |

Send `conditions: ["rsi", "volume_increase"]` with `matchMode`. Legacy `type: rsi` (or `ma_crossover` / `volume_increase`) still creates a single-condition rule.

`interval` defaults to `1h` (any supported candle interval).

Edit later with `PATCH`: `enabled` pauses or resumes the rule without deleting it. Periods, thresholds, `matchMode`, `conditions`, and `interval` can change in place.

Combo evaluation loads enough candles for the **longest** selected condition (for example MA slow 200 → 230 bars), not a fixed 100.

## Deduping

A hit is stored only when the rule **becomes** true (previous bar false, latest bar true). If the same condition stays true on later candles, no new row is written. After it goes false and then true again, a new result is stored.

Each result also has `marketDataKey` = candle **open time**. Unique constraint:

`(ruleId, exchange, symbol, marketDataKey)`

Re-running the scanner on the same bar does not create another row.

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
| MCP | `create_scanner_rule`, `list_scanner_rules`, `update_scanner_rule`, `delete_scanner_rule`, `list_scanner_results` |

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
- The rule is **snapshotted when the job is queued**. Editing the live rule before the worker starts does not change this run.
- Signals use the same false→true onset as live hits (a condition that stays true is one signal).
- **No duplicate run** for the same client + rule snapshot + symbol + date range while status is pending/running/completed (returns existing job). Editing the rule and starting again creates a **new** job.
- After cancel/failed, a new job with the same fingerprint can be started.
- Each signal stores close at match and optional **calendar-day** forward returns (1 / 5 / 20 days) when future candles exist.
- Max range: 400 days. Progress fields: `processedBars`, `totalBars`, `progressPct`.

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/adapter/scannerstore/ ./internal/service/scanner/ ./internal/transport/http/handler/ -run "Scanner|Backtest|Forward" -count=1
```
