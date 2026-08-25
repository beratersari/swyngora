# Feature: Cross-exchange price difference tracking

## Problem / goal

Users want to know when the same coin trades at a meaningful price gap across **Binance**, **Coinbase**, and **Bybit** after fees. The system should open durable opportunities (buy venue / sell venue) without flooding duplicates, and re-open only after the edge has gone away and returns.

## Behavior

1. **Create a watch**: coin (`symbol`, e.g. `BTCUSDT`), `minNetDiffPct`, and per-exchange fees (`feeBinancePct`, `feeCoinbasePct`, `feeBybitPct`).
2. Background worker fetches last prices on all three venues (Coinbase maps `*USDT` → `*-USD`).
3. **Net edge** for buy A / sell B:

   `netPct = (sell * (1 - feeSell/100) / (buy * (1 + feeBuy/100)) - 1) * 100`

4. If `netPct >= minNetDiffPct`, open an **opportunity** with `buyExchange` and `sellExchange`.
5. **While open**: same `(watch, buy, sell)` is only updated (prices / lastSeenAt), not recreated.
6. When net falls **below** the limit (with fresh prices on both legs): opportunity is **closed**.
7. When edge later exceeds the limit again: a **new** opportunity is created.
8. If a venue price is **missing or stale** (CloseTime older than 2 minutes, or **zero/unknown**): that venue is skipped for this tick; incomplete data does **not** open or close a route. Binance uses API `closeTime`; Bybit uses ticker `time`; Coinbase uses Exchange `/ticker` trade time. Adapters must not stamp `time.Now()` as CloseTime.
9. `minNetDiffPct` floor is **0.20%** so USDT≈USD noise does not open false opportunities.
10. Open opportunities live in SQLite and **survive worker restarts**.
11. **Executable quote:** walk the **buy venue asks** and **sell venue bids** for a size. `notional` is quote currency spent on the buy book **before** the buy fee (e.g. `10000` USDT). The same base quantity is sold on the other book (rematched if one side is thinner). **Max size** is the largest quantity whose **cumulative** after-fee profit is still positive — a later book level that loses money on its own is still taken while the running total stays above zero. The displayed fill is capped at that size. `usedNotional` / `usedPct` is how much of the entered money can actually be deployed; `unusedNotional` is the rest. `executable` is true only when the full requested size fills **and** profit after fees is positive.
12. **All-venue scan:** one amount is walked on every Binance / Coinbase / Bybit buy→sell pair (fees + live depth). Routes are ranked by after-fee profit, then by usable money. `bestRoute` is the top row. If a venue's order book cannot be loaded, that venue is listed in `unavailable` and is **never** shown as a normal route or chosen as best. Optional `minProfitPct` and/or `minProfitAmount` hide smaller fills (`skippedCount` is how many were dropped). Both filters must pass when both are set.

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/price-diff/watches` | Create watch |
| `GET` | `/api/v1/price-diff/watches` | List watches |
| `GET` | `/api/v1/price-diff/watches/{id}` | Get watch |
| `DELETE` | `/api/v1/price-diff/watches/{id}` | Delete watch (+ opportunities) |
| `GET` | `/api/v1/price-diff/opportunities?status=open\|closed\|all` | List opportunities |
| `GET` | `/api/v1/price-diff/opportunities/{id}` | Get opportunity |
| `GET` | `/api/v1/price-diff/opportunities/{id}/quote?notional=10000` | Executable quote for a stored opportunity (uses watch fees) |
| `GET` | `/api/v1/price-diff/quote?symbol=BTCUSDT&buyExchange=binance&sellExchange=bybit&notional=10000&feeBuyPct=0.1&feeSellPct=0.1` | Executable quote without a stored opportunity |
| `GET` | `/api/v1/price-diff/quote/scan?symbol=BTCUSDT&notional=10000&feeBinancePct=0.1&feeCoinbasePct=0.6&feeBybitPct=0.1&minProfitPct=0.5` | Rank every venue pair at one size |
| `GET` | `/api/v1/price-diff/watches/{id}/quote?notional=10000` | Same scan using a watch's symbol and fees |

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/pricediff.go`, `pricediff_quote.go` |
| Store | `backend/internal/adapter/pricediffstore` |
| Service | `backend/internal/service/pricediff` |
| Worker | `backend/internal/service/pricediff/checker.go` |
| HTTP | `backend/internal/transport/http/handler/pricediff.go` |
| MCP | `create_price_diff_watch`, `list_price_diff_watches`, `get_price_diff_watch`, `delete_price_diff_watch`, `list_price_diff_opportunities`, `get_price_diff_opportunity`, `quote_price_diff`, `quote_price_diff_opportunity`, `scan_price_diff_quotes`, `quote_price_diff_watch` |

## Config

| Env | Default |
|-----|---------|
| `PRICE_DIFF_DB_PATH` | `data/pricediff.db` |
| `PRICE_DIFF_CHECK_INTERVAL` | `30s` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/pricediff/ ./internal/transport/http/handler/ -run 'PriceDiff|NetDiff|Fresh|Opportunity|Skip|Persist|Quote' -count=1
```

## Limitations

- Informational only; not executable arb or financial advice. Quotes walk **visible** resting depth only.
- Max size includes later losing book levels as long as the **total** after-fee profit stays positive.
- Coinbase quotes USD (not USDT); USD≈USDT is assumed for comparison.
- Max 20 watches per client.
- No webhook on opportunity open yet.
