# Derivatives board

## Goal

Rank the liquid USDT perpetuals in one sortable table: mark, funding, open interest, crowd long, top-versus-crowd, day CVD, the current price-versus-CVD run, print size, order count, unusual readings, liquidation heat, bid share, taker buy share, and 24-hour liquidations.

## Behavior

`GET /api/v1/market/derivatives-board` follows the same liquid USDT names as the futures recorder. The recorder keeps 100 names. The board opens on 36 and accepts up to 100. `exchange` is `binance`, `bybit`, `okx`, or `all`.

Each row has:

- Mark price from the fresh venue with the most open interest, and which venue that was.
- Predicted funding. When every venue's open interest is also fresh, the rate is weighted by that open interest. Otherwise the fresh rates are averaged and `fundingWeight` is `equal`.
- Open interest in USDT, plus the change over 1 hour and 24 hours.
- Crowd long percent: the account long share. When every venue's open interest is also fresh, the shares are weighted by that open interest. Otherwise they are averaged. A 5-minute print older than 10 minutes is old.
- Taker buy share over the last hour. Notionals are summed across venues. An hour that is not complete stays blank.
- Long and short liquidation notional over the last 24 hours.
- Top minus crowd, in percentage points. Bybit does not publish a top-trader ratio, so the combined cell uses the venues that do and names Bybit as unsupported. `gapScope` lists who is inside the number.
- Day CVD: aggressive buy minus sell since 00:00 UTC. The cell stays empty until every expected venue's tape reaches that open and the newest bucket is still fresh.
- Price versus CVD: the current run where price and cumulative delta move opposite ways (`price_up_cvd_down` or `price_down_cvd_up`), and how long that run has lasted. When the tape is definite and there is no current run, the cell says the two are in line (`agrees`). A dash means the tape does not cover the read. The cell needs the tape to reach 00:00 UTC, a price on the newest bar, and a newest minute that is still fresh. One venue shows that venue. Combined uses the summed series, not an average of the venue labels, and stays empty unless every expected venue's 24-hour tape is complete. A later bar where price and CVD move the same way ends the run; flat bars after it do not. A 4-hour window that disagrees without a current run is in line, not a run. Prices come from Binance 5-minute candles, then Bybit if Binance has none. A missing price leaves this cell blank and still leaves day CVD in place.
- Print size: percent change of the 15-minute average fill versus the 15 minutes before it. This reads the in-memory tape, or stored flow-at-prices minutes when that tape does not cover the half hour. It does not walk public trade history.
- Order count: `(buy − sell) / (buy + sell)` over those same 15 minutes. Counts are summed only when every expected venue's window is complete and the newest print is still fresh. A venue with no fresh prints stays blank and is not counted as zero.
- Unusual: the farthest high or low reading versus that coin's own stored samples (funding, open-interest change, completed-hour liquidations, completed-day CVD, or hourly spread). The cell names the venue. Venues are not averaged. A coin with no standout stays blank, including when its own history is still too short. Print size is not part of this scan.
- Liquidation heat: the hottest 5-minute grade versus that venue's own typical rate (`quiet`, `elevated`, `cascade`, `extreme`). It stays empty until the venue has been watched for 6 hours. When the venues disagree, the cell names the hottest one.
- Bid share: resting bid notional within 1% of mid, summed across fresh books.
- Recorded span: how far back stored open interest, the spot book, flow-at-prices, and liquidation prints go for that coin. The cell shows the longest span of each series. The title lists each venue.

A combined cell is empty unless every expected venue is fresh for that column. A missing or old venue is named on the row and is not treated as zero. `definite` on the board is true only when every row is complete. The page is `/derivatives`. It opens on 36 names, and the coin menu can ask for 100. Any column sorts, and a blank cell stays at the bottom. Clicking a row opens that coin's chart.

The same page has a heatmap. Rows are the coins already on the board. One control picks the column that paints the cells: funding, open-interest change, crowd long, taker buy, top minus crowd, day CVD, price versus CVD, print size, order count, unusual distance, liquidation heat, or bid share. The largest distance from that column's neutral is first. A cell with no definite value stays blank and is not drawn as zero. Day CVD color is this coin against the largest day CVD on the screen, because the column is a notional. Price versus CVD uses 60 minutes as full strength: price up while CVD falls is one direction, the reverse is the other, and in line stays neutral. The other columns use a fixed scale, so a quiet hour does not paint every cell at full strength. There is no second route and no new request.

## Where the code lives

- Domain: `backend/internal/domain/derivatives_board.go`
- Service: `backend/internal/service/market/derivatives_board.go`
- HTTP: `GET /api/v1/market/derivatives-board`
- Web: `frontend/src/components/pages/DerivativesPage` and `frontend/src/components/organisms/DerivativesHeatmap`
- MCP / AI: `get_derivatives_board`

## How to verify

`go test` for `domain`, `service/market`, and the derivatives handler. Frontend: `DerivativesBoard/helpers.test.ts`, `DerivativesHeatmap/helpers.test.ts`, and `DerivativesPage.test.tsx`.

Open `/derivatives`. The table lists the liquid perpetuals. A dash in a combined column means that venue set is not complete. Choosing one venue shows that venue's own numbers. Heatmap uses those same rows: pick a column, confirm a withheld cell stays blank, and confirm the strongest reading is first.

## Limits

The page can list at most 100 USDT perpetuals, which is the recorder cap. It opens on 36. Choosing 100 ranks the recorded set, including order count and unusual, on this board. Open interest, funding, long/short, and the 1% perpetual book are live REST snapshots. The board response is cached about 20 seconds. Day CVD, the price-versus-CVD run, print size, and order count do not start a REST history walk: day CVD and the run share stored minutes plus the live book, and print size and order count use the in-memory tape or, when that tape does not cover the window, stored flow-at-prices minutes. Unusual reads the same stored samples as the unusual scan and does not merge venues. Liquidation heat needs about 6 hours of watching before a grade is shown. Taker buy share uses each venue's existing 1-hour window: Binance is the public taker series, Bybit is complete after this process has collected that hour, and OKX only has the newest 500 swap trades, so its hour is often incomplete. Liquidation totals are the prints this process has seen in the last 24 hours. The recorded column is stored history, not a live quote. There is no fourth exchange on this board.
