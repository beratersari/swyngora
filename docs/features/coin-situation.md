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

The coin page shows the same clauses in English and Turkish, above the tabs. Equities skip the read.

`/derivatives` opens this read under a row whose unusual cell is a standout. `/unusual` opens it under each scan row. The standout stays in the cell. A derivatives row with no standout does not open the read. The request is this same combined symbol read, and it runs when the row opens.

A market order stores this read when it fills. A limit, stop, bracket, or OCO keeps the placement read on the order and stores a new read on the fill. The trade, **Entered as**, and the situation alert use that fill read. A failed read at the fill stays blank and does not keep the placement read. The portfolio trade history shows the fill read beside the fill and says it is the market at the fill. Nasdaq and BIST orders do not call the futures read.

An open crypto position shows two reads. **Entered as** is the read stored when the newest buy that still has remaining quantity filled. Selling that buy, lifo or fifo, moves the line to the newest buy that remains. A buy that stored no read leaves the line off, even when an older lot has one. **Now** is the live `GET /api/v1/market/situation` for that symbol. Between them, one line names the stored clauses whose kind changed, in the same sentences as the card. A clause the entry did not store stays quiet. A clause the live read left blank is named as missing, with the venue, and is not treated as a new opinion. When every stored clause still matches, the line says the entry read is still the live read. The same pair is on the coin Trade tab when the selected book holds the coin. An equity position stays on size and profit.

A sell stores its own read. Trade history compares that read with each buy the sell closed. A buy that stored no read is left off. The same sentences say what changed, or that the entry read was still true at the sell. `GET /api/v1/portfolio/situation-change` returns the open-position comparison, or pass `tradeId` for one fill. `liveOk` false omits the comparison, so a failed live read is not a missing clause. MCP `get_situation_change`. Python `get_situation_change`.

An alert of kind `situation` fires when the live read leaves the clauses stored on that entry. A clause the entry did not store is ignored, so an older fill does not alert only because a new clause appeared. A blank live clause counts as a change of a clause that was stored. The prints clause compares `prints.side` only. A change in the dollar amounts with the same side stays quiet. No open crypto position stays quiet. `mode=one_time` fires once. `mode=repeating` re-arms when the live read matches the entry again. The exchange is the paper book venue (`binance`, `coinbase`, or `bybit`), and the symbol is the pair on that book.

## Where the code lives

- `backend/internal/domain/situation.go` builds the read.
- `backend/internal/service/market/situation.go` loads CVD, open interest, liquidation levels, and the 24-hour print sides.
- `backend/internal/service/portfolio/service.go` stores the placement read on a resting order and a new read on the fill.
- `frontend/src/components/molecules/CoinSituation` renders the sentences.
- `frontend/src/components/molecules/OutlierSituation` loads that read under an opened outlier row.
- MCP `get_coin_situation` and `get_situation_change`. Python `get_coin_situation` and `get_situation_change`.

## How to test

```bash
cd backend
go test ./internal/domain/ -run 'TestBuildCoinSituation|TestSituationLeft|TestLatestOpenLot|TestMergeLiquidationPrintSides'
go test ./internal/adapter/futuresstore/ -run 'TestSumAndBucketLiquidations'
go test ./internal/service/market/ -run 'TestGetCoinSituation_MeasuredPrints'
go test ./internal/service/portfolio/ -run 'TestPlaceOrder_StoresCoinSituation|TestPlacePendingOrder_|TestPlaceOrder_Equity|TestView_EntrySituation|TestEntrySituation_|TestSituationChange_Names'
go test ./internal/service/pricealert/ -run 'TestChecker_Situation'

cd frontend
npx vitest run src/components/molecules/CoinSituation/CoinSituation.test.tsx src/components/molecules/PositionSituation/PositionSituation.test.tsx src/components/organisms/PortfolioPositionsTable/PortfolioPositionsTable.test.tsx src/components/organisms/PortfolioTradesTable/PortfolioTradesTable.test.tsx src/components/organisms/CreateAlertForm/CreateAlertForm.test.tsx
```

Open `/markets/binance/BTCUSDT`. The situation card sits under the stats. Place a paper limit buy below the market, then let it fill after the situation changes. Trade history and **Entered as** show the market at the fill, with the line "Market when this order filled." The placement read stays on the resting order and is not the position read.

## Limits

The read uses the live combined tape. A venue that is still filling history leaves its clause blank. A resting order that fills later stores the market at the fill. Each partial fill stores its own read. The open position shows the newest remaining buy.
