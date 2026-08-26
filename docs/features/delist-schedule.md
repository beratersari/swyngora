# Spot delist schedule (all venues)

## Goal

Show pairs that an exchange **will delist in the next month** or **already delisted in the last 30 days** on that venue’s Markets list (and Coin Detail) with an amber **Delist** tag and date.

## Behavior

1. Backend refreshes a per-venue in-memory schedule every **1 hour** (and once on startup).
2. **Binance** — official `GET /sapi/v1/spot/delist-schedule` when `BINANCE_API_KEY` is set, plus public CMS “Will Delist TOKEN on DATE” titles so last-30-day full-token removals stay visible after they leave the official schedule.
3. **Bybit** — public `GET /v5/announcements/index?type=delistings` (no key). List rows often have empty descriptions, so the job fetches the announcement HTML (capped) and parses the spot halt date (“will end after …”) plus pair list. Perp-only, Alpha, and lending notices are skipped.
4. **Coinbase / Nasdaq / BIST** — no free upcoming calendar; schedule is empty (`enabled: false`).
5. Spot rows get `delistTime` (halt), `announcedAt` (when the venue published the notice — Binance CMS `releaseDate`, Bybit `publishTime`), and a synthetic `Delist` tag. The tag shows both dates when `announcedAt` is known.
6. Pairs that delist in the next **31 days** or in the last **30 days** stay on the default `status=TRADING` list even if the venue already marked them `BREAK`. Missing book rows are injected as stubs so they still appear.
7. Stubs with no live tape are filled from the venue’s last kline at halt (last / high / low / volume). Live book prices are not overwritten. After the halt clock, the home venue’s frozen 24h **change and %** are replaced with the same off-venue tape coin detail uses (another listed venue, else CoinGecko `price_change_24h` + `price_change_percentage_24h`). Coin-detail candles keep leftover venue klines after a midnight schedule (halt + 24h fetch). They do **not** invent flat bars after the last print. After halt the 24h ticker is marked `halted`. Market cap uses the Binance supply snapshot when present; otherwise CoinGecko public `/coins/markets` (exact ticker, delist rows only).
8. After the official halt, `GET /api/v1/market/post-delist` returns labeled off-venue movement at the chart interval: another listed venue if the pair still trades there, otherwise CoinGecko USD last + OHLC. Coin detail **appends those bars on the single main chart** after the last home-venue print and shows an After delist note. Informational only — not this venue’s book. There is no second chart.
9. `GET /api/v1/market/delist-schedule?exchange=` returns that venue’s cache. `enabled` is per venue.
10. UI: Markets banner + Tags column; Coin Detail header tag plus an **After delist** panel. Filter tag **Delist (30 days)** when the venue has rows.

## Code

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/delist.go`, `post_delist.go` |
| Store | `backend/internal/adapter/deliststore/` |
| Binance fetch | `backend/internal/adapter/binance/delist.go` |
| Bybit fetch | `backend/internal/adapter/bybit/delist.go` |
| Job | `backend/internal/service/delistjob/` (one runner per source) |
| Enrichment | `enrichDelistTimes`, `injectUpcomingDelists` |
| HTTP | `GET /api/v1/market/delist-schedule` + `SpotMarket.delistTime` / `announcedAt`; `GET /api/v1/market/post-delist` |
| MCP | `list_delist_schedule`, `get_post_delist` |
| UI | `MarketsPage`, `MarketsTable` / `SpotMetricValue`, `DetailHeader`, coin-detail chart vertical lines (announced + halt; snapped to the **containing** bar for the active interval — 1m/5m/1h/1d/… — not the nearest open), `PostDelistPanel` |

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
curl -sS 'http://127.0.0.1:8080/api/v1/market/post-delist?exchange=binance&symbol=VICUSDT'
```

## Limits

- Coinbase, Nasdaq, and BIST have no public future delist calendar.
- Bybit dates/symbols come from announcement text and article HTML; a notice without a parseable date is skipped.
- Not a substitute for official announcements on edge cases.
