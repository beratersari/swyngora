# Taker buy / sell flow

## Goal

Show whether **buyers or sellers are hitting the futures book more
aggressively** right now. That is different from long/short **account**
ratio (who already holds positions).

## Behavior

`GET /api/v1/market/taker-flow?symbol=BTCUSDT`

- `exchange=all` (default): **Binance** and **Bybit** separately plus `combined`.
- Windows: **5m**, **1h**, **4h**.
- Each window: `buyNotional`, `sellNotional`, `delta` (buy − sell), `deltaPct`,
  `dominant` (`buy` / `sell` / `balanced`).
- `summary` reads taker flow together with price, open interest, and funding
  (e.g. aggressive buying + rising price + rising OI ≈ long buildup).

### Sources

| Venue | Source |
|---|---|
| Binance USD-M | Public `takerlongshortRatio` 5m bars (4h available immediately) |
| Bybit linear | Recent REST trades + `publicTrade` websocket (4h `complete` after this process has been collecting) |

Volumes are **USDT notional**.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/taker_flow.go`, `taker_book.go` |
| Adapters | `adapter/binance/takerflow.go`, `adapter/bybit/takerflow.go`, `tradehub.go` |
| Service | `backend/internal/service/market/takerflow.go` |
| HTTP | `GET /api/v1/market/taker-flow` |
| MCP / AI | `get_taker_flow` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ -count=1
curl "http://localhost:8080/api/v1/market/taker-flow?symbol=BTCUSDT"
```
