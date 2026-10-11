# Coin situation

## Problem

The coin page already loads price versus CVD, open interest, and liquidation levels on separate tabs. A paper fill stored the price and the fee, and not the reason the order was placed.

## Behavior

`GET /api/v1/market/situation?symbol=BTCUSDT` returns seven clauses:

- `priceCvd.kind` is `price_up_cvd_down`, `price_down_cvd_up`, or `agrees`.
- `openInterest.flow` is `opened`, `closed`, or `flat` over the last hour.
- `liquidation.side` is the leverage model. It is `long` when the nearest estimated long pocket below last is closer, `short` when the nearest short pocket above last is closer, or `none`. `distancePct` is that distance.
- `prints.side` is measured force-order notional over the last 24 hours. It is `long` or `short` when that side is at least 60% of the recorded total, `even` when both sides printed and neither reaches 60%, or `quiet` when the recorded total is under 1 USDT. `longNotional` and `shortNotional` are USDT. `window` is `24h`. This fact is set only when Binance, Bybit, and OKX each recorded the whole window: the watch started at or before the open, and no known feed gap meets that window. A gap that ends exactly at the open does not count. The archive sum adds book prints newer than each venue's newest stored row. A missing archive leaves the clause blank and names all three venues. Partial prints are not published as the 24-hour total.
- `printAnchors` is that same measured notional since 00:00 UTC (`day`) and since the current 4-hour open (`4h`, at 00/04/08/12/16/20 UTC). Each anchor is `long`, `short`, `even`, or `quiet` with `longNotional` and `shortNotional` only when all three venues recorded that clock. `complete` is false and `missing` names the venue when the clock was not fully recorded. A partial sum is not shown. A quiet recorded clock is a real zero. A failed clock does not blank the other clock or the 24-hour total.
- `funding.payer` is `long`, `short`, or `none` for the open-interest-weighted predicted rate. `ratePct` is that rate in percent. It is set only when Binance, Bybit, and OKX all contribute.
- `crowd.lean` is `top_more_long`, `top_more_short`, or `even`. `gapPct` is the average of the Binance and OKX top-minus-crowd gaps, in percentage points. Bybit does not publish a top-trader ratio and is not required.
- `size.kind` is whale versus retail on the latest sized combined bar (`whale_buy_retail_sell`, `whale_sell_retail_buy`, `both_buy`, `both_sell`, `whale_buy`, `whale_sell`, `retail_buy`, `retail_sell`, or `flat`). A whale print is at least $100,000.

A clause is omitted when a venue that publishes it does not cover it. The matching `*Missing` list names that venue. A missing venue is not counted as zero.

The coin page shows the same clauses in English and Turkish, above the tabs, including the day and 4-hour print clocks. Equities skip the read. `/liquidations` and the coin liquidation map show those two clocks for the selected book. A single-venue book names only that venue. Combined stays blank until Binance, Bybit, and OKX all recorded the clock.

A paper market order, limit, stop, bracket, or OCO stores this read at the moment the order is placed. A later fill keeps that read. The portfolio trade history shows it beside the fill. Nasdaq and BIST orders do not call the futures read.

An open crypto position shows two reads. **Entered as** is the read stored on the newest buy that still has remaining quantity. Selling that buy, lifo or fifo, moves the line to the newest buy that remains. A buy that stored no read leaves the line off, even when an older lot has one. **Now** is the live `GET /api/v1/market/situation` for that symbol. The same pair is on the coin Trade tab when the selected book holds the coin. An equity position stays on size and profit.

An alert of kind `situation` fires when the live read leaves the clauses stored on that entry. A clause the entry did not store is ignored, so an older fill does not alert only because a new clause appeared. A blank live clause counts as a change of a clause that was stored. The prints clause compares `prints.side` only. A change in the dollar amounts with the same side stays quiet. The day and 4-hour anchors are not part of that comparison. No open crypto position stays quiet. `mode=one_time` fires once. `mode=repeating` re-arms when the live read matches the entry again. The exchange is the paper book venue (`binance`, `coinbase`, or `bybit`), and the symbol is the pair on that book.

## Where the code lives

- `backend/internal/domain/situation.go` builds the read.
- `backend/internal/service/market/situation.go` loads CVD, open interest, liquidation levels, the 24-hour print sides, and the day and 4-hour print clocks.
- `backend/internal/service/portfolio/service.go` stores it on the order.
- `frontend/src/components/molecules/CoinSituation` renders the sentences.
- MCP `get_coin_situation`. Python `get_coin_situation`.

## How to test

```bash
cd backend
go test ./internal/domain/ -run 'TestBuildCoinSituation|TestBuildPrintAnchors|TestAttachPrintAnchors|TestSituationLeft|TestLatestOpenLot|TestMergeLiquidationPrintSides'
go test ./internal/adapter/futuresstore/ -run 'TestSumAndBucketLiquidations'
go test ./internal/service/market/ -run 'TestGetCoinSituation_MeasuredPrints'
go test ./internal/service/portfolio/ -run 'TestPlaceOrder_StoresCoinSituation|TestPlacePendingOrder_Keeps|TestPlaceOrder_Equity|TestView_EntrySituation|TestEntrySituation_'
go test ./internal/service/pricealert/ -run 'TestChecker_Situation'

cd frontend
npx vitest run src/components/molecules/CoinSituation/CoinSituation.test.tsx src/components/molecules/PositionSituation/PositionSituation.test.tsx src/components/organisms/PortfolioPositionsTable/PortfolioPositionsTable.test.tsx src/components/organisms/LiquidationHeatmap/LiquidationHeatmap.test.tsx
```

Open `/markets/binance/BTCUSDT`. The situation card sits under the stats. Place a paper buy, then open the portfolio trade history. The fill shows the same kind of sentences.

## Limits

The read uses the live combined tape. A venue that is still filling history leaves its clause blank. The stored read is the one at order placement, so a limit that fills later does not replace it with the market at fill time.
