# Repeated same-size prints

## Goal

Average print size says a window was made of large or small fills. It
does not say whether **the same size keeps hitting again**. This read
flags a clip that shows up several times inside a minute.

## Behavior

`GET /api/v1/market/bot-tracker?symbol=BTCUSDT`

- Last **5 minutes** of aggressive futures prints
- A size is flagged when it appears **4 or more times inside any 60 seconds**
- Buy and sell are separate. `buyCount`, `sellCount`, and `delta` count **only the repeated clips**, not every market order. The tape labels them "Repeated buy prints", "Repeated sell prints", and "Repeated delta".
- Sizes within about **0.2%** count as the same clip
- Prints under **5,000 USDT** are ignored
- `exchange=all` (default) returns Binance, Bybit, and OKX, plus the busiest clips across them

Example: six market buys near 57,889 USDT inside 40 seconds is one buy clip. The same size spaced 40 seconds apart across five minutes is not, because no 60-second stretch holds four of them.

| Venue | Source |
|---|---|
| Binance USD-M | Live `aggTrade` tape when it covers the window, otherwise paged `aggTrades` |
| Bybit linear | Live `publicTrade` tape when it covers the window, otherwise the newest public trades |
| OKX USDT-swap | Live public `trades` tape when it covers the window, otherwise the newest public trades |

A repeated size is a pattern. It is not proof of a named bot. Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/bot_tracker.go` |
| Service | `GetBotTracker` in `backend/internal/service/market/bottracker.go` |
| HTTP | `GET /api/v1/market/bot-tracker` |
| MCP / AI | `get_bot_tracker` |
| Web | Coin detail futures tape, under print size. Overview chart draws a square on the candles on screen when the burst falls in that view. The read itself is still the last 5 minutes |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ -count=1 -run 'BotTracker|PrintSize'
curl "http://localhost:8080/api/v1/market/bot-tracker?symbol=BTCUSDT"
```
