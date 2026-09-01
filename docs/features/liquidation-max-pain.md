# Liquidation max pain

## Goal

Show the **largest liquidation pockets** above and below last price — the
price with the most estimated size, how far it is, and whether it is longs
or shorts.

This is **not** the hunt target. Hunt also weighs how expensive it is to walk
the spot book. Max pain only ranks by liquidation size.

This is **not** options max pain. It is a futures liquidation-area read
(CoinGlass-style).

## Behavior

`GET /api/v1/market/liquidation-max-pain?symbol=BTCUSDT`

- `exchange=all` (default) returns Binance and Bybit separately. Combined
  `above` / `below` is the **single largest** pocket on that side. Prices are
  never averaged.
- **Above:** shorts that would liquidate if price rises (`side=short`).
- **Below:** longs that would liquidate if price falls (`side=long`).
- Each pocket: `price`, `notional` (modeled + observed), `movePct`, `side`,
  optional `leverage` / `source`.
- `aboveLevels` / `belowLevels` list the next-largest clusters.

Size comes from the same leverage-mix × blended OI model as hunt, plus 24h
observed liquidation clusters. Informational only.

Web: `/liquidations?view=max-pain`

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/liquidation_max_pain.go` |
| Service | `backend/internal/service/market/max_pain.go` |
| HTTP | `GET /api/v1/market/liquidation-max-pain` |
| MCP / AI | `get_liquidation_max_pain` |
| Web | `frontend/src/components/organisms/LiquidationMaxPain` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/transport/http/handler/ -count=1 -run MaxPain
curl "http://localhost:8080/api/v1/market/liquidation-max-pain?symbol=BTCUSDT"
```
