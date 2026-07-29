# MCROSS-A: Cross-exchange field matrix + symbol mapping (analysis)

| Field | Value |
|---|---|
| **ID** | MCROSS-A |
| **Epic** | mobile-cross-exchange-compare |
| **Status** | done |
| **Area** | mobile / analysis |
| **Path** | `project-management/tasks/mobile/cross-exchange/MCROSS-A.md` |

## Purpose

Map multi-exchange ticker contracts and symbol conventions to a coin-detail comparison UI. **No backend work required for v1** — compose existing APIs.

Sources:

- OpenAPI: `getTicker24h`, `listExchanges`
- Backend: `normalizeSymbolForExchange` (Coinbase hyphenation)
- Domain: `Ticker24h`, supported exchanges
- Mobile: coin detail ViewModel, `useGetTicker24hQuery`, formatters

---

## 1. Endpoints

| Method | Path | Mobile use |
|--------|------|------------|
| `GET` | `/api/v1/market/ticker/24h` | One request per exchange row |
| `GET` | `/api/v1/market/exchanges` | Optional dynamic venue list |

---

## 2. Ticker fields → UI

| Field | Type | Compare UI |
|-------|------|------------|
| `symbol` | string | Subtitle / resolved symbol |
| `lastPrice` | string | Primary price |
| `priceChangePercent` | string | 24h % + tone |
| `quoteVolume` | string | Volume column (compact) |
| `highPrice` / `lowPrice` | string | Optional secondary; Coinbase may need care |
| `tradeCount` | int | Optional; often 0 off Binance — show "—" |
| `volume` | string | Base volume; deprioritize vs quoteVolume |

---

## 3. Symbol mapping matrix

| From (example) | Base | binance | bybit | coinbase |
|----------------|------|---------|-------|----------|
| binance `BTCUSDT` | BTC | BTCUSDT | BTCUSDT | BTC-USD (then BTC-USDT) |
| coinbase `ETH-USD` | ETH | ETHUSDT | ETHUSDT | ETH-USD |
| bybit `SOLUSDT` | SOL | SOLUSDT | SOLUSDT | SOL-USD |

Document candidate order in design §4; implement in MCROSS-1.

---

## 4. Gap vs mobile today

| Capability | Backend | Mobile today | Epic |
|------------|---------|--------------|------|
| Multi-exchange tickers | ✅ | Single exchange on detail | Parallel fetch |
| Symbol normalize Coinbase | ✅ server | Limited client helpers | Client candidates + server normalize |
| Compare UI | — | — | New organism |
| Dedicated compare API | ❌ | — | Out of scope v1 |

---

## 5. Decisions

1. **Ticker-only v1** (no spot fan-out).  
2. **Max 3 venues** (product set).  
3. **No FX conversion** between USDT/USD quotes — label the quote implicitly via symbol.  
4. **Partial failure** per row.  
5. Optional backend compare endpoint deferred.

---

## Acceptance

- [x] Matrix complete; linked from design/epic  
- [x] Candidate lists agreed for binance/bybit/coinbase  
- [x] Status → done when analysis accepted  

## Status

`done`
