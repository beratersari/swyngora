# Binance spot delist schedule

## Goal

Show when a Binance spot pair is **scheduled to delist**, as a warning tag on Markets and Coin Detail.

## Behavior

1. Backend (when `BINANCE_API_KEY` is set) polls  
   `GET https://api.binance.com/sapi/v1/spot/delist-schedule` every **1 hour** (and once on startup).
2. Results are stored in memory (`deliststore`) keyed by symbol.
3. Spot list items include optional `delistTime` (RFC3339 UTC).
4. `GET /api/v1/market/delist-schedule?exchange=binance` returns the full cached schedule.
5. UI shows a **Delist** brand tag with the date (e.g. `Delist 17 Aug 2026 03:00 UTC`).

Without `BINANCE_API_KEY`, refresh is disabled and tags stay empty (API still returns `items: []`).

## Code

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/delist.go` |
| Store | `backend/internal/adapter/deliststore/` |
| Binance fetch | `backend/internal/adapter/binance/delist.go` |
| Job | `backend/internal/service/delistjob/` |
| Enrichment | `market.Service.enrichDelistTimes` |
| HTTP | `GET /api/v1/market/delist-schedule` + `SpotMarket.delistTime` |
| MCP | `list_delist_schedule` |
| UI | `MarketsTable`, `DetailHeader` (`BrandTag` variant `delist`) |

## Config

| Env | Default | Purpose |
|---|---|---|
| `BINANCE_API_KEY` | empty | Required for schedule fetch |
| `DELIST_REFRESH_EVERY` | `1h` | Poll interval |
| `DELIST_REFRESH_ON_STARTUP` | `true` | Fetch once after boot |

## Verify

```bash
# backend with key in backend/.env
curl -sS 'http://127.0.0.1:8080/api/v1/market/delist-schedule?exchange=binance' | head
curl -sS 'http://127.0.0.1:8080/api/v1/market/spot?exchange=binance&q=ACX&limit=5'
```

## Limits

- Binance only (schedule API is venue-specific).
- Requires a Binance API key (MARKET_DATA wallet endpoint).
- Not a substitute for reading official announcements for edge cases.
