# Feature: Swing engine

## Problem / goal

Port the **crypto_analyzer** swing auto-scanner into Swyngora and harden the junior-finance math: closed bars only, Wilder RSI/ADX/ATR, SMA-seeded EMA, BTC regime, and ATR/structure stops with a real R:R floor.

Informational only — not financial advice.

## Behavior

1. Fetch **closed** 4h (200) + 1d (220) candles.
2. Detect patterns: EMA 9/21 cross, EMA50/200 pullback, SuperTrend(10,3), RSI recovery, MACD hist, ADX+DI, BB squeeze + volume dry-up, volume breakout.
3. Quality gates: min 2 patterns, **fresh** event for trigger, 24h quote volume ≥ 1M, BTC regime (bear blocks trend longs; chop allows mean-reversion/squeeze), multi-TF EMA alignment, ADX double-chop block, max risk 8%, min R:R **1.5** watch / **1.8** trigger.
4. Stop = farther of swing-low − 0.25 ATR and entry − 1.5 ATR (cap 2.5 ATR). TP = max(1.8R, entry + 2.5 ATR).
5. Stages: `rejected` | `watch` | `trigger`. Grade A/B/C from score + stage.

## API

| Method | Path | Auth |
|--------|------|------|
| `GET` | `/api/v1/market/swing?exchange=&symbol=` | Public |
| `GET` | `/api/v1/swing/setups?limit=&exchange=` | `X-Client-Id` (watchlist) |
| `GET` | `/api/v1/swing/custom/factors` | Public catalog |
| `GET`/`POST` | `/api/v1/swing/custom/profiles` | Saved mixes (`X-Client-Id`) |
| `GET`/`PATCH`/`DELETE` | `/api/v1/swing/custom/profiles/{id}` | One mix |
| `GET` | `/api/v1/swing/custom/scan` | Watchlist scored with a mix (`profileId` or `weight*`) |
| `POST` | `/api/v1/scanner/backtests` | Lab: pass `profileId` (or `built_in`) instead of `ruleId` to replay that mix |

MCP: `analyze_swing`, `scan_swing_setups`, `list_custom_signal_factors`, `list_custom_signal_profiles`, `create_custom_signal_profile`, `update_custom_signal_profile`, `delete_custom_signal_profile`, `scan_custom_signals`, `start_backtest`.

### Custom mix

The built-in engine on Setups is unchanged. The **Custom** tab lets the user pick which swing patterns to include, assign percents that must sum to 100, and optionally add **liquidation hunt (up)**, **forensics tape**, and **similar setups**. Hunt uses the combined hunt `upScore`. Forensics and similar setups score whether the current tape / after-path **supports a long custom signal or goes against it** (up lean raises the factor; down lean lowers it; mixed or missing tape is omitted). The after-path uses the **same range as the swing bar** (a 4h signal uses the 4h similar/forensics outcome, not the 1h desk default). Coverage and missing OI / taker / book / options are checked on **every** similar setup, not only the first hit: if any selected tape field is missing, that setup is omitted from the after-path vote — it is not treated as evidence. The custom result reports `tapeSetups` / `tapeUsed` / `tapeThin` and `tapeFresh` (how old the current OI, taker, book, and options prints are; a print older than its slack is marked stale). Current tape with **no** OI / taker / book / options prints marks forensics and similar setups **missing** — not fresh at quality 100. Each missing series cuts that factor's quality by a quarter (OI only → 25%). It also reports one **evidence** score (`strong` / `fair` / `weak`) from the data quality of the factors in the mix — missing weighted factors, incomplete similar setups, and stale tape pull evidence down so a high lean score is not treated as reliable by itself. Quality is not completeness: an 80 quality mix is not called complete. `evidence.fresh` / `evidence.old` / `evidence.missing` count how many weighted factors have current data, stale tape, or no data. They are off until the user gives them weight — classic Setups and an unweighted custom mix stay the same. Backtest Lab can replay a saved mix (or the built-in swing mix) on one symbol: hunt and tape stay omitted, and a signal fires when the mix first crosses 62. Scoring matches the engine: only present patterns (score > 0) are averaged; a pattern that did not fire is omitted, not scored as 0. The built-in comparison uses the engine `swingScore`. Users can edit, duplicate, and mark one saved mix as default — Custom opens that mix. Mixes persist per `clientId` in the scanner SQLite file. User data export/import includes a `customSignals` section (preview lists `addNames` / `updateNames` / `conflicts`; merge updates a unique name; same-name conflicts need a pick or are skipped). Switching the default mix is a single SQLite transaction so a failed update cannot leave zero or two defaults.

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/swing.go`, `swing_indicators.go`, `custom_signal.go`, `custom_signal_tape.go` |
| Service | `backend/internal/service/swing` |
| HTTP | `backend/internal/transport/http/handler/swing.go`, `swing_custom.go` |
| Store | `backend/internal/adapter/scannerstore` (`custom_signal_profiles`) |
| UI | `/signals` `SwingEngineGrid` + Custom tab `SignalsCustomPanel` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/swing/ ./internal/adapter/scannerstore/ ./internal/transport/http/handler/ -run 'Swing|ATR|ADX|MACD|CustomSignal' -count=1
```
