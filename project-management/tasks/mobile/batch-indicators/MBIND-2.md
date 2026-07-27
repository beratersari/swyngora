# MBIND-2: Batch indicators helpers + formatters

| Field | Value |
|---|---|
| **ID** | MBIND-2 |
| **Epic** | mobile-batch-indicators |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MBIND-A |

## Summary

Pure helpers under `libs/utils/` (+ `config` constants if needed):

- `chunkSymbols(symbols, max=50)`
- `groupPairsByExchange(pairs)` → `Record<exchange, symbols[]>`
- `buildBatchIndicatorsBody({ exchange, symbols, interval?, rsiPeriod?, emaPeriods? })`
- `indexBatchItemsBySymbol(items)` → `Map<string, Snapshot>`
- `formatRsi(value)` / `rsiTone(value)` (e.g. overbought/oversold/neutral bands for color only — not advice)
- Defaults constants: interval `1h`, rsi 14, ema `12,26`, max batch 50, enrich caps

Unit tests for chunk, group, index, format, tone.

## Design

MBIND-A §4–6 · design doc § helpers

## Acceptance

- [ ] Pure functions only (no React)  
- [ ] Tests green  
- [ ] Exported from `libs/utils/index.ts`  
- [x] Status → done  

## Status

`done`
