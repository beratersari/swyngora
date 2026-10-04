# Chart desk

## Goal

Show the book, the tape, and estimated liquidation pockets on the same candle a person is already looking at, and say how long that coin has actually been recorded.

## Behavior

On a crypto coin Overview chart:

- Resting bid and ask size from the last 30 minutes of the order-book heatmap sits behind the candles. Green is bids, red is asks. Stronger color is more size.
- Estimated liquidation intensity sits behind those candles for 12h, 24h, 3d, 7d, 14d, 30d, or 90d. Gold through magenta is more estimated notional. Longs and shorts can be shown on their own. Binance, Bybit, and OKX stay separate. Combined color is drawn only when all three have their own price and open interest. The caption names the leverage mix, an even long/short split, and any venue missing from the window. The same read is `GET /api/v1/market/liquidation-hunt/heatmap`.
- Open liquidation levels sit on the candles for the same range. A band stays where that slice of open interest was placed until a later candle trades through it. Solid is still open. Dashed is already traded through. The line under the candles is still-open long size minus still-open short size. Combined bands stay off the chart when any venue has no price and open interest, and the caption says the book is withheld. The same read is `GET /api/v1/market/liquidation-open-levels`.
- The right edge is the footprint for the visible window: buy versus sell at each price. Panning the chart picks the nearest flow window (5 minutes through 30 days).
- Bars just left of that footprint are the largest estimated long and short liquidation pockets. The dashed 10× / 25× / 50× / 100× lines stay.
- A pane under CVD is crowd long percent versus top-trader long percent for each venue that publishes both (Binance and OKX).
- The next pane is stored USDT open interest, one line each for Binance, Bybit, and OKX. The pane after that is stored funding percent, settled prints when that venue has them. The next pane is liquidation notional in each candle: short liquidations point up, long liquidations point down, and a candle with both shows the difference. The last pane is still-open long liquidation size minus still-open short liquidation size.
- A line under the chart names how long futures open interest, the spot book, flow-at-prices, and recorded liquidation prints have been stored on each venue. A venue with no samples is left out. Startup backfill covers the liquid set, up to 100 names, as far as each venue publishes.

The Book, Footprint, Crowd vs top, Open interest, Funding, Liquidations, Liq map, and Open levels switches turn those layers off. Range and venue show when either the map or the open levels are on. Side stays on the map. Equities do not get them. The three new panes read `GET /api/v1/market/futures-history` (newest 1,000 rows from the first loaded candle). A venue with no stored rows is left off that pane.

## Where the code lives

- Chart drawing: `frontend/src/components/molecules/CandleChartHost/CandleChartHost.desk.ts`
- Mappers: `frontend/src/components/pages/CoinDetailPage/chartDesk.ts`
- Recorder universe: `backend/internal/domain/liquid_recorder.go` (100 names, seeds first)
- Recording span: `GET /api/v1/market/history-span`, MCP `get_history_span`

## How to verify

Open `/markets/binance/BTCUSDT`. The overview chart shows the book shading, the liquidation heatmap, the footprint, the liquidation bars, and the crowd strip when those series have data. The map caption names the leverage mix. Open interest, funding, and liquidation panes appear under the candles once stored rows exist. The caption names the futures recording span once a sample exists. A range longer than the loaded candles asks you to pan left.

`go test` for `domain`, `futureshist`, `bookhist`, `market`, and the history-span handler. Frontend: `chartDesk.test.ts` and `DetailChartToolbar.test.tsx`.

## Limits

Open interest, funding, and liquidation panes use the newest 1,000 stored rows from the first loaded candle. A long chart can be empty on the left when more samples exist than that cap. Book shading is the live 30-minute heatmap, not years of depth. The liquidation map behind the candles uses the same 12h–90d grid as `/liquidations`. Open levels use that same window and stay at their birth price; the heatmap still reprices the whole book on every column. Both are the leverage model, not a map of forced closes. `/liquidations?view=open` is the same open-level read. The first paint loads 300 candles, about 12 days on the default 1-hour interval, so 30-day and 90-day columns sit off the left until you pan. That first view keeps the latest candle on the right after open interest, funding, and the open-level line add times that fall between candles. The liquidation pockets beside the candles are that same model. The wider recorder samples about 100 USDT perpetuals. Binance books outside the seed list use the top 100 levels and are snapshotted about every 2 minutes. Flashes stay on the seed pairs and coins someone opened. Bybit and Coinbase snapshot the wider list every minute. Bybit's liquidation socket follows that same 100-name list. Startup backfills public futures history for that same liquid list. Recorded liquidation prints stay with the futures archive (about 90 days). Public refill is still about 7 days.
