# Compare mode (relative performance)

## Goal

Overlay 2–3 trading pairs as **% change from the first candle** in a shared interval window.

## Behavior

- URL: `/compare?pairs=binance:BTCUSDT,binance:ETHUSDT&interval=1h`
- Max **3** pairs
- Candles from existing market API; pure client normalization
- Lightweight Charts multi-line host

## Code

| Piece | Path |
|-------|------|
| Utils | `frontend/src/libs/utils/compareSeries.ts` |
| Chart | `frontend/src/components/molecules/CompareChartHost/` |
| Page | `frontend/src/components/pages/ComparePage/` |

## Verify

```bash
cd frontend && npm test -- compare
```
