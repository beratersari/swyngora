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

MCP: `analyze_swing`, `scan_swing_setups`, `list_custom_signal_factors`, `list_custom_signal_profiles`, `create_custom_signal_profile`, `update_custom_signal_profile`, `delete_custom_signal_profile`, `scan_custom_signals`.

### Custom mix

The built-in engine on Setups is unchanged. The **Custom** tab lets the user pick which swing patterns to include, assign percents that must sum to 100, and optionally add **liquidation hunt (up)** — the combined hunt `upScore`. Scoring matches the engine: only present patterns (score > 0) are averaged; a pattern that did not fire is omitted, not scored as 0. The built-in comparison uses the engine `swingScore`. Users can edit, duplicate, and mark one saved mix as default — Custom opens that mix. Mixes persist per `clientId` in the scanner SQLite file. User data export/import includes a `customSignals` section (preview lists `addNames` / `updateNames` / `conflicts`; merge updates a unique name; same-name conflicts need a pick or are skipped). Switching the default mix is a single SQLite transaction so a failed update cannot leave zero or two defaults.

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/swing.go`, `swing_indicators.go`, `custom_signal.go` |
| Service | `backend/internal/service/swing` |
| HTTP | `backend/internal/transport/http/handler/swing.go`, `swing_custom.go` |
| Store | `backend/internal/adapter/scannerstore` (`custom_signal_profiles`) |
| UI | `/signals` `SwingEngineGrid` + Custom tab `SignalsCustomPanel` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/swing/ ./internal/adapter/scannerstore/ ./internal/transport/http/handler/ -run 'Swing|ATR|ADX|MACD|CustomSignal' -count=1
```
