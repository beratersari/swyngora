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

MCP: `analyze_swing`, `scan_swing_setups`.

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/swing.go`, `swing_indicators.go` |
| Service | `backend/internal/service/swing` |
| HTTP | `backend/internal/transport/http/handler/swing.go` |
| UI | `/signals` `SwingEngineGrid` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/swing/ ./internal/transport/http/handler/ -run 'Swing|ATR|ADX|MACD' -count=1
```
