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

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/around.go`, `around_compare.go` |
| Service | `backend/internal/service/market/around.go` |
| HTTP | `GET /api/v1/market/around`, `.../around/compare` |
| MCP / AI | `get_around`, `compare_around` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ -run Around
curl "http://localhost:8080/api/v1/market/around?symbol=BTCUSDT&at=2026-08-20T14:00:00Z"
curl "http://localhost:8080/api/v1/market/around/compare?symbol=BTCUSDT&from=2026-08-20T12:00:00Z&to=2026-08-20T16:00:00Z"
```
