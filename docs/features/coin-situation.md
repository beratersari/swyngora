# Coin situation

## Problem

The coin page already loads price versus CVD, open interest, and liquidation levels on separate tabs. A paper fill stored the price and the fee, and not the reason the order was placed.

## Behavior

`GET /api/v1/market/situation?symbol=BTCUSDT` returns seven clauses:

- `priceCvd.kind` is `price_up_cvd_down`, `price_down_cvd_up`, or `agrees`.
- `openInterest.flow` is `opened`, `closed`, or `flat` over the last hour.
- `liquidation.side` is the leverage model. It is `long` when the nearest estimated long pocket below last is closer, `short` when the nearest short pocket above last is closer, or `none`. `distancePct` is that distance.
- `prints.side` is measured force-order notional over the last 24 hours. It is `long` or `short` when that side is at least 60% of the recorded total, `even` when both sides printed and neither reaches 60%, or `quiet` when the recorded total is under 1 USDT. `longNotional` and `shortNotional` are USDT. `window` is `24h`. This fact is set only when Binance, Bybit, and OKX each recorded the whole window: the watch started at or before the open, and no known feed gap meets that window. A gap that ends exactly at the open does not count. The archive sum adds book prints newer than each venue's newest stored row. A missing archive leaves the clause blank and names all three venues. Partial prints are not published as the 24-hour total.
- `funding.payer` is `long`, `short`, or `none` for the open-interest-weighted predicted rate. `ratePct` is that rate in percent. It is set only when Binance, Bybit, and OKX all contribute.
- `crowd.lean` is `top_more_long`, `top_more_short`, or `even`. `gapPct` is the average of the Binance and OKX top-minus-crowd gaps, in percentage points. Bybit does not publish a top-trader ratio and is not required.
- `size.kind` is whale versus retail on the latest sized combined bar (`whale_buy_retail_sell`, `whale_sell_retail_buy`, `both_buy`, `both_sell`, `whale_buy`, `whale_sell`, `retail_buy`, `retail_sell`, or `flat`). A whale print is at least $100,000.

A clause is omitted when a venue that publishes it does not cover it. The matching `*Missing` list names that venue. A missing venue is not counted as zero.

A live clause other than the liquidation model may include `since`. That time is the first stored minute of the current kind. The samples are the five-minute minutes already kept for CVD, open interest, funding, top versus crowd, and measured prints. The clock stays off when the current kind runs back to the oldest loaded minute, when a minute inside the run cannot be read, or when the minutes jump by more than ten minutes. One missed five-minute sample still counts. The estimated liquidation pocket is not stored as its own minute series, so its `since` stays absent. A paper fill stores the clauses without `since`. The coin card and the position **Now** line add the time as a second sentence, in English and Turkish. **Entered as**, trade history, and the line that names what changed do not. A situation alert still compares kinds only.

The same request accepts optional `at`, a candle close in RFC3339. Omit it for the live read. The coin chart sends the close of a clicked candle. A candle that is still forming is read at the current minute, so the request does not change on every render. The clauses are the stored five-minute minute at or before that time: price versus CVD, open interest over the hour ending then, measured prints over the 24 hours ending then, funding, Binance and OKX top versus crowd, and whale versus retail. A CVD bar covers the five minutes that start at its label, so a close that lands on a boundary uses the bar that ended there. A sample older than ten minutes, or a venue the archive does not have, leaves that clause blank and names the venue. That blank is unreadable, not a new opinion. `liquidation` and `liquidationMissing` stay absent, because the estimated pocket is not stored as its own series. A clock more than about a minute ahead of now is rejected. Nasdaq and BIST do not call it. The card under the chart names the time that was read, in English and Turkish. Clear, or the same candle, removes it. The forensics card on that same click still uses the candle open.

The coin page shows the same clauses in English and Turkish, above the tabs. Equities skip the read.

`/derivatives` opens this read under a row whose unusual cell is a standout. `/unusual` opens it under each scan row. The standout stays in the cell. A derivatives row with no standout does not open the read. The request is this same combined symbol read, and it runs when the row opens.

A market order stores this read when it fills. A limit, stop, bracket, or OCO keeps the placement read on the order and stores a new read on the fill. The trade, **Entered as**, and the situation alert use that fill read. A failed read at the fill stays blank and does not keep the placement read. The portfolio trade history shows the fill read beside the fill and says it is the market at the fill. Nasdaq and BIST orders do not call the futures read.

An open crypto position shows the live read once, under **Now**. Every remaining buy is listed above it, newest first. Each line has the remaining size, the fill time, and the clauses stored on that buy. Under those clauses, the same sentences name which stored clauses the live read has left. A clause the buy did not store stays quiet. A clause the live read left blank is named as missing, with the venue, and is not treated as a new opinion. When every stored clause still matches, the line says the entry read is still the live read. A buy that stored no read shows its size and time and says so. **Entered as** is the newest buy. `entrySituation`, the top-level comparison, and a situation alert follow that buy. An older buy stays on the list when a newer buy stored nothing. Selling the newest buy, lifo or fifo, moves **Entered as** to the newest buy that remains. The same list is on the coin Trade tab when the selected book holds the coin. An equity position stays on size and profit. `GET /api/v1/portfolio` returns `openReads` without the live comparison. `GET /api/v1/portfolio/situation-change` adds that comparison when `liveOk` is true.

The paper desk also adds those closed slices up. `GET /api/v1/portfolio/situation-record` groups them by the kind stored on the buy. Each row has the number of closes, how many of those had realized profit above zero, and the sum of that profit. A close with several stored clauses is counted once and appears on each of those rows, so the rows are not one total. A crypto close whose buy stored no read is `skipped`. An equity close is left out. An open position is not included. The same sentences are on the portfolio page, in English and Turkish. MCP `get_situation_record`. Python `get_situation_record`.

A sell stores its own read. Trade history compares that read with each buy the sell closed. A buy that stored no read is left off. The same sentences say what changed, or that the entry read was still true at the sell. `GET /api/v1/portfolio/situation-change` returns the open-position comparison, or pass `tradeId` for one fill. `liveOk` false omits the comparison, so a failed live read is not a missing clause. MCP `get_situation_change`. Python `get_situation_change`.

An alert of kind `situation` fires when the live read leaves the clauses stored on that entry. A clause the entry did not store is ignored, so an older fill does not alert only because a new clause appeared. A blank live clause counts as a change of a clause that was stored. The prints clause compares `prints.side` only. A change in the dollar amounts with the same side stays quiet. No open crypto position stays quiet. `mode=one_time` fires once. `mode=repeating` re-arms when the live read matches the entry again. The exchange is the paper book venue (`binance`, `coinbase`, or `bybit`), and the symbol is the pair on that book.

## Where the code lives

- `backend/internal/domain/situation.go` builds the read. `situation_held.go` stamps `since` on the live read and clears it on the copy stored with a fill. `situation_at.go` builds the same clauses at a candle close and leaves the estimated pocket off.
- `backend/internal/service/market/situation.go` loads CVD, open interest, liquidation levels, and the 24-hour print sides. `situation_held.go` reads the stored minutes for the clocks. `situation_at.go` loads those stored minutes up to `at`. It does not subscribe to the live tape and does not write buckets.
- `backend/internal/domain/situation_lots.go` lists every remaining buy, newest first. `backend/internal/service/portfolio/situation_left.go` compares each of those buys with the live read. The situation alert still uses the newest buy.
- `backend/internal/service/portfolio/service.go` stores the placement read on a resting order and a new read on the fill. That stored copy has no `since`.
- `backend/internal/domain/situation_record.go` groups closed slices by that stored read. `backend/internal/service/portfolio/situation_record.go` loads the book's sold lots.
- `frontend/src/components/molecules/CoinSituation` renders the sentences.
- `frontend/src/components/molecules/PositionSituation` lists every remaining buy, then the live read, on the portfolio position and the coin Trade tab.
- `frontend/src/components/organisms/CoinCandleSituation` renders the stored read under a clicked candle.
- `frontend/src/components/molecules/OutlierSituation` loads that read under an opened outlier row.
- MCP `get_coin_situation` takes optional `at`. Python `get_coin_situation` does the same. MCP `get_situation_change` and `get_situation_record`. Python `get_situation_change` and `get_situation_record`.

## How to test

```bash
cd backend
go test ./internal/domain/ -run 'TestBuildCoinSituation|TestBuildSituationAt|TestSituationLeft|TestLatestOpenLot|TestOpenLotReads|TestWithOpenLotComparisons|TestMergeLiquidationPrintSides|TestHeldSince|TestStampSituationHeld|TestWithoutSituationHeld|TestBuildSituationRecord'
go test ./internal/adapter/futuresstore/ -run 'TestSumAndBucketLiquidations'
go test ./internal/service/market/ -run 'TestGetCoinSituation'
go test ./internal/transport/http/handler/ -run 'TestGetCoinSituation'
go test ./internal/transport/mcp/ -run 'TestAPIClient_GetCoinSituation|TestBackend_GetCoinSituation'
go test ./internal/service/portfolio/ -run 'TestPlaceOrder_StoresCoinSituation|TestPlacePendingOrder_|TestPlaceOrder_Equity|TestView_EntrySituation|TestEntrySituation_|TestSituationChange_Names|TestSituationChange_ListsEvery|TestSituationRecord_'
go test ./internal/service/pricealert/ -run 'TestChecker_Situation'

cd frontend
npx vitest run src/components/molecules/CoinSituation/CoinSituation.test.tsx src/components/organisms/CoinCandleSituation src/components/molecules/PositionSituation/PositionSituation.test.tsx src/components/organisms/PortfolioPositionsTable/PortfolioPositionsTable.test.tsx src/components/organisms/PortfolioTradesTable/PortfolioTradesTable.test.tsx src/components/organisms/CreateAlertForm/CreateAlertForm.test.tsx src/components/pages/CoinDetailPage/CoinDetailPage.test.tsx
```

`TestGetCoinSituation` also matches `TestGetCoinSituationAt`.

Open `/markets/binance/BTCUSDT`. The situation card sits under the stats. Click a candle. The card under the chart reads the stored minutes at that candle's close and says the estimated pocket is not part of the read. Clear removes it. An equity chart does not show it. Place a paper limit buy below the market, then let it fill after the situation changes. Trade history and **Entered as** show the market at the fill, with the line "Market when this order filled." The placement read stays on the resting order and is not the position read. Place a second paper buy. The Trade tab and the portfolio position list both buys, newest first, each with its size, fill time, and stored read. The situation alert still follows the newest buy.

## Limits

The read uses the live combined tape. A venue that is still filling history leaves its clause blank. A resting order that fills later stores the market at the fill. Each partial fill stores its own read. The open position lists every remaining buy. Entered as and the situation alert stay on the newest one. A `since` time needs the stored minutes to reach a different kind. The estimated liquidation pocket has no clock. A past candle read uses only minutes already stored, leaves that pocket off, and does not run for equities.
