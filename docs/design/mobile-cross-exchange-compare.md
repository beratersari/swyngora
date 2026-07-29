# Design: Mobile cross-exchange coin comparison

**Status:** Planned  
**Epic:** `project-management/epics/mobile-cross-exchange-compare.md`  
**Feature:** `docs/features/mobile-cross-exchange-compare.md`  
**Tasks:** `project-management/tasks/mobile/cross-exchange/MCROSS-*.md`  
**Branch:** `feature/mobile-cross-exchange-compare`  
**Depends on:** Coin detail (done)  
**Backend:** Existing multi-exchange ticker / spot only — **no new endpoints** in v1

---

## 1. Problem / goal

Users open a coin on one venue (e.g. Binance `BTCUSDT`) but cannot see **how price, 24h change, and volume compare** on Coinbase and Bybit without manually switching markets.

**Goal:** On coin detail, show a **side-by-side (or stacked) comparison** across configured exchanges for the same **base asset**, with venue-correct symbols.

---

## 2. Product framing

| In this epic | Out of scope |
|--------------|--------------|
| Coin detail “Across exchanges” section | New compare API / aggregation endpoint |
| Parallel `GET /ticker/24h` (and optional spot row) per venue | WebSocket / streaming arb |
| Symbol mapping helpers (BTCUSDT ↔ BTC-USD) | Paper trading / alerts on spread |
| Partial failure per exchange | True arbitrage execution |
| Tap row → open detail for that exchange/symbol | Web (`frontend/`) redesign |

---

## 3. Data sources (compose existing APIs)

| Need | Endpoint | Notes |
|------|----------|-------|
| Per-venue quote | `GET /api/v1/market/ticker/24h?exchange=&symbol=` | Primary metrics: last, change %, quoteVolume, high/low |
| Venue list | `GET /api/v1/market/exchanges` or hardcode supported set | binance, coinbase, bybit |
| Optional liquidity row | `GET /api/v1/market/spot?q=&exchange=&base=` | Only if ticker insufficient; prefer ticker-only for v1 |
| Supply | Shared Binance supply (already on detail) | **Do not** re-fetch per exchange |

**No MCP/OpenAPI change required** for client composition. Optional later: dedicated `GET /api/v1/market/compare?base=` for AI + lower chattiness (out of scope v1 unless team prefers backend).

---

## 4. Symbol mapping (critical)

Backend already normalizes Coinbase-style symbols (`BTCUSD` / `btcusd` → `BTC-USD`). Mobile still must **guess candidates** per venue from the current detail route.

### Strategy (v1)

1. Parse current `symbol` + `exchange` into **base** and preferred **quote** when possible (e.g. `BTCUSDT` → base `BTC`, quote `USDT`; `BTC-USD` → base `BTC`, quote `USD`).
2. Build candidate symbols per exchange:

| Exchange | Preferred candidates (try in order) |
|----------|-------------------------------------|
| binance | `{BASE}USDT`, `{BASE}USDC`, `{BASE}USD` |
| bybit | `{BASE}USDT`, `{BASE}USDC` |
| coinbase | `{BASE}-USD`, `{BASE}-USDT`, `{BASE}-USDC` |

3. For each non-source exchange: call ticker with first candidate; on **404 / not found**, try next candidate once (cap attempts).  
4. Source exchange row uses the **route symbol** (no remap).  
5. If all candidates fail → row status `unavailable` (not fatal for section).

Helpers must be pure + unit-tested (`libs/utils/crossExchange.ts` or similar). Prefer reusing any existing format/normalize utils.

---

## 5. UX flow

```text
Coin detail (any exchange/symbol)
  → section "Across exchanges"
  → parallel tickers for each configured venue
  → rows: exchange · symbol used · last · 24h% · quote vol
  → highlight best/worst price (optional visual, not a trade signal)
  → tap other venue → navigate Detail { exchange, symbol } on same stack
  → partial errors: show per-row failed; other rows still visible
```

### Placement

- Below stats / above or below chart (product choice: **below header stats**, before chart) so users see venues without scrolling past chart.
- Disclaimer caption: informational; prices are venue-local; not arb advice.

### Polling

- Align with detail ticker poll (~same as `DETAIL_TICKER_POLL_MS`) while section focused + AppState active.
- Skip when app backgrounded / screen unfocused.
- Bound concurrency: 3 exchanges max (current product set).

---

## 6. UI structure

```text
CoinDetail
  Header / stats (existing)
  [NEW] CrossExchangeCompare organism
    row: Binance  BTCUSDT   $…  +x%  vol…
    row: Coinbase BTC-USD   $…  …
    row: Bybit    BTCUSDT   $…  …
  Interval + chart (existing)
  …
```

Atomic (kebab-case):

| Component | Level | Role |
|-----------|-------|------|
| `cross-exchange-row` | molecule or organism | One venue line |
| `cross-exchange-compare` | organism | List + loading skeletons + section title |

Props only — no RTK in organisms.

---

## 7. ViewModel sketch

```ts
type CrossExchangeRowVM = {
  exchange: string;
  symbol: string;       // resolved symbol used for request
  isSource: boolean;    // current detail venue
  lastPriceLabel: string;
  changePercentLabel: string;
  changeTone: 'success' | 'danger' | 'neutral';
  quoteVolumeLabel: string;
  status: 'ok' | 'loading' | 'unavailable' | 'error';
  errorMessage?: string;
};

// On CoinDetailPage ViewModel (or colocated hook)
{
  crossExchangeTitle: string;
  crossExchangeRows: CrossExchangeRowVM[];
  crossExchangeDisclaimer: string | null;
  onPressExchangeRow(exchange: string, symbol: string): void;
}
```

---

## 8. i18n

Namespace `detail` (preferred): section title, unavailable, error, disclaimer, a11y labels. **en** + **tr**.

---

## 9. Testing

- Pure: parse base/quote, candidate lists, pick winner label  
- Organism: loading / error / press  
- Detail ViewModel: mock RTK multi-query or lazy queries; partial failure  
- No live network in unit tests  

---

## 10. Acceptance (epic-level)

- [ ] Detail shows comparison for supported venues when possible  
- [ ] Coinbase uses hyphenated symbols; Binance/Bybit compact  
- [ ] Source venue always listed with route symbol  
- [ ] One failed venue does not hide others  
- [ ] Tap non-source venue opens that coin detail  
- [ ] Poll pauses on background / unfocus  
- [ ] Tests + docs closed  

## 11. Out of scope

- New OpenAPI compare path (v1)  
- Multi-hop quote conversion math beyond venue-native quotes  
- Order-book depth / spreads beyond 24h ticker  
- Alerts on cross-venue premium  
- Stocks  
