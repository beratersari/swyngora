# Feature: Crypto holder snapshots

## Problem / goal

Coin detail already shows supply. Users also want **who holds the coin**: address count,
concentration (top 10 / 50 / 100 %), and the largest wallets.

## Behavior

- `GET /api/v1/market/holders?asset=BTC` (or `symbol=BTCUSDT`)
- Crypto only. Equities skip the query in product UI.
- `404` `catalog_unmapped`: no Binance marketing `cmcUniqueId`.
- `404` `holders_unpublished`: CMC has an id but no holder table.
- Response includes `holderCount`, optional `dailyActive`, top-10/20/50/100 share
  percents, up to 20 `topHolders`, and `stale` when last-good is served.
- Ticker → CoinMarketCap id comes from the daily Binance marketing snapshot
  (`cmcUniqueId`), including rows that have an id but no supply numbers.
  Pair forms (`BTC-USD`, `ETHTRY`) normalize to the base asset.
- Request path uses a 1h TTL cache (env `HOLDERS_CACHE_TTL`). A 429, upstream
  error, or empty CMC blip serves last-good (`stale: true`) when present.
  Unpublished assets are negative-cached so they do not hammer CMC.
- Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/holders.go` |
| Catalog | Binance marketing list (`LookupAsset`) |
| Adapter | `backend/internal/adapter/cmc/` |
| Service | `GetHolders` |
| HTTP | `GET /api/v1/market/holders` |
| MCP / AI | `get_holders` |
| Profile | `GET /api/v1/market/asset-profile` + MCP `get_asset_profile` |
| Web | `frontend/src/components/organisms/HolderPanel/` on coin detail |
| Mobile | holder count + top-10 share on coin-detail stats |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/cmc/ ./internal/adapter/binance/ ./internal/service/market/ ./internal/transport/http/handler/
curl -s 'http://localhost:8080/api/v1/market/holders?asset=BTC' | jq '{asset, holderCount, topTenSharePct, topHolders: .topHolders[:3]}'
```

Open `/markets/binance/BTCUSDT?tab=holders` — Holders tab. Wallet size uses share × circulating supply when the raw CMC balance is dust-scale, plus an estimated USD value.

## Limits

- Coverage follows CoinMarketCap’s published holder tables, not every Binance pair.
- Public web JSON can change shape — parsing is isolated and fixture-tested.
- Top wallets are not labeled (exchange / contract / unknown).
- Stocks (`nasdaq` / `bist`) are skipped.
- Web Holders tab explains unmapped vs unpublished. Mobile shows the same reason
  string and hides holder tiles on equities.
