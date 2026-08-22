# Equities adapter (Nasdaq + BIST)

Public cash-equity quotes for **Nasdaq** (`nasdaq`, USD) and **Borsa Istanbul** (`bist`, TRY).

## Data source

Public endpoints only (no API key, no paid plan):

| Venue | List / last / mcap | Candles / single ticker |
|-------|--------------------|-------------------------|
| Nasdaq | Nasdaq.com screener download | Yahoo spark + chart |
| BIST | TradingView public Turkey scanner (`scanner.tradingview.com/turkey/scan`) | Yahoo spark + chart (`THYAO.IS`) |

BIST falls back to the Bigpara ticker board + Yahoo spark if the scanner is down. Spark has no market-cap field, so that fallback usually omits mcap.

## Behavior

- Nasdaq list is the public Nasdaq.com screener tape (thousands of names) and includes **market cap**.
- BIST list is the public Turkey scanner (~640 names) and includes **TRY market cap** plus a sector tag (Finance, Transportation, …). Names such as `LINK`, `QUICK`, and `BERA` are Borsa Istanbul companies, not crypto pairs.
- BIST tickers are stored as `THYAO` and requested from Yahoo as `THYAO.IS`.
- Candles map Swyngora intervals onto Yahoo `1m` / `5m` / `15m` / `60m` / `1d` / `1wk` / `1mo`.
- No public order book: `GetOrderBook` returns not found.
- The market service does **not** apply Binance product tags or crypto circulating-supply mcap to equity rows.
- **Ticker / candles:** cache TTL only. An upstream miss after expiry returns an error — last-good is not served as a live mark (paper fills and alerts).
- **Spot list:** last-good on screener failure until `Cleanup()` (wired from `cmd/server` with the crypto cache tick).

## Tests

```bash
cd backend
go test ./internal/adapter/equities/ -count=1
```
