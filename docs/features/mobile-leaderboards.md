# Feature: Gainers / losers / high-volume leaderboards (mobile)

**Status:** Implemented  
**Surface:** Product mobile (`mobile/`) — Chrome via react-native-web  
**Backend:** Existing spot list sorts only  
**Epic:** `project-management/epics/mobile-leaderboards.md`  
**Design:** `docs/design/mobile-leaderboards.md`  
**Tasks:** `project-management/tasks/mobile/leaderboards/MLEAD-*.md`

---

## 1. Problem / goal

Expose **full ranked leaderboards** for 24h gainers, 24h losers, and highest quote volume — beyond Home teasers of 5 rows.

---

## 2. Behavior (happy path)

1. Open **Leaderboards** (from Home “See all” or Markets entry).  
2. Switch **Gainers | Losers | Volume**.  
3. Optionally change **exchange** and **quote**.  
4. Scroll for more rows; pull to refresh.  
5. Tap a row → coin detail.  

### Limits

- Rankings use server spot sort (`priceChangePercent`, `quoteVolume`) for `TRADING` pairs.  
- Informational only — not financial advice.  
- Coinbase default quote may be USD (align with Markets/Coinbase norms).

---

## 3. APIs

| Method | Path | Use |
|--------|------|-----|
| `GET` | `/api/v1/market/spot` | Sorted/paged leaderboards |
| `GET` | `/api/v1/market/exchanges` | Exchange chips (optional) |
| `POST` | `/api/v1/market/indicators/batch` | Optional RSI on rows |

No new OpenAPI. MCP already exposes `list_spot_markets`.

---

## 4. Code homes (planned)

| Area | Path |
|------|------|
| Page | `mobile/src/modules/app/pages/leaderboards-page/` or `modules/markets/pages/leaderboards-page/` |
| Helpers | `mobile/src/libs/utils/leaderboardQuery.ts` |
| Constants | `mobile/src/config/leaderboardConstants.ts` |
| UI | Reuse markets list/row + small segment control |

---

## 5. How to verify

```bash
cd backend && go run ./cmd/server
cd mobile && npm run web
# Home → See all movers → Gainers board
# Switch Losers / Volume; change exchange
```

---

## 6. Known limitations / follow-ups

- No multi-interval gainers (only 24h metrics from ticker join).  
- Losers may include low-liquidity pairs; optional min volume filter later.  
- Trade-count board is a one-line sort extension if product wants it.
