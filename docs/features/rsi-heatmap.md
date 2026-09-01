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
- Seeded from **~120 closed candles** (the venue’s still-forming last bar is dropped)
- `averageRsi` plus oversold / neutral / overbought counts
- Cached **60s**; expired maps are served immediately while they refresh.
  Default Binance USDT 1h Top 100 is kept warm in the background.
  A Top 50 request reuses a larger cached map. Informational only — not financial advice

Web: `/heatmap?view=rsi`. Timeframe and Top 50 / 100. Dots are equal
size and stay circular (the scatter is not stretched to the frame);
left is rank #1 by market cap. Hover a coin for RSI — empty plot
space does not pick a nearby pair. Click to open coin detail.

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

- First uncached load fetches ~120 candles per plotted pair (process-wide
  batch semaphore, 24 at a time). The default 1h map is warmed on startup
  and every 60s so `/heatmap?view=rsi` is usually a cache hit.
- Values can still differ from another site if that site uses perps, the
  forming bar, or a short Cutler/SMA RSI. This endpoint does not seed
  the unfinished venue candle.
- Equities work when the venue has candles.
