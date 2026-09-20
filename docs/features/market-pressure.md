# Combined market pressure

## Goal

The **resting order book** can look bid-heavy, **CVD** can be selling, and
**absorption** can say bids are taking that flow. Those three reads should
collapse to **one −1…+1 score** so you can see which side actually has
pressure, plus **how that score moved over the last hour**.

## Behavior

`GET /api/v1/market/pressure?symbol=BTCUSDT`

- `exchange=all` (default) uses the combined spot book and combined
  Binance+Bybit CVD / absorption
- `current.value` is **−1 (sell) to +1 (buy)**
- `components` stay visible: `book` (resting imbalance), `flow` (1h CVD
  buy−sell share), `absorption` (bids absorbing sells = buy, asks
  absorbing buys = sell)
- When book and CVD **disagree** and absorption is at least moderate,
  absorption is weighted more (50%) so the net is not a wash
- `agreement` is `same` | `mixed` | `opposite`
- `change1h` is this score minus the score ~1h ago (book history + the
  previous hour of CVD/absorption). `windows[0].kind` is
  `more_buy` | `more_sell` | `flipped` | `stable`

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/market_pressure.go` |
| Service | `backend/internal/service/market/pressure.go` |
| HTTP | `GET /api/v1/market/pressure` |
| MCP / AI | `get_market_pressure` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ ./internal/transport/mcp/ -run Pressure
curl "http://localhost:8080/api/v1/market/pressure?symbol=BTCUSDT"
```
