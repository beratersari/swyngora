# Feature: Display currency

## Problem / goal

BIST last prices and mcap are **TRY**. Nasdaq (and Coinbase) are **USD**. Crypto venues quote **USDT**. Users need one switch to view those numbers in the same currency.

## Behavior

- Header control: **Native** | **USD** | **TRY** | **EUR**
- Native leaves each venue in its quoted currency (and labels it)
- Conversion uses `GET /api/v1/market/fx` (ECB/Frankfurter, 15m cache)
- `USDT` is treated as `USD`
- Converted: last / open / high / low / quote volume / market cap / chart OHLC / **EMA (and other price-axis MAs)** overlay + latest EMA snapshot
- Not converted: base volume (shares/coins), % change, RSI/indicators, paper-book cash (still USDT), order-book grouping
- Chart uses the **spot** FX rate on historical bars (not historical FX)
- Preference is stored in `localStorage` (`swyngora.displayCurrency`)

## Where the code lives

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/fx.go` |
| Adapter | `backend/internal/adapter/fx/` |
| Service | `backend/internal/service/market/fx.go` |
| HTTP | `GET /api/v1/market/fx` |
| MCP | `get_fx_rates` |
| Web | `CurrencySwitcher` + `DisplayCurrencyProvider` |

## How to test

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/fx/ ./internal/service/market/ -count=1
cd frontend && npx vitest run src/libs/utils/displayCurrency.test.ts src/components/molecules/CurrencySwitcher
curl -s http://127.0.0.1:8080/api/v1/market/fx
```
