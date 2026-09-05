# Deribit adapter

Public REST client for **BTC and ETH options** (no API key).

Used by `GET /api/v1/market/options` to build open interest, 24h volume,
put/call ratio, DVOL / ATM implied vol, and expiry distribution.

## Endpoints

| Call | Path |
|---|---|
| Listed options | `GET /api/v2/public/get_book_summary_by_currency?currency=BTC&kind=option` |
| 30-day IV index | `GET /api/v2/public/ticker?instrument_name=BTC-DVOL` |
| IV history (24h change) | `GET /api/v2/public/get_volatility_index_data` |
| USD index | `GET /api/v2/public/get_index_price?index_name=btc_usd` |

Open interest and volume on inverse options are in the **base coin**.
USD notional is `coin × underlying` (or venue `volume_usd` when present).

## Config

| Variable | Default |
|---|---|
| `DERIBIT_BASE_URL` | `https://www.deribit.com` |
| `OPTIONS_CACHE_TTL` | `30s` |

## Tests

```bash
go test ./internal/adapter/deribit/ ./internal/domain/ -count=1
```
