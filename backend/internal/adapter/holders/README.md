# adapter/holders

Cascades public holder snapshots for `GET /api/v1/market/holders`.

## Order

1. CoinMarketCap public detail (`cmcUniqueId`, then `slug`)
2. Coin Metrics community `AdrBalCnt` (native L1 address counts)
3. GeckoTerminal `/networks/{net}/tokens/{addr}/info` (`holders.count`)
4. Ethplorer ERC-20 `getTokenInfo` + `getTopTokenHolders` (`freekey`)
5. Routescan EVM `tokenholdercount` / `tokenholderlist` (Chiliz, ETH, BSC, …)
6. Tronscan `token_trc20` (`holders_count`)

Contracts for 3–4 come from the CMC asset profile, then CoinGecko platforms.

Cache keys use `NormalizeAssetKey`: `WBTCUSDT` and `WBTC` share an entry;
`WBTC` and `W` do not.

```bash
cd backend && go test ./internal/adapter/holders/ ./internal/adapter/geckoterminal/ ./internal/adapter/ethplorer/ ./internal/adapter/cmc/
```
