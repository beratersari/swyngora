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
- History samples every 5 minutes (`OPTIONS_HISTORY_INTERVAL`) into `data/options.db`.

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
| `termStructure` | ATM / call / put mark IV and put−call skew on every live expiry |
| `skew` | Headline ATM skew (put IV − call IV), term slope (back − front ATM), and ~5% OTM wing skew |
| `expiries` | OI, volume, PCR, ATM IV, skew, 1-SD expected move, max pain, and strike payouts by expiry |
| `vsMarket` | Best-effort compare vs live spot range and futures funding / account long-short. `findings` is **only** populated when implied vol or put/call lean disagrees |
| `history` | 1h / 4h / 24h / 7d change plus a chart series for open interest, put/call, and DVOL |
| `moves` | Unusual short-window OI / put-call / IV vs that series' own typical. Empty when nothing stands out |

Open interest and volume are in the **base coin**. USD is `coin × underlying`
(or Deribit `volume_usd` when present).

Web: `/options` — BTC / ETH toggle, headline cards, **important moves** (only when OI / put-call / IV jumps much more than typical), **over-time** chart (OI / put-call / IV for 12h, 48h, 7d), term structure + skew, expiry bars. Selecting an expiry (`?expiry=`) shows the 1-SD expected move and max-pain strike. A vs-market panel appears only when options IV or put/call lean disagrees with spot or futures.

MCP: `get_options_overview`.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/options.go`, `options_history.go`, `options_moves.go` |
| Adapter | `backend/internal/adapter/deribit`, `backend/internal/adapter/optionsstore` |
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
- Term structure is ATM mark IV by expiry. Skew is ATM put IV minus ATM call IV (positive = puts richer). Wing skew uses the listed strike nearest 5% below / above spot on the 30-day tenor. Not 25-delta risk reversal (book summary has no delta).
- Expected move is one standard deviation: `spot × (ATM IV / 100) × √(DTE/365)`. Not a guarantee.
- Max pain evaluates listed strikes only: payout = Σ callOI × max(S−K, 0) + Σ putOI × max(K−S, 0) using coin OI. Inverse settlement is not modeled.
- `vsMarket` uses Binance spot 24h high–low (annualized) vs ATM/DVOL, nearest-expiry expected move vs that range, PCR/skew lean vs futures account long% and funding, and max pain vs the Deribit index. Missing tape is skipped — the options snapshot still returns.
- `history` DVOL series comes from Deribit `get_volatility_index_data` (hourly). Open interest and put/call are sampled every 5 minutes into `data/options.db` while the backend runs — they start empty on a fresh process.
- `moves` compares the latest 1h / 4h / 24h change to the median of prior same-length windows. A finding fires only when the move is at least a floor (e.g. 5% OI / 0.10 PCR / 1.5 DVOL pts in 1h) **and** at least 2× typical. One finding per series (hottest window).
- Informational only — not financial advice.
