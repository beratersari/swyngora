# adapter/cryptoid

Public [Chainz CryptoID](https://chainz.cryptoid.info/api.dws) client for UTXO
holder counts when CoinMarketCap has no table (PIVX, some Dash-like coins).

## Purpose

`GET {base}/{coin}/api.dws?q=addresses` → `{ known, nonzero }`
`GET {base}/{coin}/api.dws?q=rich` → top addresses + `total` supply for shares.

No API key. Coin path is the lowercased base ticker (`PIVXUSDT` → `pivx`).

## How to test

```bash
cd backend && go test ./internal/adapter/cryptoid/
```
