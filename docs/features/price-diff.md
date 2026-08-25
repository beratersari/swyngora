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
11. **Executable quote:** walk the **buy venue asks** and **sell venue bids** for a size. `notional` is quote currency spent on the buy book **before** the buy fee (e.g. `10000` USDT). The same base quantity is sold on the other book (rematched if one side is thinner). Response includes average buy/sell, slippage vs top of book, profit after the watch fees, and **max size** — the largest quantity where every increment still adds after-fee profit (`maxLimitedBy` is `profit`, `buy_book`, `sell_book`, `both_books`, or `empty_book`). `executable` is true only when the full requested size fills on both books **and** profit after fees is positive. Simulated depth — not a fill.

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

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/pricediff.go`, `pricediff_quote.go` |
| Store | `backend/internal/adapter/pricediffstore` |
| Service | `backend/internal/service/pricediff` |
| Worker | `backend/internal/service/pricediff/checker.go` |
| HTTP | `backend/internal/transport/http/handler/pricediff.go` |
| MCP | `create_price_diff_watch`, `list_price_diff_watches`, `get_price_diff_watch`, `delete_price_diff_watch`, `list_price_diff_opportunities`, `get_price_diff_opportunity`, `quote_price_diff`, `quote_price_diff_opportunity` |

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
- Max size stops at the first increment whose after-fee edge is no longer positive (does not keep filling until cumulative profit hits zero).
- Coinbase quotes USD (not USDT); USD≈USDT is assumed for comparison.
- Max 20 watches per client.
- No webhook on opportunity open yet.
