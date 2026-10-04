# Derivatives board

## Goal

Rank the liquid USDT perpetuals in one sortable table: mark, funding, open interest, crowd long, top-versus-crowd, day CVD, print size, liquidation heat, bid share, taker buy share, and 24-hour liquidations.

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
- Print size: percent change of the 15-minute average fill versus the 15 minutes before it. This reads the in-memory tape only. A tape that does not cover the half hour stays empty.
- Liquidation heat: the hottest 5-minute grade versus that venue's own typical rate (`quiet`, `elevated`, `cascade`, `extreme`). It stays empty until the venue has been watched for 6 hours. When the venues disagree, the cell names the hottest one.
- Bid share: resting bid notional within 1% of mid, summed across fresh books.
- Recorded span: how far back stored open interest, the spot book, flow-at-prices, and liquidation prints go for that coin. The cell shows the longest span of each series. The title lists each venue.

A combined cell is empty unless every expected venue is fresh for that column. A missing or old venue is named on the row and is not treated as zero. `definite` on the board is true only when every row is complete. The page is `/derivatives`. It opens on 36 names, and the coin menu can ask for 100. Any column sorts, and a blank cell stays at the bottom. Clicking a row opens that coin's chart.

## Where the code lives

- Domain: `backend/internal/domain/derivatives_board.go`
- Service: `backend/internal/service/market/derivatives_board.go`
- HTTP: `GET /api/v1/market/derivatives-board`
- Web: `frontend/src/components/pages/DerivativesPage`
- MCP / AI: `get_derivatives_board`

## How to verify

`go test` for `domain`, `service/market`, and the derivatives handler. Frontend: `helpers.test.ts` and `DerivativesPage.test.tsx`.

Open `/derivatives`. The table lists the liquid perpetuals. A dash in a combined column means that venue set is not complete. Choosing one venue shows that venue's own numbers.

## Limits

The page can list at most 100 USDT perpetuals, which is the recorder cap. Open interest, funding, long/short, and the 1% perpetual book are live REST snapshots. The board response is cached about 20 seconds. Day CVD and print size do not start a REST history walk: day CVD uses stored minutes plus the live book, and print size uses only the in-memory tape. Liquidation heat needs about 6 hours of watching before a grade is shown. Taker buy share uses each venue's existing 1-hour window: Binance is the public taker series, Bybit is complete after this process has collected that hour, and OKX only has the newest 500 swap trades, so its hour is often incomplete. Liquidation totals are the prints this process has seen in the last 24 hours. The recorded column is stored history, not a live quote. There is no fourth exchange on this board.
