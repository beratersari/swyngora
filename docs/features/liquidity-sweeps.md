# Liquidity sweeps

## Goal

Price often **turns at a high or low a few times** (stops and resting
orders sit just beyond). Then it pokes **a little through** that level,
takes that liquidity, and **comes back**. That poke-and-return is a
**liquidity sweep**.

Useful when:

- a wick goes above last week’s high and the next bars close back under
- you want **which level** was taken, **how far** through, **how long**
  it stayed beyond, and **how much volume** printed in that move

## Behavior

`GET /api/v1/market/liquidity-sweeps?symbol=BTCUSDT`

- `exchange=all` (default): **Binance** and **Bybit** separately
- 15-minute **spot** candles, last ~7 days
- A level is a clustered swing high or low with **at least two** prior
  turns
- `side`: `high` (through a high, then back under) or `low` (through a
  low, then back over)
- `level` / `tests` / `levelTime`: the shelf and how many times it held
  before the poke
- `extreme` / `excursion` / `excursionPct`: farthest print beyond the level
- `duration` / `bars`: how long it stayed beyond (until a close back on
  the original side)
- `volume` plus `buyVolume` / `sellVolume` when the venue publishes
  taker-buy (Binance klines; Bybit is total-only)
- `status`: `swept` (back) or `open` (still beyond, under 2 hours)
- A poke that stays through for **more than 2 hours** is treated as a
  **breakout**, not a sweep

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/liquidity_sweep.go` |
| Service | `backend/internal/service/market/sweep.go` |
| HTTP | `GET /api/v1/market/liquidity-sweeps` |
| MCP / AI | `get_liquidity_sweeps` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/... -run Sweep
curl "http://localhost:8080/api/v1/market/liquidity-sweeps?symbol=BTCUSDT"
```
