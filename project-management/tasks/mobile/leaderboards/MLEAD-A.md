# MLEAD-A: Spot sort field matrix + leaderboard UX (analysis)

| Field | Value |
|---|---|
| **ID** | MLEAD-A |
| **Epic** | mobile-leaderboards |
| **Status** | done |
| **Area** | mobile / analysis |
| **Path** | `project-management/tasks/mobile/leaderboards/MLEAD-A.md` |

## Purpose

Map existing spot list sorts to gainers / losers / volume boards. **No backend work** — API exists.

Sources:

- OpenAPI: `listSpotMarkets` (`sort`, `order`, `quote`, `exchange`, pagination)
- Mobile: `homeDashboardQuery` movers/volume builders, Markets list pagination
- Feature: `docs/features/market-data.md`

---

## 1. Endpoint

| Method | Path | operationId |
|--------|------|-------------|
| `GET` | `/api/v1/market/spot` | `listSpotMarkets` |

---

## 2. Board → query matrix

| Board | sort | order | default quote | notes |
|-------|------|-------|---------------|-------|
| Gainers | `priceChangePercent` | `desc` | USDT | Home movers already uses this |
| Losers | `priceChangePercent` | `asc` | USDT | Not on Home today |
| High volume | `quoteVolume` | `desc` | USDT | Home volume teaser |

Common: `status=TRADING`, `limit`/`offset`, `exchange`.

Optional later: `sort=tradeCount` (Binance-meaningful only).

---

## 3. Fields for rows

Reuse Markets / Home mapping: symbol, lastPrice, priceChangePercent, quoteVolume, tags (optional), mcap (optional).  
Add **rank** as client-side `offset + index + 1`.

---

## 4. Mobile gap

| Capability | Today | Epic |
|------------|-------|------|
| Top 5 movers on Home | ✅ | Deep link to full gainers |
| Top 5 volume on Home | ✅ | Deep link to full volume |
| Full losers board | ❌ | New |
| Board page + pagination | ❌ | New |
| Exchange/quote on board | partial (Home fixed) | Chips on board |

---

## 5. Decisions

1. **Single LeaderboardsPage** with segment control (not three routes).  
2. Route param `board` for deep links.  
3. Share query builders with Home (extract common util).  
4. Batch RSI on boards is **optional P1** — implement if Markets batch pattern is cheap to reuse.  
5. No min-volume filter in v1.

---

## Acceptance

- [x] Matrix complete; linked from design/epic  
- [x] Confirmed Home builders can be shared without behavior break  
- [x] Status → done when analysis accepted  

## Status

`done`
