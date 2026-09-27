# Spot vs perpetual book

## Goal

Show whether resting liquidity is stronger in the **spot** book or the **perpetual** book, and which side (bids or asks) is thinner on each.

## Behavior

`GET /api/v1/market/orderbook/spot-perp?symbol=BTCUSDT`

- Bands are **±0.1% / ±0.5% / ±1%** of each book's own mid. `rangePct` (default **1**, max 1) is the widest band considered. The headline is the widest band both books actually reach.
- Spot sums **Binance, Coinbase, and Bybit**. Perp sums **Binance USD-M, Bybit linear, and OKX USDT-swap**.
- A venue counts in a band only when that book reaches the percent on **both** sides. A shallow snapshot is not treated as a thin book.
- `thinnerBook` is `spot`, `perp`, or `even` (within about 10% of the larger total).
- `thinnerSide` on each book is `bid`, `ask`, or `balanced` (inside about 8% of that book's two-sided size).
- `venues` compares Binance spot with Binance perp, and Bybit spot with Bybit perp.
- `ratio` is perp total divided by spot total at the headline band.

Informational only — not financial advice.

Web: coin detail **Flow** tab, above perp depth.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/spot_perp_book.go` |
| Service | `GetSpotPerpLiquidity` |
| HTTP | `GET /api/v1/market/orderbook/spot-perp` |
| MCP / AI | `compare_spot_perp_liquidity` |
| UI | `frontend` coin detail `CoinFlowTab` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ -run SpotPerp
curl "http://localhost:8080/api/v1/market/orderbook/spot-perp?symbol=BTCUSDT"
```
