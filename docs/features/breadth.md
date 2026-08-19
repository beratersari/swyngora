# Market breadth (up vs down)

## Goal

Show how much of the market is going up or down: how many of the coins we
follow are up and how many are down, as a count and a percentage, for the
last 1 hour, 4 hours, and 24 hours. Also say whether BTC and ETH are moving
with the rest of the market, or only a few large coins are carrying it.

## Behavior

`GET /api/v1/market/breadth`

Universe: the most liquid **USDT** spot pairs we already list (USD on
Coinbase), one row per base asset, default **80** coins (`limit` max 150).
Leveraged tokens (UP/DOWN/3L/…) are dropped.

For each window:

- `up` / `down` / `flat` counts and percents
- volume-weighted up/down share (24h quote volume as size)
- BTC and ETH percent change
- `alignment`: `with_market` | `carrying` | `lagging` | `mixed`
- a short summary

1h and 4h use Binance rolling-window tickers (other venues fall back to
candles). 24h uses the spot ticker change. Informational only.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/breadth.go` |
| Adapter | `backend/internal/adapter/binance/window.go` |
| Service | `backend/internal/service/market/breadth.go` |
| HTTP | `GET /api/v1/market/breadth` |
| MCP / AI | `get_market_breadth` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/service/market/ -run Breadth
curl "http://localhost:8080/api/v1/market/breadth"
```
