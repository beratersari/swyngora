# Derivatives board

## Goal

Show mark, funding, open interest, and 24-hour liquidations for the liquid USDT perpetuals in one table.

## Behavior

`GET /api/v1/market/derivatives-board` follows the same liquid USDT names as the futures recorder, capped at 36. The default is 20. `exchange` is `binance`, `bybit`, `okx`, or `all`.

Each row has:

- Mark price from the fresh venue with the most open interest, and which venue that was.
- Predicted funding. When every venue's open interest is also fresh, the rate is weighted by that open interest. Otherwise the fresh rates are averaged and `fundingWeight` is `equal`.
- Open interest in USDT, plus the change over 1 hour and 24 hours.
- Long and short liquidation notional over the last 24 hours.

A combined cell is empty unless every expected venue is fresh for that column. A missing or old venue is named on the row and is not treated as zero. `definite` on the board is true only when every row is complete. The page is `/derivatives`. A coin opens its chart.

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

The universe is at most 36 USDT perpetuals. Open interest and funding are live REST snapshots, cached about 20 seconds. Liquidation totals are the prints this process has seen in the last 24 hours. There is no fourth exchange on this board.
