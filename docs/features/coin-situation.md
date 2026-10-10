# Coin situation

## Problem

The coin page already loads price versus CVD, open interest, and liquidation levels on separate tabs. A paper fill stored the price and the fee, and not the reason the order was placed.

## Behavior

`GET /api/v1/market/situation?symbol=BTCUSDT` returns three clauses from the combined Binance, Bybit, and OKX tape:

- `priceCvd.kind` is `price_up_cvd_down`, `price_down_cvd_up`, or `agrees`.
- `openInterest.flow` is `opened`, `closed`, or `flat` over the last hour.
- `liquidation.side` is `long` when the nearest estimated long pocket below last is closer, `short` when the nearest short pocket above last is closer, or `none`. `distancePct` is that distance.

A clause is omitted when a venue does not cover it. `priceCvdMissing`, `openInterestMissing`, and `liquidationMissing` name that venue. A missing venue is not counted as zero.

The coin page shows the same clauses in English and Turkish, above the tabs. Equities skip the read.

A paper market order, limit, stop, bracket, or OCO stores this read at the moment the order is placed. A later fill keeps that read. The portfolio trade history shows it beside the fill. Nasdaq and BIST orders do not call the futures read.

## Where the code lives

- `backend/internal/domain/situation.go` builds the read.
- `backend/internal/service/market/situation.go` loads CVD, open interest, and liquidation levels.
- `backend/internal/service/portfolio/service.go` stores it on the order.
- `frontend/src/components/molecules/CoinSituation` renders the sentences.
- MCP `get_coin_situation`. Python `get_coin_situation`.

## How to test

```bash
cd backend
go test ./internal/domain/ -run TestBuildCoinSituation
go test ./internal/service/portfolio/ -run 'TestPlaceOrder_StoresCoinSituation|TestPlacePendingOrder_Keeps|TestPlaceOrder_Equity'

cd frontend
npx vitest run src/components/molecules/CoinSituation/CoinSituation.test.tsx
```

Open `/markets/binance/BTCUSDT`. The situation card sits under the stats. Place a paper buy, then open the portfolio trade history. The fill shows the same kind of sentences.

## Limits

The read uses the live combined tape. A venue that is still filling history leaves its clause blank. The stored read is the one at order placement, so a limit that fills later does not replace it with the market at fill time.
