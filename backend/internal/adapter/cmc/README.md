# adapter/cmc

CoinMarketCap **public website** `data-api` client for crypto **holder** snapshots.

## Purpose

Swyngora does not subscribe to CoinMarketCap’s paid API. This adapter calls the
same unauthenticated JSON the public coin pages use:

`GET {CMC_BASE_URL}/data-api/v3/cryptocurrency/detail?id={cmcId}`

or, when Binance has a marketing `slug` but no `cmcUniqueId`:

`GET {CMC_BASE_URL}/data-api/v3/cryptocurrency/detail?slug={slug}`

Ticker → CMC id/slug comes from the Binance marketing symbol list
(`cmcUniqueId` / `slug`), already refreshed daily for supply. An empty
`holders` object with a positive `cdpTotalHolder` still counts as a snapshot.

## Layout

| File | Role |
|---|---|
| `client.go` | HoldersPort: catalog lookup, TTL cache, singleflight, parse |
| `client_test.go` | Fixture parse, cache hit, 404, 429 stale, pair strip |

## How to test

```bash
cd backend && go test ./internal/adapter/cmc/
```

## Dependencies

- `domain.AssetCatalogPort` (Binance marketing snapshot)
- In-memory TTL cache (`HOLDERS_CACHE_TTL`, default 1h)

## Config / env

| Variable | Default | Notes |
|---|---|---|
| `CMC_BASE_URL` | `https://api.coinmarketcap.com` | Public data-api host |
| `HOLDERS_CACHE_TTL` | `1h` | Per-asset snapshot |

No API key. CMC’s public data-api omits the `holders` table for the default
Go HTTP/2 fingerprint; the client uses HTTP/1.1 plus browser `Origin` /
`Referer` / `User-Agent` so the payload matches the public coin page.
Response shape can change — keep parsing isolated and fixture-tested.

## Ownership

Used only by `service/market.GetHolders`. Do not call this adapter from HTTP handlers.
