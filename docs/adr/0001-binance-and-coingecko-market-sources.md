# ADR 0001: Binance for market data, CoinGecko for supply

## Status

Accepted (2026-07-25)

## Context

Swyngora needs candlesticks, 24h volume, and supply metrics. The target exchange for trading market data is Binance. Binance public REST APIs provide klines and 24hr ticker (including base and quote volume) without an API key. They do **not** expose circulating / total / max supply on public market endpoints.

## Decision

1. Use **Binance** public REST as the source for candles and 24h ticker/volume.
2. Use free **CoinGecko** public REST for circulating / total / max supply (and optional USD price).
3. Document the split clearly in OpenAPI, API responses (`source` / `note`), and package docs.
4. Cache both with TTLs; no paid plan / pricing-tier selection in product code.

## Consequences

- Clients must understand that supply is not exchange-native Binance data.
- CoinGecko free-tier rate limits may require longer TTLs or later self-hosting.
- Adding another exchange later should keep the same domain ports (`MarketDataPort`, `SupplyPort`).
