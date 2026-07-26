# DET-4: RSI/EMA panel on detail (implementation)

| Field | Value |
|---|---|
| **ID** | DET-4 |
| **Epic** | coin-detail-and-indicators |
| **Status** | done |
| **Area** | frontend |
| **Type** | implementation |
| **Blocked by** | DET-B, DET-1, DET-3 |

## Summary

IndicatorPanel: latest RSI/EMA cards, RSI 0–100 chart with 30/70 bands, optional EMA overlays on candle chart, disclaimer copy, mapper unit tests. See DET-B design decisions.

## Status

**done.** `IndicatorPanel` + `IndicatorChartHost` (RSI 0–100, bands 30/70), EMA overlays on candles, mappers/tests in `libs/utils/indicators*`.
