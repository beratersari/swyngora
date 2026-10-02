# Futures long/short ratio

## Goal

Show the **current long/short account ratio** and **recent 5-minute history**
for a coin on Binance USD-M, Bybit linear, and OKX USDT-swap, so a user or the AI
can ask “are more traders long or short BTC?” On Binance and OKX, also show
**top-trader account** and **top-trader position** ratios, a **single gap**
between big traders and the crowd (top long% minus crowd long%), and whether
that gap is **widening or shrinking**. Show when **top traders and price
diverge** (top getting more long while price drops, or more short while
price rises), and **when that run started**. Say when the crowd and bigger
books lean opposite.

## Behavior

- `GET /api/v1/market/long-short-ratio?symbol=BTCUSDT`
  - `exchange=all` (default) returns **each venue separately** (not averaged)
  - `exchange=binance`, `bybit`, or `okx` hoists `current` + `history` to the top level
  - `limit` 5m prints (default 24 ≈ 2 hours, max 100)
- `GET /api/v1/market/open-interest` includes the same payload on `longShort`
- `current` is the **share of accounts** that are long vs short, **not** position size
- On Binance, `topAccounts` is the top 20% of traders by margin (account count)
  and `topPositions` is the same traders by **position size**. On OKX both
  layers are the top 5% of traders by position value
- `gap` is one signed number: **top long% − crowd long%** in percentage points.
  Top 65% long vs crowd 55% long is `+10` (`lean=top_more_long`). Size
  (`top_positions`) is preferred over account count when both exist. Bybit has
  no public top-trader feed, so no gap there. OKX has the same gap once its
  top-position ratio is present
- `gapWindows` (1h / 4h / 24h) compare that gap to ~window ago:
  `widened` / `narrowed` / `flipped` / `stable`. `gapHistory` is 5m samples
  (newest first). `gapMoves` calls out larger jumps. 24h uses stored futures
  history when the live REST window is shorter
- `divergence` is when top-trader long share and 5m price move opposite ways
  for at least 15 minutes (max 4h). `kind` is `top_long_price_down` (top
  getting more long, price down) or `top_short_price_up`. `since` is when
  the current run started; `duration` / `priceChangePct` / `topChangePct`
  cover that run (e.g. +3pp long vs −1.5% over 30m). Price is the venue's
  5m close. `gapHistory[].vsPrice` is `opposite` or `confirms` per bar
- `split.opposite` means all-account crowding and top-trader size lean different
  ways. Hunt, squeeze, and positioning then treat **size** as who is positioned
  and keep the crowd print visible
- `ratio` is long/short (1 = even). `bias` is `long` if ratio ≥ 1.05, `short` if ≤ 0.95
- `change` on each venue is the signed ratio move vs the previous 5m print
- Durable 5m samples are also stored in SQLite. Query them with
  `GET /api/v1/market/futures-history?metric=long_short`. See [`futures-history.md`](futures-history.md).

Sources (public REST, no API key):

- Binance: `GET /futures/data/globalLongShortAccountRatio?period=5m`
  plus `topLongShortAccountRatio` and `topLongShortPositionRatio`
- Bybit: `GET /v5/market/account-ratio?category=linear&period=5min` (`buyRatio` / `sellRatio`)

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/longshort.go`, `longshort_gap.go`, `longshort_price.go` |
| Adapters | `adapter/binance/longshort.go`, `adapter/bybit/longshort.go` |
| Service | `GetLongShortRatio` (also attached by `GetOpenInterest`) |
| HTTP | `GET /api/v1/market/long-short-ratio` |
| MCP / AI | `get_long_short_ratio` (also on `get_open_interest.longShort`) |
| Telegram | `/ls <symbol>` and a Long/short block on `/oi` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ ./internal/transport/http/handler/ ./internal/transport/mcp/
curl "http://localhost:8080/api/v1/market/long-short-ratio?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/open-interest?symbol=BTCUSDT"
```

## Limits

- Not Coinbase. Not COIN-M / inverse.
- Top-trader layers and the crowd-vs-top gap are Binance-only (Bybit has no public equivalent).
- Combined `all` does not invent a blended ratio or a blended gap (Binance gap is hoisted when present).
- Informational only — not financial advice.
