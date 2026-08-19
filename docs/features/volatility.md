# Price volatility

## Goal

Show how volatile a coin’s price has been: how much it moved over the last
1 hour, 4 hours, and 24 hours, whether that activity is higher or lower than
normal, whether the range is expanding or shrinking, and whether the coin is
jumpier or calmer than BTC and ETH.

## Behavior

`GET /api/v1/market/volatility?symbol=SOLUSDT`

| Field | Meaning |
|---|---|
| `netPct` | Close-to-close change in the window |
| `rangePct` | High–low range as a percent of the start (catches chop that nets to ~0) |
| `realizedPct` | Path noise: stdev of bar returns × √n |
| `vsNormal` | `higher` / `typical` / `lower` vs the median of earlier same-length windows |
| `trend` | `expanding` / `shrinking` / `stable` vs the previous window |
| `vsBtc` / `vsEth` / `vsMarket` | `more_volatile` / `similar` / `calmer` |

1h uses 1-minute bars; 4h and 24h use 5-minute bars. Default venue is Binance
spot. Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/volatility.go` |
| Service | `backend/internal/service/market/volatility.go` |
| HTTP | `GET /api/v1/market/volatility` |
| MCP / AI | `get_price_volatility` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ -run Volatil
curl "http://localhost:8080/api/v1/market/volatility?symbol=SOLUSDT"
```
