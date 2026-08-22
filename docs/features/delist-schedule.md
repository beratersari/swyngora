# Spot delist schedule (all venues)

## Goal

Show pairs that an exchange **will delist in the next month** or **already delisted in the last 30 days** on that venue’s Markets list (and Coin Detail) with an amber **Delist** tag and date.

## Behavior

1. Backend refreshes a per-venue in-memory schedule every **1 hour** (and once on startup).
2. **Binance** — official `GET /sapi/v1/spot/delist-schedule` when `BINANCE_API_KEY` is set, plus public CMS “Will Delist TOKEN on DATE” titles so last-30-day full-token removals stay visible after they leave the official schedule.
3. **Bybit** — public `GET /v5/announcements/index?type=delistings` (no key). List rows often have empty descriptions, so the job fetches the announcement HTML (capped) and parses the spot halt date (“will end after …”) plus pair list. Perp-only, Alpha, and lending notices are skipped.
4. **Coinbase / Nasdaq / BIST** — no free upcoming calendar; schedule is empty (`enabled: false`).
5. Spot rows get `delistTime` + a synthetic `Delist` tag.
6. Pairs that delist in the next **31 days** or in the last **30 days** stay on the default `status=TRADING` list even if the venue already marked them `BREAK`. Missing book rows are injected as stubs so they still appear.
7. `GET /api/v1/market/delist-schedule?exchange=` returns that venue’s cache. `enabled` is per venue.
8. UI: Markets banner + Tags column; Coin Detail header tag. Filter tag **Delist (30 days)** when the venue has rows.

## Code

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/delist.go` |
| Store | `backend/internal/adapter/deliststore/` |
| Binance fetch | `backend/internal/adapter/binance/delist.go` |
| Bybit fetch | `backend/internal/adapter/bybit/delist.go` |
| Job | `backend/internal/service/delistjob/` (one runner per source) |
| Enrichment | `enrichDelistTimes`, `injectUpcomingDelists` |
| HTTP | `GET /api/v1/market/delist-schedule` + `SpotMarket.delistTime` |
| MCP | `list_delist_schedule` |
| UI | `MarketsPage`, `MarketsTable` / `SpotMetricValue`, `DetailHeader` |

## Config

| Env | Default | Purpose |
|---|---|---|
| `BINANCE_API_KEY` | empty | Required for Binance official schedule |
| `DELIST_REFRESH_EVERY` | `1h` | Poll interval (all sources) |
| `DELIST_REFRESH_ON_STARTUP` | `true` | Fetch once after boot |

## Verify

```bash
curl -sS 'http://127.0.0.1:8080/api/v1/market/delist-schedule?exchange=binance' | head
curl -sS 'http://127.0.0.1:8080/api/v1/market/delist-schedule?exchange=bybit' | head
curl -sS 'http://127.0.0.1:8080/api/v1/market/spot?exchange=binance&tag=Delist&limit=20'
curl -sS 'http://127.0.0.1:8080/api/v1/market/spot?exchange=bybit&tag=Delist&limit=20'
```

## Limits

- Coinbase, Nasdaq, and BIST have no public future delist calendar.
- Bybit dates/symbols come from announcement text and article HTML; a notice without a parseable date is skipped.
- Not a substitute for official announcements on edge cases.
