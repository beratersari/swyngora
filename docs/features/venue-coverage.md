# Feature: Market-wide venue coverage

## Problem / goal

A combined Binance + Bybit + OKX total can look like the whole market when one venue never answered or its last sample is old. The desk should see which venue is missing, which venue is old, and should not get a market lean from that mix.

## Behavior

`coverage` is attached to these reads when they are asked for `exchange=all` or for one futures venue:

- `GET /api/v1/market/order-count/overview`
- `GET /api/v1/market/print-size/overview`
- `GET /api/v1/market/liquidations/overview`
- `GET /api/v1/market/perp-depth/overview`

Each expected venue is `fresh`, `stale`, or `missing`.

- **Missing** — the venue did not answer (request error, feed never seen, or no book).
- **Old** — the venue answered, and the newest print or book is older than the budget (tape 2 minutes, perp book 45 seconds, liquidation socket 2 minutes). A tape that answered with no prints is fresh, not old.
- **Fresh** — the sample is inside that budget.

`coverage.complete` and `definite` are true only when every expected venue is fresh. A single-venue request only has to cover that venue. When `definite` is false the summary says which venues are missing and which are old, and the order-count lean, print-size read, and perp bid/ask lean are cleared. Liquidation totals stay in the payload for inspection, and the Liquidations page does not present them as the market.

Perp depth history stores a minute only when `definite` is true. A missing or old venue is left out of that minute, so the six-hour total and the thicker/thinner windows do not treat the gap as a real change in book size.

The same fields are on the matching MCP tools.

## Where the code lives

- `backend/internal/domain/venue_coverage.go`
- Order count, print size, perp depth, and liquidation overview services and HTTP DTOs
- `frontend/src/components/molecules/VenueCoverage`
- Order count page, print size page, liquidations overview, and the Markets perp depth card

## How to test

```bash
cd backend
go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ ./internal/transport/mcp/
```

```bash
cd frontend
npm test -- src/components/molecules/VenueCoverage/VenueCoverage.test.tsx
```

Open Order count, Print size, Liquidations, or Markets. If OKX or another venue is down or quiet past the budget, the screen lists that venue and does not show a market lean.

## Limits

Other combined reads (funding, open interest, taker flow, fear and greed) do not use this coverage yet. Spot books (Coinbase) are not part of this futures set.
