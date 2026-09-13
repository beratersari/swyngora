# ADR 0005: OKX as a third futures venue

## Status

Accepted (2026-09-13).

## Context

Liquidation / OI / funding desks were Binance USD-M + Bybit linear only.
That is thinner than CoinGlass-style market coverage. OKX USDT-swap is
the next large public book (no API key).

## Decision

1. Add `ExchangeOKX` as a **futures-only** venue. Do **not** add it to
   `SupportedExchanges` (spot / equity listing).
2. Public REST + public WS only (`www.okx.com`, `ws.okx.com`).
3. Map `BTCUSDT` ↔ `BTC-USDT-SWAP`. Use `oiCcy` / `oiUsd` for OI.
   Liquidation `sz` is contracts; notional is `sz * ctVal * bkPx`.
4. Combined `exchange=all` includes Binance, Bybit, and OKX. A missing
   venue is listed, not replaced.
5. Stored futures history and live funding-arb legs include OKX.
   Around spot candles stay Binance/Bybit; OKX attaches stored futures
   when present.

## Consequences

- Combined `complete` is stricter (three live streams).
- Venue divergence remains a Binance vs Bybit headline (API fields).
  Positioning / squeeze / snapshot / hunt list OKX as a third venue.
- No OKX account long/short or spot order book in this change.
