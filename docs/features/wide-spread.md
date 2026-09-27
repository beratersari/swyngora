# Wide bid-ask spreads

## Goal

Show which coins have a bid-ask spread that has opened versus that coin's own recent typical spread.

## Behavior

Samples are recorded from books this process already watches (the live order book, the heatmap warmer, and the wall sampler). No extra polling loop.

`GET /api/v1/market/orderbook/spreads/scan`

- Latest top-of-book spread versus the median of earlier samples of the same book.
- A coin is listed when the latest spread is at least **1.5×** its typical and at least 0.005 percentage points wider.
- At least 5 samples are required. Until then the list stays empty.
- Widest ratio first. `exchange=all` keeps each venue's book separate.

`GET /api/v1/market/orderbook/spread?symbol=BTCUSDT` is the same read for one coin. The coin order book shows the sentence when the spread has opened.

Web: `/spreads`.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/spread_widen.go` |
| Service | `ScanWideSpreads`, `GetWideSpread` |
| HTTP | `GET /api/v1/market/orderbook/spreads/scan`, `GET /api/v1/market/orderbook/spread` |
| MCP / AI | `scan_wide_spreads`, `get_wide_spread` |
| UI | `frontend` `/spreads` and the coin order book |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ -run Spread
```
