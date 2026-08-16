# Feature: Crypto holder snapshots

## Problem / goal

Coin detail already shows supply. Users also want **who holds the coin**: address count,
concentration (top 10 / 50 / 100 %), and the largest wallets.

## Behavior

- `GET /api/v1/market/holders?asset=BTC` (or `symbol=BTCUSDT`)
- Crypto only. Equities and unmapped tickers return `404`.
- Response includes `holderCount`, optional `dailyActive`, top-10/20/50/100 share
  percents, and up to 20 `topHolders` (`address`, `balance`, `sharePct`).
- Ticker → CoinMarketCap id comes from the daily Binance marketing snapshot
  (`cmcUniqueId`). Holder JSON is CoinMarketCap’s public `data-api` (no paid plan).
- Request path uses a 1h TTL cache (env `HOLDERS_CACHE_TTL`). A 429 or upstream
  error serves last-good when present.
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
