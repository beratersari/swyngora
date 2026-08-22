# ADR 0001: Binance-only market data and supply

## Status

Superseded decision (2026-07-25): **Binance only** for market metrics and circulating supply.  
Original (same day): Binance for market data + CoinGecko for supply — **withdrawn**.

## Context

Swyngora needs candlesticks, 24h volume, spot listings, and supply metrics aligned with Binance-listed assets.

- Official Binance Spot REST (`api.binance.com`) provides klines, 24h ticker, and exchangeInfo without an API key, but **not** supply metrics.
- Binance public marketing symbol list (`/bapi/composite/v1/public/marketing/symbol/list`) exposes **circulatingSupply**, **totalSupply**, and **maxSupply** (plus price) for listed assets.
- CoinGecko free markets pages only cover roughly the top ~1,000 coins by global mcap, so many Binance-listed bases had null supply when CoinGecko was the sole source.

## Decision

1. Use **Binance** public Spot REST for candles, 24h ticker/volume, and spot market listing.
2. Use **Binance** public marketing symbol list for **circulating / total / max supply** (and optional USD price).
3. Do **not** call CoinGecko (or any other third-party metadata API) for supply.
4. Refresh the marketing list on a daily schedule (default 03:00 UTC) plus startup; user requests read the supply cache only.
5. Document the split (Spot REST vs marketing list) in OpenAPI, API notes, and package docs.
6. No paid plan / pricing-tier selection in product code.

## Consequences

- Supply coverage matches Binance marketing-listed symbols (~470), not global mcap rank or every historical pair on exchangeInfo.
- Max supply is null when Binance does not define a hard cap (e.g. ETH).
- Marketing list is a public web API used by Binance UI; shape can change — isolate parsing in the Binance adapter and test with fixtures.
- ADR filename retains historical `coingecko` slug; content is Binance-only.

## Alternatives considered

| Option | Why not |
|---|---|
| CoinGecko free markets | Incomplete vs Binance book; ticker mismatches |
| Product catalog (`cs` only) | Circulating only; no total/max |
| Hybrid Binance + CoinGecko | User requested Binance-only |
| Official Spot-only | No supply fields on documented market endpoints |


---

## Amendment (2026-07-25): multi-exchange market data

**Spot market data** (candles, 24h ticker, listings) may now come from **Binance**, **Coinbase**, or **Bybit** public free endpoints, selected via `exchange` query parameter.

**Supply / circulating metrics** remain **Binance marketing symbol list only** (daily snapshot), applied as asset-level enrichment across venues. **Exception (delist rows):** if that snapshot has no circulating supply for a scheduled delist base, market cap is filled from CoinGecko’s public `/coins/markets` (no key, exact ticker). Default `/supply` and non-delist lists stay Binance-only. No paid API tiers.

**Holder snapshots** are a separate feed (CoinMarketCap public data-api, mapped via marketing `cmcUniqueId`). See [0003](0003-coinmarketcap-holder-source.md).
