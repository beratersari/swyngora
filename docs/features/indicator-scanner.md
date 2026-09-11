# Technical indicator scanner

## Goal

Users define **RSI**, **EMA crossover**, and **volume increase** rules. The backend evaluates those rules against **symbols on the client's watchlist** on a schedule, saves matches to history, and **never stores the same hit twice** for the same rule, symbol, and market bar.

Informational only — not financial advice.

## Product web UI

Route **`/signals`** (nav: Signals) is the swing-signal desk:

| Tab | Behavior |
|-----|----------|
| Setups | API confluence on `GET /api/v1/scanner/results`: ≥2 of EMA / RSI / volume on the same pair+interval in 24h. Grade A = 3/3. Same-bar overlap is flagged. |
| Custom | User-picked swing-engine factors + optional liquidation-hunt up score, with live weight preview and saved mixes (`GET /api/v1/swing/custom/scan`). Existing swing cards stay on Setups. |
| Hits | Raw scanner match history with the exact trigger bar time (UTC, seconds) and a jump to the coin chart. |
| Rules | Create, edit, enable/disable, or delete rules. Pick RSI, EMA crossover, and/or volume, then **all must match** or **one is enough**. One-click **4h swing stack** still adds the three expert long-side filters as separate rules. |
| Lab | Historical backtest on one symbol, or a custom mix on the whole watchlist, with 1/5/20d forward returns and first-touch TP/SL trades. |

Coin detail overlays scanner hits as chart markers (toggle next to pump markers). Scanner still evaluates **watchlist symbols only**.

### Swing engine (from crypto_analyzer, hardened)

Backend `GET /api/v1/market/swing` and `GET /api/v1/swing/setups` run a **closed-bar** 4h + 1d engine:

- Patterns: EMA 9/21 cross, EMA50/200 pullback, SuperTrend, Wilder RSI recovery, MACD, Wilder ADX, BB squeeze, volume breakout.
- Quality: BTC regime (longs blocked in bear except mean-reversion), multi-TF EMA alignment, ADX chop filter, min 24h quote volume, fresh event required, min R:R **1.8** for trigger.
- Stops: farther of structure swing-low − 0.25 ATR and entry − 1.5 ATR (capped at 2.5 ATR). TP at least 1.8R (or 2.5 ATR if larger).
- Math uses Wilder RSI/ADX/ATR and SMA-seeded EMA (the source Python scanner used SMA-of-last-N RSI and first-price EMA seed).

Informational only. MCP: `analyze_swing`, `scan_swing_setups`.

`GET /api/v1/market/indicators` includes `latest.zone` (`oversold` / `neutral` / `overbought`) from Wilder RSI bands. The web indicator panel does not re-threshold the number.

**Not financial advice.** Live checker may match a forming bar; treat hits as informational until you confirm on a closed candle.

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/scanner/rules` | Create rule (`conditions` + `matchMode`, or legacy `type`) |
| `GET` | `/api/v1/scanner/rules` | List rules |
| `GET` | `/api/v1/scanner/rules/{id}` | Get one rule |
| `PATCH` | `/api/v1/scanner/rules/{id}` | Enable/disable or edit interval, conditions, periods, thresholds |
| `DELETE` | `/api/v1/scanner/rules/{id}` | Delete rule (+ cascaded results) |
| `GET` | `/api/v1/scanner/results` | Match history (`limit`, `offset`) plus `setups` and `hits24h` |

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

Run a saved **scanner rule** or a **custom signal profile** over a date range to see past signals and what price did afterward. One coin, or a custom mix on every watchlist coin.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/scanner/backtests` | Start job (`ruleId` **or** `profileId`, `rangeStart`, `rangeEnd`). One coin: send `symbol`. Custom mix on the whole watchlist: `profileId` + `scope=watchlist` (or `symbol=watchlist`). `profileId=built_in` is the built-in swing mix. |
| `GET` | `/api/v1/scanner/backtests` | List jobs |
| `GET` | `/api/v1/scanner/backtests/{id}` | Progress (`progressPct`, `signalCount`, `status`) |
| `POST` | `/api/v1/scanner/backtests/{id}/cancel` | Cancel pending/running job |
| `GET` | `/api/v1/scanner/backtests/{id}/signals` | Match dates + `exchange`/`symbol` + `return1d` / `return5d` / `return20d` (%) + `trade` + `performance` (horizon stats, trade TP/SL, per-coin `markets` on watchlist jobs) |

### Behavior
- Job runs in the **background** (`pending` → `running` → `completed` | `canceled` | `failed`).
- The rule is **snapshotted when the job is queued**. Editing the live rule before the worker starts does not change this run.
- Signals use the same false→true onset as live hits (a condition that stays true is one signal).
- **No duplicate run** for the same client + rule snapshot + symbol + date range while status is pending/running/completed (returns existing job). Editing the rule and starting again creates a **new** job.
- After cancel/failed, a new job with the same fingerprint can be started.
- Each signal stores close at match and optional **calendar-day** forward returns (1 / 5 / 20 days) when future candles exist.
- Each signal also stores a paper **trade plan** when levels can be built: **side** (long, or short for RSI-below / death-cross rules), **leverage** (1–10x, sized so a full stop is about 10% of notional), **entry**, **stop**, and **take-profit**. Later candles decide the result: whichever of TP or SL the high/low hits **first** wins. If one bar touches both, a **smaller interval** is used to see which level printed first (4h → 1h → 15m → 5m → 1m). Outcome and leveraged `pnlPct` follow that finer first touch. If the smaller tape is missing or still tags both at 1m, the stop is taken. A gap through the **stop** at the next open fills at that **open** (PnL uses the open, not the stop). A gap through take-profit still fills at the TP. If neither is hit by the last bar, the trade stays **open** and PnL is marked to the last close. `pnlPct` is the leveraged percent.
- Job results include a **performance** summary per horizon: average return, median return, win rate (share of returns > 0), and sample count (signals that have that horizon). Missing future candles are left out of that column, not scored as 0.
- `performance.trades` counts TP hits, SL hits, and still-open trades, plus win rate (TP / closed) and average / median leveraged PnL of closed trades.
- Max range: 400 days. Progress fields: `processedBars`, `totalBars`, `progressPct`.
- **Custom profile jobs** (`kind=custom_profile`) replay **closed 4h** swing factors with the snapshotted weights. The mix score still uses the factors that have data. If any **selected** factor has no historical data at that bar (hunt, tape, or not enough candles), that timestamp is **skipped** — no signal. The job reports `skippedBars` and `missingFactors`. A signal fires only on complete bars when the mix first crosses **62**. Each hit lists the selected factors at that bar (`factors`: id, score, weight, status). Jobs list `sourceName` for the mix.
- **Watchlist custom jobs** (`scope=watchlist`) snapshot the client's watchlist when queued and run the same mix on every pair (each item's own exchange). Signals include `exchange` and `symbol`. `performance.markets` is **one row per snapshotted coin**, including coins with **0 signals** and coins that failed to load (`status=ok|no_data|error` plus `error`). An empty watchlist is rejected. If every coin fails the job fails; otherwise it completes and names how many coins had errors. Progress is by coin.
- A Lab fill is taken at the **next bar open** (the signal is only known at the close). Optional `startingBalance` (default 10000) and `riskPct` (default 1) size each fill so a full stop loses about that percent of **current** equity. `maxOpenRiskPct` (default 3) is the combined stop risk of **open** fills — e.g. 1% each with a 3% cap means at most three concurrent trades. A new signal is **skipped** when there is no room; when a fill closes, later signals can use the free risk. A trade that never hits TP/SL stays **open** at the price **at range end** (if the range cuts a candle, finer tape or that bar's open is used — not the later close). When several coins signal at the same time, **higher mix scores** take the shared risk first. A coin with an open fill does not open another until that fill closes. TP/SL **hit time** is set only when a candle **opens through** the level (the exact print). A wick on any interval — including 1m — is not an exact time: the result stays TP/SL, `hitTimeUnknown` is true, and shared-book risk stays used until that bar **closes**. Smaller intervals only tighten which bar we wait on. `rangeEndBalance` marks those opens in-range. `afterRangeBalance` applies only later TP/SL hits. MCP `start_backtest` accepts `scope=watchlist` or `symbol=watchlist`, `startingBalance`, `riskPct`, and `maxOpenRiskPct`.

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/adapter/scannerstore/ ./internal/service/scanner/ ./internal/transport/http/handler/ -run "Scanner|Backtest|Forward" -count=1
```
