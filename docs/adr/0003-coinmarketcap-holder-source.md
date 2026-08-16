# ADR 0003: CoinMarketCap public data-api for holder snapshots

## Status

Accepted (2026-08-16).

## Context

Coin detail needed on-chain holder counts and top wallets. Official CoinMarketCap
and CoinGecko holder APIs sit behind paid plans. Binance Spot REST does not
publish holders. GeckoTerminal’s public token endpoints omit holder lists
(`top_holders` is unauthorized).

The Binance marketing symbol list we already refresh for supply includes
`cmcUniqueId` and `slug` per asset.

## Decision

1. Map `BTC` / `BTCUSDT` → CMC id from the Binance marketing snapshot (cache-only).
2. Fetch holders from the **unauthenticated** CoinMarketCap website API:
   `GET https://api.coinmarketcap.com/data-api/v3/cryptocurrency/detail?id=`.
3. Cache per asset (default 1h). Serve last-good on 429 / upstream errors.
4. Do not add paid-plan selection or API keys for this source.
5. Keep parsing in `adapter/cmc` with fixtures.

## Consequences

- Holder coverage matches CMC’s published tables, not every listed pair.
- Same class of risk as the Binance marketing list: public web JSON can change.
- ADR 0001 still applies to **supply** (Binance only). Holders are a separate feed.

## Alternatives considered

| Option | Why not |
|---|---|
| CoinGecko / GeckoTerminal holders | Free token info has no holder list; onchain holders are paid |
| Ethplorer + Blockchair | Chain-specific, incomplete for CEX tickers |
| Official CMC Pro API | Paid plan — rejected by project policy |
