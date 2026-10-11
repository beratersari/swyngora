# Futures funding rate

## Goal

Show the **current (predicted next) funding rate** and **recent settled history**
for a coin on Binance USD-M, Bybit linear, and OKX USDT-swap, next to open
interest, so a user or the AI can ask “are longs paying to hold BTC?”
Hyperliquid is a separate venue. Its published rate is hourly. `exchange=hyperliquid`
returns that rate alone. It is not folded into `exchange=all`, and it is not
part of the OI-weighted pressure, because that blend is the 8-hour venues.

Venue rates stay separate. Combined `exchange=all` also exposes **one
OI-weighted pressure** so an 8B book at +0.01% is not treated the same as a
1B book at −0.01%.

## Behavior

- `GET /api/v1/market/funding-rate?symbol=BTCUSDT`
  - `exchange=all` (default) returns **each venue separately**
  - `exchange=binance`, `bybit`, `okx`, or `hyperliquid` hoists `current` + `history` to the top level. Hyperliquid `intervalHours` is 1. The other venues are usually 8.
  - `limit` settled prints (default 12, max 30)
  - `pressure` is predicted funding **weighted by USDT open interest**
    (`sum(rate × OI) / sum(OI)`). Venues with no OI are omitted. `weightPct`
    is each venue's share of that combined OI; `dominantVenue` is the largest
    book. Example: Binance 8B at +0.01% and Bybit 1B at −0.01% → combined
    **+0.0078%** (longs still pay), not zero. A configured venue that is
    absent from the blend blanks combined pressure and is named in
    `pressure.missing`. Venues with no OI stay omitted inside a blend
    that already includes every asked venue.
  - `pressureWindows` is live pressure vs ~1h / 4h / 24h ago (`flipped` /
    `steeper` / `flatter` / `stable`)
  - `pressureHistory` is 5-minute OI-weighted samples over the last 24h
    (from stored predicted funding + OI)
  - `pressureMoves` calls out flips and large steepen/flatten events
- `GET /api/v1/market/funding-arb` sizes a long-one-venue / short-the-other trade
  from the funding gap, after fees and spot-perp. See [`funding-arb.md`](funding-arb.md).
- `GET /api/v1/market/open-interest` includes the same payload on `funding`
- Coin detail **Futures tape** shows the OI-weighted pressure and 1h/24h change
- `current` is the **predicted next** payment (not yet settled; hoisted only
  for a single venue)
- `history` is settled payments, **newest first**
- `rate` is decimal (`0.0001` = 0.01%); `ratePct` is percent
- `payer` is who pays at settlement: `long` (positive rate) or `short` (negative)
- `avgLast3` is the mean of the last three settlements (~24h at 8h)
- Durable predicted + settled samples are also stored in SQLite. Query them with
  `GET /api/v1/market/futures-history?metric=funding`. See [`futures-history.md`](futures-history.md).

Sources (public REST, no API key):

- Binance: `GET /fapi/v1/premiumIndex` + `GET /fapi/v1/fundingRate`
- Bybit: `GET /v5/market/tickers?category=linear` + `GET /v5/market/funding/history`
- OKX: `GET /api/v5/public/funding-rate` + `GET /api/v5/public/funding-rate-history` (`BTC-USDT-SWAP`)

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/funding.go`, `funding_pressure.go` |
| Adapters | `adapter/binance/funding.go`, `adapter/bybit/funding.go`, `adapter/okx/funding.go` |
| Service | `GetFundingRate` (also attached by `GetOpenInterest`) |
| HTTP | `GET /api/v1/market/funding-rate` |
| MCP / AI | `get_funding_rate` (also on `get_open_interest.funding`) |
| Telegram | `/funding <symbol>` and a Funding block on `/oi` |
| Web | Coin detail tape (`TapePanel`) |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ ./internal/transport/http/handler/ ./internal/transport/mcp/
curl "http://localhost:8080/api/v1/market/funding-rate?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/open-interest?symbol=BTCUSDT"
```

## Limits

- Not Coinbase. Not COIN-M / inverse.
- Per-venue rows are never averaged. Combined `all` adds `pressure` (OI-weighted)
  instead of inventing a simple mean of venue rates.
- Interval defaults to 8h when the venue does not publish one.
- Informational only — not financial advice.
