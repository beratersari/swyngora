# Cross-exchange funding opportunities

## Goal

Find **real, sized** trades from the **funding-rate gap** between Binance USD-M
and Bybit linear perpetual. Show which venue to **long**, which to **short**,
fold in **taker fees** and each venue's **spot vs perpetual** price, then say
about how much the entered size could earn after the next funding and over a
hold window.

## Behavior

`GET /api/v1/market/funding-arb?symbol=BTCUSDT&notional=10000`

- Long the venue with the **cheaper** predicted funding, short the **richer** one
- Positive funding: longs pay shorts. Collect `shortRate − longRate` per settlement
- `notional` is the quote size held on **each** leg (default 10000)
- `holdHours` (default 24) is how long the horizon payout assumes you hold
- Fees default to paper taker (0.10% Binance, 0.10% Bybit). Override with
  `feeBinancePct` / `feeBybitPct` (percent; `0` means zero fee)
- Round-trip = open **and** close both legs (four taker fills)
- Each venue shows perp last/mark vs spot/index (`basisPct`)
- `perpGapPct` is short perp minus long perp (shorting the expensive book is extra
  if they meet)
- `netHorizonIfBasisConverges` adds that spot-perp gap if **both** perps move back
  to their own spot — **not guaranteed**
- `worthIt` is true only when predicted funding over `holdHours` covers
  **round-trip** fees
- `breakEvenSettlements` / `breakEvenHoldHours` say how long the same spread
  must last to pay those fees
- `carry` is the same-venue alternative (long spot / short perp when funding is
  positive, or the reverse)

`GET /api/v1/market/funding-arb/scan?notional=10000&limit=15`

- Walks top 24h-volume USDT pairs on Binance that also have Bybit funding
- Ranks by `netHorizonAfterRoundTrip`

This is **not** an executable arb and **not** financial advice. Predicted rates
can change before settlement.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/funding_arb.go` |
| Service | `backend/internal/service/market/funding_arb.go` |
| HTTP | `GET /api/v1/market/funding-arb`, `GET /api/v1/market/funding-arb/scan` |
| MCP / AI | `get_funding_arb`, `scan_funding_arb` |
| Telegram | `/fundingarb <symbol> [notional] [holdHours]`, `/fundingarb scan [notional]` |

## How to verify

```bash
cd backend
go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ ./internal/transport/mcp/ ./internal/transport/telegram/ -count=1 -run FundingArb
curl "http://localhost:8080/api/v1/market/funding-arb?symbol=BTCUSDT&notional=10000"
curl "http://localhost:8080/api/v1/market/funding-arb/scan?notional=10000&limit=10"
```

## Limits

- Binance USD-M and Bybit linear only (not Coinbase, not inverse)
- Uses the **predicted next** rate, not a locked payment
- Default 0.10% taker fees often need several settlements to break even
- Visible last/mark vs spot only — no live book walk for the perp legs
- Informational only
