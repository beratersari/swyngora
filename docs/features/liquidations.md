# Futures liquidations

## Goal

Show Coinglass-style **long vs short liquidation** totals for a coin over the last
**5 minutes, 1 hour, 4 hours, and 24 hours**, so a user or the AI can ask
“how much BTC was liquidated in the last 24 hours?”

## Behavior

- `GET /api/v1/market/liquidations?symbol=BTCUSDT`
  - `exchange=all` (default) sums **Binance USD-M** and **Bybit linear perpetual**
  - `exchange=binance` or `bybit` for one venue
- `GET /api/v1/market/liquidations/overview`
  - Market-wide **1h / 4h / 12h / 24h** totals (no symbol)
  - `window` (default `24h`) ranks `coins` by total notional for a treemap
  - `limit` default 50, max 100
- Each window includes:
  - `longNotional` / `shortNotional` / `totalNotional` (USDT)
  - `count`
  - `biggest` (side, price, qty, notional, venue, time)
  - `complete` — false until **this coin on this venue** has had a **live websocket**
    for the full window. Time does not count if the socket never connects or drops.
    Combined `all` uses the shorter live coverage of the venues watching that coin.
- Streams (background, always on):
  - Binance: `wss://fstream.binance.com/ws/!forceOrder@arr` (all USD-M symbols). At most the **largest** liquidation per symbol per ~1s.
  - Bybit: `wss://stream.bybit.com/v5/public/linear` topic `allLiquidation.{symbol}`. Seed majors + top linear USDT by 24h turnover; querying a symbol also subscribes it.
- Side meaning: **long** = long positions were force-closed; **short** = shorts were force-closed.
- Live windows are computed from an in-memory 24h book. Every print is written to SQLite (Binance and Bybit in separate rows). On startup the last **24 hours** are reloaded, including live-coverage clocks, so 1h/4h/12h/24h totals stay usable after a restart. A dropped persist queue is written synchronously instead of discarded. Stored rows: `GET /api/v1/market/futures-history?metric=liquidations`. See [`futures-history.md`](futures-history.md).
- Hypothetical “what if spot is walked to force liquidations” is a separate model: [`liquidation-hunt.md`](liquidation-hunt.md).
- Cascade detector: `GET /api/v1/market/liquidation-cascade`
  - Compares the last **1m / 5m / 15m** long and short notional to that stream's own typical (median of prior blocks over ~6 hours).
  - `grade` is `quiet` / `elevated` (≥2×) / `cascade` (≥4× + enough hits) / `extreme` (≥8×).
  - Binance and Bybit are scored **separately**. `both.agree` is true only when the **same side** is `cascade` or hotter on both venues.
  - `symbol=all` pools every tracked coin (market-wide risk). `GET /liquidation-cascade/scan` adds ranked bursting coins.

## Cascade grades

| Grade | Meaning |
|---|---|
| `quiet` | Current burst is in a normal range for that stream |
| `elevated` | About 2× typical, at least two hits |
| `cascade` | About 4× typical with enough prints in the window |
| `extreme` | About 8× typical with a larger hit count |

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/liquidation.go`, `liquidation_levels.go`, `liquidation_cascade.go` |
| Adapters | `adapter/binance/liqhub.go`, `adapter/bybit/liqhub.go` |
| Service | `GetLiquidations`, `GetLiquidationOverview`, `GetLiquidationLevels`, `GetLiquidationCascade`, `ScanLiquidationCascades` |
| HTTP | `GET /api/v1/market/liquidations`, `/liquidations/overview`, `/liquidation-levels`, `/liquidation-cascade`, `/liquidation-cascade/scan` |
| MCP / AI | `get_liquidations`, `get_liquidation_overview`, `get_liquidation_levels`, `get_liquidation_cascade`, `scan_liquidation_cascades` |
| Web | `/liquidations` — cards + treemap; **Chart** tab; **Cascade** tab (coin or market); Heatmap tab |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ ./internal/transport/http/handler/
curl "http://localhost:8080/api/v1/market/liquidations?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/liquidations?symbol=BTCUSDT&exchange=binance"
curl "http://localhost:8080/api/v1/market/liquidations/overview?window=24h&limit=20"
curl "http://localhost:8080/api/v1/market/liquidation-levels?symbol=BTCUSDT&exchange=all&range=24h"
curl "http://localhost:8080/api/v1/market/liquidation-levels?symbol=all&exchange=binance"
curl "http://localhost:8080/api/v1/market/liquidation-cascade?symbol=BTCUSDT&exchange=all"
curl "http://localhost:8080/api/v1/market/liquidation-cascade?symbol=all"
curl "http://localhost:8080/api/v1/market/liquidation-cascade/scan"
```

## Limits

- Not Coinbase. Not COIN-M / inverse.
- 24h is complete only after that coin has had a live feed for 24h on the requested venue(s). A newly tracked coin, or a dropped socket, stays `complete=false` even if the server is older.
- Binance public feed is a 1s largest-hit sample, not every fill.
