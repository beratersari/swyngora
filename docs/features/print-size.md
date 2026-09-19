# Average print size

## Goal

High buy volume on a 2% up-move can be **a few large market buys** or
**many small trades**. Show **average USDT size per aggressive fill**,
buy vs sell, and whether that average just jumped (e.g. 20K → 180K in
15 minutes).

## Behavior

`GET /api/v1/market/print-size?symbol=BTCUSDT`

- `exchange=all` (default): Binance USD-M and Bybit linear separately plus `combined`
- Windows: **5m** and **15m**, each vs the **same length just before**
- Per side: `avg`, `priorAvg`, `count`, `notional`, `ratio` (current/prior), `kind` (`jumped` / `faded` / `stable`)
- `read`: `few_large` (size jumped, count did not), `many_small` (count jumped, size did not), `larger`, `smaller`, `stable`
- Combined averages **total notional / total count** (not a mean of venue averages)

Sources:

| Venue | Source |
|---|---|
| Binance USD-M | Paged `aggTrades` (up to ~30 minutes) |
| Bybit linear | Newest ~1000 public trades (minutes of tape when busy) |

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/print_size.go` |
| Adapters | `adapter/binance/whales.go` (`ListPrints`), `adapter/bybit/whales.go` |
| Service | `GetPrintSize` |
| HTTP | `GET /api/v1/market/print-size` |
| MCP / AI | `get_print_size` |
| Web | Coin detail futures tape (`TapePanel`) |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/service/market/ ./internal/transport/http/handler/ -count=1 -run PrintSize
curl "http://localhost:8080/api/v1/market/print-size?symbol=BTCUSDT"
```
