# MCROSS-1: Constants + cross-exchange helpers + tests

| Field | Value |
|---|---|
| **ID** | MCROSS-1 |
| **Epic** | mobile-cross-exchange-compare |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/cross-exchange/MCROSS-1.md` |

## Summary

Pure helpers + constants (no UI):

- `config/crossExchangeConstants.ts` — venue list order, max candidate tries, poll alignment note  
- `libs/utils/crossExchange.ts` —  
  - `parseMarketSymbol(exchange, symbol)` → `{ base, quote } | null`  
  - `symbolCandidatesForExchange(base, targetExchange)` → ordered string[]  
  - `buildCrossExchangePlan(sourceExchange, sourceSymbol, venues)` → plan rows  
  - mappers: ticker DTO → row view fields (reuse `formatPrice` / `formatChangePercent` / `changeTone` / compact volume)  
- Unit tests for parse, candidates, plan, edge cases (`BTC-USD`, `btcusdt`, unknown)  
- Export from `libs/utils/index.ts`

## Design

`docs/design/mobile-cross-exchange-compare.md` §3–4 · `MCROSS-A.md`

## Acceptance

- [x] Coinbase candidates hyphenated; Binance/Bybit compact  
- [x] Source venue plan uses original symbol only  
- [x] No React / navigation imports  
- [x] Tests green  
- [x] Status → done when finished  

## Status

`done`
