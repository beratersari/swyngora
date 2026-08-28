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
3. Cache per normalized base (default 1h). `WBTC` and `W` are different keys
   (`NormalizeAssetKey` does not peel `BTC`/`ETH`/`BNB` off wrapped tickers).
   Serve last-good on 429 / upstream errors.
4. Do not add paid-plan selection or API keys for this source.
5. Keep parsing in `adapter/cmc` with fixtures.

## Consequences

- First hop is still CMC public detail (id, then slug).
- When CMC has no table, the holders cascade tries Coin Metrics `AdrBalCnt`,
  GeckoTerminal `/info`, Ethplorer `freekey` (ERC-20), Routescan EVM
  (including Chiliz), and Tronscan TRC-20 using published contracts.
- Same class of risk as other public web JSON: shapes can change; parsing is isolated.
- ADR 0001 still applies to **supply** (Binance only). Holders are a separate feed.

## Alternatives considered

| Option | Why not |
|---|---|
| CoinGecko / GeckoTerminal holders | GeckoTerminal `/info` now publishes `holders.count` (used as fallback 3). `/top_holders` is unauthorized without a paid CoinGecko plan and is not used. |
| Ethplorer + Blockchair | Ethplorer `freekey` is used as fallback 4 for ERC-20 only. |
| Official CMC Pro API | Paid plan — rejected by project policy |
