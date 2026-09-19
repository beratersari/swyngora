# Futures basis (perp vs spot)

## Goal

Show how far the **perpetual futures price** is from **spot/index** on Binance
USD-M and Bybit linear: dollar gap, percent gap, premium or discount, whether
that gap is getting bigger or smaller, and a short read with funding and open
interest. Also say if both venues agree. Compare **listed dated futures** so
you can see whether the near expiry is richer than the far expiry (backwardation)
or the far expiry is richer (contango), and whether that curve is up or down.

## Behavior

`GET /api/v1/market/basis?symbol=BTCUSDT`

- `last`: perp last vs spot **index** (what you see vs the official spot basket)
- `mark`: mark vs index (what funding is built from)
- `kind`: `premium` | `discount` | `flat` (flat inside ±0.01%)
- `windows` 5m / 1h / 4h: past basis %, change in percentage points, `expanding` /
  `shrinking` / `stable` / `flipped`
- `agreement` (when `exchange=all`): `same` | `opposite` | `mixed`
  (`mixed` also when both sides match but the percent gaps differ by ≥ 0.05 pp)
- `curve` (per venue): listed contracts sorted by days to expiry (perp first).
  `near` / `far` are the shortest and longest dated when two dated contracts
  exist; otherwise perp vs the one dated book. `shape` is `contango` (far
  richer), `backwardation` (near richer), `flat`, or `humped`. `slope` is
  `up` / `down` / `flat`. `richer` is `near` | `far` | `none`.
- Web: `/basis?symbol=BTCUSDT`

Sources: Binance `premiumIndex` + mark/index 5m klines (`indexPriceKlines` uses
`pair`); Bybit linear ticker + mark/index 5m klines. Venue spot last is attached
when the spot pair exists.

To turn the Binance vs Bybit funding gap (plus this basis) into a sized long/short
read, see [`funding-arb.md`](funding-arb.md).

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/basis.go` |
| Adapters | `adapter/binance/basis.go`, `adapter/bybit/basis.go` |
| Service | `backend/internal/service/market/basis.go` |
| HTTP | `GET /api/v1/market/basis` |
| MCP / AI | `get_basis` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ -run Basis
curl "http://localhost:8080/api/v1/market/basis?symbol=BTCUSDT"
```
