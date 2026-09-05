# Feature: Options overview (BTC / ETH)

## Problem / goal

CoinGlass-style **options** data was missing. For BTC and ETH the desk
should show listed **open interest**, **24h volume**, **put/call ratio**,
**implied volatility**, and **expiry distribution**.

This is **not** futures open interest (`/market/open-interest`).

## Behavior

`GET /api/v1/market/options?asset=BTC`

- `asset` is `BTC` or `ETH` (`BTCUSDT` / `ETHUSDT` accepted). Required
  unless `symbol` is sent as the same alias.
- `exchange=deribit` (default; only venue in this slice).
- Source: **Deribit public REST** (no API key). Inverse options only.
- Cached ~30s (`OPTIONS_CACHE_TTL`).

| Field | Meaning |
|---|---|
| `openInterest` / `openInterestUsd` | Sum of listed option OI (base coin / USD) |
| `volume` / `volumeUsd` | 24h traded volume |
| `callOi` / `putOi` | Call vs put OI |
| `putCallOi` | Put OI ÷ call OI (omitted when there are no calls) |
| `putCallVolume` | Put volume ÷ call volume |
| `lean` | `puts` (≥1.2) / `calls` (≤0.8) / `balanced` |
| `dvol` | Deribit 30-day implied vol index (percent) |
| `dvolChange24h` | DVOL point change vs ~24h ago |
| `atm` | ATM IV on the expiry nearest 30 days (7–45 DTE preferred); call/put average when both exist |
| `expiries` | OI, volume, PCR, and share of total OI by expiry (nearest first) |

Open interest and volume are in the **base coin**. USD is `coin × underlying`
(or Deribit `volume_usd` when present).

Web: `/options` — BTC / ETH toggle, headline cards, expiry bars.

MCP: `get_options_overview`.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/options.go` |
| Adapter | `backend/internal/adapter/deribit` |
| Service | `GetOptions` |
| HTTP | `GET /api/v1/market/options` |
| MCP / AI | `get_options_overview` |
| Web | `frontend/src/components/pages/OptionsPage` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/deribit/ ./internal/service/market/ ./internal/transport/http/handler/ -count=1 -run Options
curl -s 'http://localhost:8080/api/v1/market/options?asset=BTC' | jq .
```

## Limits

- BTC and ETH only. No SOL or other underlyings.
- Deribit only (dominant listed-options venue). Not Binance/Bybit/OKX options.
- Not CoinGlass (paid). Not historical IV surfaces or greeks heatmaps.
- ATM IV uses mark IV on listed strikes; DVOL is Deribit's published index.
- Informational only — not financial advice.
