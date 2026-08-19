# Cumulative volume delta (CVD)

## Goal

See the **buy vs sell difference over time**, not just the last few
minutes. CVD keeps adding aggressive **market-buy** notional minus
**market-sell** notional, then plots that path next to **price**.

Useful when:

- price barely moves but there are many market buys (absorption)
- price goes up but the sell side is actually stronger (divergence)

## Behavior

`GET /api/v1/market/cvd?symbol=BTCUSDT`

- `exchange=all` (default): **Binance** and **Bybit** separately plus `combined`
- 5-minute points: `buyNotional`, `sellNotional`, `delta`, running `cvd`,
  `price`, `priceChangePct`, `vsPrice`, and `divergence`
- **Divergence:** `price_up_cvd_down` or `price_down_cvd_up` on a point,
  on each 15m / 1h / 4h / 24h window, and on the series `divergence`
  object — how long the current split has been running (`duration`,
  `since`), and how far price (`priceMovePct`) and CVD (`cvdMove`)
  moved against each other
- Windows **15m / 1h / 4h / 24h**: CVD change vs price change (not only
  the latest total), labeled `confirms` | `opposite` | `absorption` | `mixed`
- Combined **adds** each venue’s delta on the overlapping **time range**
  (a quiet 5m slot on one side is treated as 0, not a hole). Combined
  `complete` follows both venues — one missing 5m bucket at the start
  does not make a 24h tape look unfinished. A short Bybit history still
  does **not** make combined complete. `overlapFrom` / `overlapTo` mark
  that range.
- Combined points and `contributions` show how much **Binance** and
  **Bybit** each added. `venueSplit` flags when one venue’s CVD rose
  while the other fell (`alignment=opposite`)

Binance uses the public 5m taker series (~24h+ immediately). Bybit uses
live trades; 24h `complete` after this process has been collecting. Bars
are also stored in `data/futures.db` (`taker_buckets`).

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/cvd.go` |
| Adapters | `adapter/binance/takerflow.go`, `adapter/bybit/takerflow.go`, `adapter/futuresstore` |
| Service | `backend/internal/service/market/cvd.go` |
| HTTP | `GET /api/v1/market/cvd` |
| MCP / AI | `get_cvd` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/adapter/futuresstore/ -run CVD
curl "http://localhost:8080/api/v1/market/cvd?symbol=BTCUSDT"
```
