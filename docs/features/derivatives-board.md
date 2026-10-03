# Derivatives board

## Goal

Rank the liquid USDT perpetuals in one sortable table: mark, funding, open interest, crowd long, taker buy share, and 24-hour liquidations.

## Behavior

`GET /api/v1/market/derivatives-board` follows the same liquid USDT names as the futures recorder. The recorder keeps 100 names. The board opens on 36 and accepts up to 100. `exchange` is `binance`, `bybit`, `okx`, or `all`.

Each row has:

- Mark price from the fresh venue with the most open interest, and which venue that was.
- Predicted funding. When every venue's open interest is also fresh, the rate is weighted by that open interest. Otherwise the fresh rates are averaged and `fundingWeight` is `equal`.
- Open interest in USDT, plus the change over 1 hour and 24 hours.
- Crowd long percent: the account long share. When every venue's open interest is also fresh, the shares are weighted by that open interest. Otherwise they are averaged. A 5-minute print older than 10 minutes is old.
- Taker buy share over the last hour. Notionals are summed across venues. An hour that is not complete stays blank.
- Long and short liquidation notional over the last 24 hours.
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

The page can list at most 100 USDT perpetuals, which is the recorder cap. Open interest, funding, and long/short are live REST snapshots. The board response is cached about 20 seconds. Taker buy share uses each venue's existing 1-hour window: Binance is the public taker series, Bybit is complete after this process has collected that hour, and OKX only has the newest 500 swap trades, so its hour is often incomplete. Liquidation totals are the prints this process has seen in the last 24 hours. The recorded column is stored history, not a live quote. There is no fourth exchange on this board.
