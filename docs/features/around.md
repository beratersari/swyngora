# Around a time (before / during / after)

## Goal

Give a coin and a **specific time** and see what happened **before**,
**during**, and **after** that move — one tape that brings historical
candles, flow, and stored book/futures history together and says what
changed.

## Behavior

`GET /api/v1/market/around?symbol=BTCUSDT&at=2026-08-20T14:00:00Z`

Three adjacent windows:

| Phase | Default range |
|---|---|
| **before** | `window` ending at `at` (default 1h) |
| **during** | the move starting at `at` (default 15m; override with `during`) |
| **after** | the same `window` after the move |

Each phase includes:

- price: open / high / low / close, net %, range
- quote volume, buy/sell when the venue publishes taker-buy, VWAP
- volume versus that coin's own typical (median of prior same-length windows)
- compact volume profile (POC + 70% value area)
- absorption on the window when the score is at least moderate
- stored spot-book compare (when order-book history has samples)
- stored futures: open interest, funding, long %, liquidations (when the archive has samples)

Liquidity sweeps whose poke sits in the span are attached as events.
`changes` compares the three windows (continued / reversed / faded).

`exchange=all` (default) returns Binance and Bybit separately plus
`combined` (volume-weighted). `at` is RFC3339 or unix milliseconds,
and must be within the last 30 days. If the look-forward is still in
the future, `after` (and sometimes `during`) is clipped to now.

Informational only — not financial advice.

## Compare two times

`GET /api/v1/market/around/compare?symbol=BTCUSDT&from=2026-08-20T12:00:00Z&to=2026-08-20T16:00:00Z`

Same `window` / `during` / `exchange` as `/around`. Builds both tapes and
diffs:

- **state** at the two event times: price level, stored book mid, OI,
  funding, long %
- **during** (and before/after): net %, range, volume, vs typical, taker
  delta, POC, book bid/ask change, OI change, liquidations, sweep /
  absorption counts
- stored **order book at from vs to** when the archive has samples
- `fromMove` / `toMove`: the full around tapes

`from` and `to` do not have to be in chronological order. MCP:
`compare_around`.

## Important moves

`GET /api/v1/market/around/moves?symbol=BTCUSDT`

Walks recent 15m (or 1h) candles, groups same-direction legs, and keeps
the strongest up and down stretches (default floor 1.5% on 15m / 2.5%
on 1h, last 24h, up to 8 moves). Each hit includes the around-the-move
tape so you see volume, VWAP, book, OI, and sweeps during that move.

`lookback` `4h` / `12h` / `24h` / `3d` / `7d`. `direction` `up` /
`down` / `both`. MCP: `find_around_moves`.

## Common changes before moves

`GET /api/v1/market/around/precursors?symbol=BTCUSDT`

Same scan as `/around/moves`, then compares the **before** window of
each move and lists conditions that show up often:

- price already quiet / rising / falling
- volume elevated vs typical
- takers buying or selling
- open interest rising or falling
- bid or ask liquidity pulled
- a sweep or absorption in the before-window

`common` is **60%+** of those before-windows with **at least 3**
samples. Default lookback **7d** (more history than `/moves`).

`combos` groups conditions that fired **together** in the same
before-window (for example elevated volume + bid pulled + OI rising).
Each combo reports how often it showed up before **increases** vs
**drops** (`lean`: `up` / `down` / `both`). MCP:
`find_around_precursors`.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/around.go`, `around_compare.go`, `around_moves.go`, `around_precursors.go` |
| Service | `backend/internal/service/market/around.go`, `around_moves.go` |
| HTTP | `GET /api/v1/market/around`, `.../compare`, `.../moves`, `.../precursors` |
| MCP / AI | `get_around`, `compare_around`, `find_around_moves`, `find_around_precursors` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ -run Around
curl "http://localhost:8080/api/v1/market/around?symbol=BTCUSDT&at=2026-08-20T14:00:00Z"
curl "http://localhost:8080/api/v1/market/around/compare?symbol=BTCUSDT&from=2026-08-20T12:00:00Z&to=2026-08-20T16:00:00Z"
curl "http://localhost:8080/api/v1/market/around/moves?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/around/precursors?symbol=BTCUSDT"
```
