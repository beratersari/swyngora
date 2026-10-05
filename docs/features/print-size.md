# Average print size

## Goal

High buy volume on a 2% up-move can be **a few large market buys** or
**many small trades**. Show **average USDT size per aggressive fill**,
buy vs sell, and whether that average just jumped (e.g. 20K → 180K in
15 minutes).

## Behavior

`GET /api/v1/market/print-size?symbol=BTCUSDT` is one coin.

`GET /api/v1/market/print-size/overview` is the liquid USDT pairs together.

- The market average sums notional and fill counts, then divides. A busier coin weighs more
- Each window is compared with the same length just before, so the average shows whether prints got larger or smaller
- `largerCoins`, `smallerCoins`, `fewLargeCoins`, and `manySmallCoins` are one vote per pair
- The list is the recorder's liquid USDT perpetuals, seed pairs first (default 100, min 5, max 100). The Print size page requests that full list, shows 5m and 15m, and ranks the pairs
- `coverage` names Binance, Bybit, and OKX as fresh, old, or missing. `definite` is false when any of them is missing or old, and the page does not show a market size read (`docs/features/venue-coverage.md`)

One coin:

- `exchange=all` (default): Binance USD-M and Bybit linear separately plus `combined`
- Windows: **5m** and **15m**, each vs the **same length just before**
- Per side: `avg`, `priorAvg`, `count`, `notional`, `ratio` (current/prior), `kind` (`jumped` / `faded` / `stable`)
- `read`: `few_large` (size jumped, count did not), `many_small` (count jumped, size did not), `larger`, `smaller`, `stable`
- Combined averages **total notional / total count** (not a mean of venue averages)

Sources:

| Venue | Source |
|---|---|
| Binance USD-M | Live `aggTrade` tape when it covers the window, otherwise stored flow-at-prices minutes, otherwise paged `aggTrades` |
| Bybit linear | Live `publicTrade` tape when it covers the window, otherwise stored flow-at-prices minutes, otherwise the newest public trades |
| OKX USDT-swap | Live public `trades` tape when it covers the window, otherwise stored flow-at-prices minutes, otherwise the newest public trades |

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/print_size.go` |
| Adapters | `adapter/binance/whales.go` (`ListPrints`), `adapter/bybit/whales.go` |
| Service | `GetPrintSize`, `GetPrintSizeOverview` |
| HTTP | `GET /api/v1/market/print-size`, `GET /api/v1/market/print-size/overview` |
| MCP / AI | `get_print_size`, `get_print_size_overview` |
| Web | Coin detail futures tape (`TapePanel`). Market desk at `/print-size` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/service/market/ ./internal/transport/http/handler/ -count=1 -run PrintSize
curl "http://localhost:8080/api/v1/market/print-size?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/print-size/overview?limit=100"
```
