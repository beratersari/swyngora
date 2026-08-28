# Cross-exchange funding opportunities

## Goal

Find **real, sized** trades from the **funding-rate gap** between Binance USD-M
and Bybit linear perpetual. Show which venue to **long**, which to **short**,
fold in **taker fees** and each venue's **spot vs perpetual** price, then say
about how much the entered size could earn after the next funding and over a
hold window.

## Behavior

Funding is **not** prorated by the hour. Binance and Bybit pay only if you still
hold at a published settlement:

| Interval | Clocks (UTC) |
|---|---|
| 8h | 00:00, 08:00, 16:00 |
| 4h | 00:00, 04:00, 08:00, 12:00, 16:00, 20:00 |
| 1h | on the hour |

Fee at a clock: `notional × rate` (long pays when the rate is positive; short
receives). A hold window with **no** clock in `(now, now+holdHours]` has
**funding 0**.

`GET /api/v1/market/funding-arb?symbol=BTCUSDT&notional=10000`

- Long the cheaper predicted rate, short the richer one
- Walks each venue's `nextFundingTime` plus its interval
- Later clocks in the window use the current predicted rate (the only future
  rate the venue publishes)
- `trade` is omitted unless after-fee net is positive
- Fees default to paper taker 0.10%; override with `feeBinancePct` / `feeBybitPct`

`GET /api/v1/market/funding-arb/scan` — same math; **only after-fee winners**

`GET /api/v1/market/funding-arb/history?symbol=BTCUSDT&from=2026-08-01&to=2026-08-08`

- Settled prints from Binance `/fapi/v1/fundingRate` and Bybit `/v5/market/funding/history`
- Groups consecutive prints with the same long/short pair
- The **first clock of a run is the entry signal** — that payment is not
  collected (the position was not open before it). A stretch that starts and
  ends at the same clock (e.g. 08:00–08:00) is not profit.
- When long/short **flips at a settlement**, that clock still pays the **old**
  sides (the position was already open), then the run ends and the new
  direction starts as an entry signal.
- Lists a run only when later settled funding minus round-trip fees is positive
- `from` / `to`: RFC3339, `YYYY-MM-DD` (UTC), or unix ms; max 30 days

`POST /api/v1/funding-arb/watches` — omit `symbol` (or set `scan` / `*`) to
follow the **funding-arb scan** (one scan follow per client). The checker
re-scans on `FUNDING_ARB_CHECK_INTERVAL` (default 30s) and notifies when a
**new coin** first has after-fee net ≥ `minProfit`. Same coin + same
long/short while still above the floor is not notified again. After it
drops below the floor the signal closes; a later re-cross notifies again.
A **direction flip** closes the old signal and opens a new one. Optional
`symbol` still follows one pair. Max 20 watches per client.

This is **not** an executable arb and **not** financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/funding_arb.go`, `funding_arb_history.go`, `funding_arb_watch.go` |
| Quote | `backend/internal/service/market/funding_arb.go` |
| Watches | `backend/internal/service/fundingarb/`, `backend/internal/adapter/fundingarbstore/` |
| HTTP | `GET /api/v1/market/funding-arb`, `/scan`, `/history`; `POST /api/v1/funding-arb/watches` (omit `symbol` = scan follow) |
| MCP / AI | `get_funding_arb`, `scan_funding_arb`, `get_funding_arb_history`, `create_funding_arb_watch`, `list_funding_arb_watches`, `get_funding_arb_watch`, `delete_funding_arb_watch`, `list_funding_arb_signals` |
| Telegram | `/fundingarb`, `/fundingarb scan`, `/fundingarb hist <sym> <from> <to>` |

## How to verify

```bash
cd backend
go test ./internal/domain/ ./internal/service/market/ ./internal/service/fundingarb/ ./internal/adapter/fundingarbstore/ ./internal/transport/http/handler/ ./internal/transport/mcp/ ./internal/transport/telegram/ -count=1 -run 'FundingArb|WatchAndSignal'
curl "http://localhost:8080/api/v1/market/funding-arb?symbol=BTCUSDT&notional=10000"
curl "http://localhost:8080/api/v1/market/funding-arb/scan?notional=10000&limit=10"
curl "http://localhost:8080/api/v1/market/funding-arb/history?symbol=BTCUSDT&from=2026-08-01&to=2026-08-08&notional=10000"
```

## Limits

- Binance USD-M and Bybit linear only (not Coinbase, not inverse)
- Live quote uses the **predicted next** rate on each future clock; history uses settled prints only
- Default 0.10% taker fees often need several settlements to break even
- Visible last/mark vs spot only — no live book walk for the perp legs
- Informational only
