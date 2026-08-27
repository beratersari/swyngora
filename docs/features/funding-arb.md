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
- Lists a run only when settled funding minus round-trip fees is positive
- `from` / `to`: RFC3339, `YYYY-MM-DD` (UTC), or unix ms; max 30 days

This is **not** an executable arb and **not** financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/funding_arb.go` |
| Service | `backend/internal/service/market/funding_arb.go` |
| HTTP | `GET /api/v1/market/funding-arb`, `/scan`, `/history` |
| MCP / AI | `get_funding_arb`, `scan_funding_arb`, `get_funding_arb_history` |
| Telegram | `/fundingarb`, `/fundingarb scan`, `/fundingarb hist <sym> <from> <to>` |

## How to verify

```bash
cd backend
go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ ./internal/transport/mcp/ ./internal/transport/telegram/ -count=1 -run FundingArb
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
