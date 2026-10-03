# Taker buy / sell flow

## Goal

Show whether **buyers or sellers are hitting the futures book more
aggressively** right now. That is different from long/short **account**
ratio (who already holds positions).

## Behavior

`GET /api/v1/market/taker-flow?symbol=BTCUSDT`

- `exchange=all` (default): **Binance**, **Bybit**, and **OKX** separately plus `combined`. Each venue sums measured prints (price × size). 1h and 4h stay incomplete until this process has recorded that window.
- Windows: **5m**, **1h**, **4h**.
- Each window: `buyNotional`, `sellNotional`, `delta` (buy − sell), `deltaPct`,
  `dominant` (`buy` / `sell` / `balanced`).
- `summary` reads taker flow together with price, open interest, and funding
  (e.g. aggressive buying + rising price + rising OI ≈ long buildup).

### Sources

| Venue | Source |
|---|---|
| Binance USD-M | `<symbol>@aggTrade` websocket. When that socket is quiet, `aggTrades` is paged forward from the last aggregate id (price × size, not base volume × the current mark). |
| Bybit linear | `publicTrade` websocket, in batches of 10. The newest public trades are merged in as well so a thin socket does not leave a hole. |
| OKX USDT-swap | Public `trades` websocket. Contract size × price. One REST seed of the newest 500 trades when the minute book is still empty. |

Volumes are **USDT notional**. Average size per fill (few large vs many small)
is `GET /api/v1/market/print-size` — see [`print-size.md`](print-size.md).
OI delta as a share of this volume is `GET /api/v1/market/participation` —
see [`participation.md`](participation.md).

Web: coin detail **Flow** tab.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/taker_flow.go`, `taker_book.go` |
| Adapters | `adapter/binance/takerflow.go`, `tradehub.go`; `adapter/bybit/takerflow.go`, `tradehub.go`; `adapter/okx/trades.go`, `tradehub.go` |
| Service | `backend/internal/service/market/takerflow.go` |
| HTTP | `GET /api/v1/market/taker-flow` |
| MCP / AI | `get_taker_flow` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ -count=1
curl "http://localhost:8080/api/v1/market/taker-flow?symbol=BTCUSDT"
```
