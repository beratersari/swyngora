# Hyperliquid

Public market data for Hyperliquid USDC-margined perpetuals. Names line up with the USDT symbol, so BTC is `BTCUSDT`. There is no API key.

## Layout

- `client.go` reads `POST /info` `metaAndAssetCtxs` for the mark, coin open interest, and the predicted hourly funding rate, and pages `fundingHistory` for settled prints.
- `book.go` reads `POST /info` `l2Book`. The public snapshot is at most 20 levels a side. A band those levels do not reach stays incomplete.
- `trades.go` reads `POST /info` `recentTrades` and keeps an in-memory tape. `B` is an aggressive buy and `A` is an aggressive sell. Size is coin size, so notional is price times size.
- `tradehub.go` listens to `wss://api.hyperliquid.xyz/ws` subscription `trades` and records the same tape.

## What this venue does not publish

Liquidations and a crowd or top-trader ratio. Those columns stay unsupported. They are not zeros. Hyperliquid is not part of `exchange=all`.

## How the tape grows

`recentTrades` is only the newest prints. There is no paged trade history. The websocket keeps the tape moving while this process is watching, and the futures worker stores the resulting minutes. A window the tape has not covered stays blank.

## Config

- `HYPERLIQUID_INFO_URL` defaults to `https://api.hyperliquid.xyz`
- `HYPERLIQUID_WS_URL` defaults to `wss://api.hyperliquid.xyz/ws`

## Test

From `backend/`:

```bash
go test ./internal/adapter/hyperliquid/
```
