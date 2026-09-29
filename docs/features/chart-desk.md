# Chart desk

## Goal

Show the book, the tape, and estimated liquidation pockets on the same candle a person is already looking at, and say how long that coin has actually been recorded.

## Behavior

On a crypto coin Overview chart:

- Resting bid and ask size from the last 30 minutes of the order-book heatmap sits behind the candles. Green is bids, red is asks. Stronger color is more size.
- The right edge is the footprint for the visible window: buy versus sell at each price. Panning the chart picks the nearest flow window (5 minutes through 7 days).
- Bars just left of that footprint are the largest estimated long and short liquidation pockets. The dashed 10× / 25× / 50× / 100× lines stay.
- A pane under CVD is Binance crowd long percent versus top-trader long percent.
- A line under the chart names how long futures open interest has been stored on each venue. A venue with no samples is left out. The ten seed pairs can show older history because startup backfill covers them. Other liquid names start when this process begins sampling them.

The Book, Footprint, and Crowd vs top switches turn those layers off. Equities do not get them.

## Where the code lives

- Chart drawing: `frontend/src/components/molecules/CandleChartHost/CandleChartHost.desk.ts`
- Mappers: `frontend/src/components/pages/CoinDetailPage/chartDesk.ts`
- Recorder universe: `backend/internal/domain/liquid_recorder.go` (36 names, seeds first)
- Recording span: `GET /api/v1/market/history-span`, MCP `get_history_span`

## How to verify

Open `/markets/binance/BTCUSDT`. The overview chart shows the book shading, the footprint, the liquidation bars, and the crowd strip when those series have data. The caption names the futures recording span once a sample exists.

`go test` for `domain`, `futureshist`, `bookhist`, `market`, and the history-span handler. Frontend: `chartDesk.test.ts` and `DetailChartToolbar.test.tsx`.

## Limits

Book shading is the live 30-minute heatmap, not years of depth. Footprint uses the stored flow ladder while this process runs (about 7 days) and the live tape inside that. Liquidation pockets are the leverage model, not a map of forced closes. The wider recorder samples about 36 USDT perpetuals. Binance books outside the seed list use the top 100 levels and are snapshotted about every 2 minutes. Flashes stay on the seed pairs and coins someone opened. Bybit and Coinbase snapshot the wider list every minute. Startup still backfills public futures history for the seed pairs only.
