# Feature: Cross-exchange coin comparison (mobile)

**Status:** Implemented  
**Surface:** Product mobile (`mobile/`) — Chrome via react-native-web  
**Backend:** Existing multi-exchange ticker (compose client-side)  
**Epic:** `project-management/epics/mobile-cross-exchange-compare.md`  
**Design:** `docs/design/mobile-cross-exchange-compare.md`  
**Tasks:** `project-management/tasks/mobile/cross-exchange/MCROSS-*.md`

---

## 1. Problem / goal

Show how the same coin trades on **Binance / Coinbase / Bybit** from coin detail, without switching markets manually.

---

## 2. Behavior (happy path)

1. Open coin detail for e.g. Binance `BTCUSDT`.  
2. See **Across exchanges** section with rows for each venue (price, 24h %, volume).  
3. Coinbase row uses a mapped symbol such as `BTC-USD` when available.  
4. Tap Coinbase row → detail for that venue/symbol.  
5. If a venue has no pair, that row shows unavailable; others still load.

### Limits

- Mechanical ticker comparison only — **not** financial advice or arb signals.  
- Quotes may differ (USDT vs USD); do not invent FX conversion in v1.  
- Trade count may be empty off Binance.

---

## 3. APIs

| Method | Path | Use |
|--------|------|-----|
| `GET` | `/api/v1/market/ticker/24h` | One call per exchange (parallel) |
| `GET` | `/api/v1/market/exchanges` | Optional venue list |

No new backend for v1. MCP already has `get_ticker` with exchange.

---

## 4. Code homes (planned)

| Area | Path |
|------|------|
| Helpers | `mobile/src/libs/utils/crossExchange.ts` |
| Constants | `mobile/src/config/crossExchangeConstants.ts` |
| Organisms | `mobile/src/components/organisms/cross-exchange-compare/` |
| Wire-in | `modules/markets/pages/coin-detail-page/` |

---

## 5. How to verify

```bash
cd backend && go run ./cmd/server
cd mobile && npm run web
# Open BTC on Binance → Across exchanges shows Coinbase/Bybit when listed
```

---

## 6. Known limitations / follow-ups

- Candidate symbol heuristics may miss exotic quotes.  
- Optional dedicated compare API for AI/agents later.  
- No depth/spread chart in v1.
