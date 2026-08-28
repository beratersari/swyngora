# Feature: RSI heatmap

## Problem / goal

The `/heatmap` page already maps 24h price change. Traders want the
same **CoinGlass / DataWallet RSI map**: each pair is a **dot**, ranked
by size, plotted against Wilder RSI.

## Behavior

`GET /api/v1/market/rsi-heatmap`

- One venue (`exchange`, default `binance`) and quote (default `USDT`)
- One `interval` (default `1h`; 15m / 4h / 1d on the web)
- Top `limit` pairs (default **100**, max **200**) by `sort`
  (`marketCapCirculating` default, or `quoteVolume`)
- **Stables omitted** (USDC, USDT, FDUSD, …)
- Seeded from **~300 closed candles**
- `averageRsi` plus oversold / neutral / overbought counts
- Cached **60s**. Informational only — not financial advice

Web: `/heatmap?view=rsi`. Timeframe and Top 50 / 100. Dots are equal
size; left is rank #1 by market cap. Hover for RSI; click to open
coin detail.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/rsi_heatmap.go` |
| Service | `backend/internal/service/market/rsi_heatmap.go` |
| HTTP | `GET /api/v1/market/rsi-heatmap` |
| MCP / AI | `get_rsi_heatmap` |
| Telegram | `/rsiheat [exchange] [quote]` |
| Web | `frontend` `RSIHeatmap` + `HeatmapPage` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ -run RSIHeat -count=1
curl -sS 'http://127.0.0.1:8080/api/v1/market/rsi-heatmap?limit=50&interval=1h'
```

Open http://localhost:5174/heatmap?view=rsi

## Limits

- First load fetches ~300 candles per plotted pair (same process-wide
  batch semaphore as indicator batch). Later hits use the 60s cache.
- Values can still differ from another site if that site uses perps, the
  forming bar, or a short Cutler/SMA RSI.
- Equities work when the venue has candles.
