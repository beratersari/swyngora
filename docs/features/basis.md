# Futures basis (perp vs spot)

## Goal

Show how far the **perpetual futures price** is from **spot/index** on Binance
USD-M, Bybit linear, and OKX USDT-swap: dollar gap, percent gap, premium or
discount, whether that gap is getting bigger or smaller, and a short read with
funding and open interest. Also say if the venues agree. OKX uses mark versus
the USDT index. Dated-futures curve stays on the venues that list those contracts. Compare **listed dated futures** so
you can see whether the near expiry is richer than the far expiry (backwardation)
or the far expiry is richer (contango), and whether that curve is up or down.

## Behavior

`GET /api/v1/market/basis?symbol=BTCUSDT`

- `last`: perp last vs spot **index** (what you see vs the official spot basket)
- `mark`: mark vs index (what funding is built from)
- `kind`: `premium` | `discount` | `flat` (flat inside ±0.01%)
- `windows` 5m / 1h / 4h: past basis %, change in percentage points, `expanding` /
  `shrinking` / `stable` / `flipped`
- `agreement` (when `exchange=all`): `same` | `opposite` | `mixed` | `unknown`.
  Agreement needs a usable quote from Binance, Bybit, and OKX. A missing or
  unusable quote leaves the comparison `unknown`. `mixed` covers one flat
  venue beside a premium or discount, and the same kind of gap when the
  percent gaps differ by ≥ 0.05 pp.
- `curve` (per venue): listed contracts sorted by days to expiry (perp first).
  `near` / `far` are the shortest and longest dated when two dated contracts
  exist; otherwise perp vs the one dated book. `shape` is `contango` (far
  richer), `backwardation` (near richer), `flat`, or `humped`. `slope` is
  `up` / `down` / `flat`. `richer` is `near` | `far` | `none`.
- `curve.history`: hourly near/far mark spread over up to 7 days (same listed
  contracts as now). Each point has `spreadPct` and `shape`.
- `curve.windows` 1h / 4h / 24h / 7d: past spread %, `change` `flipped` /
  `steeper` / `flatter` / `stable`.
- `curve.moves`: when the book flipped contango ↔ backwardation, or the
  steepest ~4h steepen/flatten that clears 8 bp. Empty when nothing stands out.
- Web: `/basis?symbol=BTCUSDT` — live curve table plus Over time chart and
  those windows/moves.

Sources: Binance `premiumIndex` + mark/index 5m klines (`indexPriceKlines` uses
`pair`); Bybit linear ticker + mark/index 5m klines; OKX USDT-swap last and
mark versus the USDT index. Venue spot last is attached when the spot pair exists.

To turn the Binance vs Bybit funding gap (plus this basis) into a sized long/short
read, see [`funding-arb.md`](funding-arb.md).

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/basis.go`, `basis_curve_history.go` |
| Adapters | `adapter/binance/basis.go`, `adapter/bybit/basis.go`, `adapter/okx/basis.go` |
| Service | `backend/internal/service/market/basis.go` |
| HTTP | `GET /api/v1/market/basis` |
| MCP / AI | `get_basis` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ -run Basis
curl "http://localhost:8080/api/v1/market/basis?symbol=BTCUSDT"
```
