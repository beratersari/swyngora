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
| `volCompare` | DVOL next to 30-day close-to-close realized vol (Binance daily, annualized 365). `gap` is DVOL minus realized in vol points (70 vs 45 → `+25`). `lean` is `implied_richer` / `realized_hotter` / `balanced` |
| `atm` | ATM IV on the expiry nearest 30 days (7–45 DTE preferred); call/put average when both exist |
| `termStructure` | ATM / call / put mark IV and put−call skew on every live expiry |
| `skew` | Headline ATM skew (put IV − call IV), term slope (back − front ATM), and ~5% OTM wing skew |
| `expiries` | OI, volume, PCR, ATM IV, skew, 1-SD expected move, max pain, dealer GEX / flip, and strike payouts by expiry |
| `gamma` | Dealer gamma: net / call / put USD delta per 1% spot move, regime (`long` = stabler, `short` = faster), flip level, and a re-gammad profile |
| `vsMarket` | Best-effort compare vs live spot range and futures funding / account long-short. `findings` is **only** populated when implied vol or put/call lean disagrees |
| `history` | 1h / 4h / 24h / 7d change plus a chart series for open interest, put/call, DVOL, 30-day realized, volGap (DVOL − realized), net dealer GEX, and flip vs index. Windows also say if regime or flip-side changed |
| `moves` | Unusual short-window OI / put-call / IV vs that series' own typical, plus a dealer-gamma or flip-side change. Empty when nothing stands out |

Open interest and volume are in the **base coin**. USD is `coin × underlying`
(or Deribit `volume_usd` when present).

Web: `/options` — BTC / ETH toggle, headline cards (DVOL card shows 30-day realized and the gap in the same place), **important moves** (only when OI / put-call / IV jumps much more than typical, or dealer gamma / flip-side changes), **over-time** chart (OI / put-call / IV overlays DVOL vs realized / GEX / flip for 12h, 48h, 7d), term structure + skew, **dealer gamma** (net GEX + flip + profile), expiry bars. Selecting an expiry (`?expiry=`) shows the 1-SD expected move, max-pain strike, and that expiry's GEX / flip. A vs-market panel appears only when options IV or put/call lean disagrees with spot or futures.

MCP: `get_options_overview`.

Forensics / around tapes on BTC and ETH attach these stored samples to the
before / during / after windows and flag unusual prints — see
[`forensics.md`](forensics.md).

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/options.go`, `options_vol.go`, `options_gamma.go`, `options_history.go`, `options_moves.go` |
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
- ATM IV uses mark IV on listed strikes; DVOL is Deribit's published index. 30-day realized vol is close-to-close log-return stdev on Binance daily candles, annualized with √365, in percent. `volCompare.gap` is DVOL minus that realized value (vol points). History of the gap uses the same daily window as of each DVOL sample. Needs ~20 daily closes; omitted when candles are missing.
- vsMarket `realizedAnn` is still 24h high–low annualized (a short-window disagreement check). It is not the 30-day realized series on `volCompare`.
- Term structure is ATM mark IV by expiry. Skew is ATM put IV minus ATM call IV (positive = puts richer). Wing skew uses the listed strike nearest 5% below / above spot on the 30-day tenor. Not 25-delta risk reversal (book summary has no delta).
- Expected move is one standard deviation: `spot × (ATM IV / 100) × √(DTE/365)`. Not a guarantee.
- Max pain evaluates listed strikes only: payout = Σ callOI × max(S−K, 0) + Σ putOI × max(K−S, 0) using coin OI. Inverse settlement is not modeled.
- Dealer gamma is Black–Scholes from each contract's mark IV (r = q = 0). Book summary has no venue greeks. Call GEX is signed negative and put GEX positive (dealers short calls / long puts). Units are USD of delta for a 1% spot move (`γ × OI × S² × 0.01`). Same-day tenors are floored at 1 hour. Contracts without mark IV are skipped (`skipped`). The flip is interpolated on a re-gammad profile of listed strikes inside ±20% of the index (plus-to-minus crossing nearest spot preferred). Not a published dealer book.
- `vsMarket` uses Binance spot 24h high–low (annualized) vs ATM/DVOL, nearest-expiry expected move vs that range, PCR/skew lean vs futures account long% and funding, and max pain vs the Deribit index. Missing tape is skipped — the options snapshot still returns.
- `history` DVOL series comes from Deribit `get_volatility_index_data` (hourly). 30-day realized vol is computed from Binance daily closes (not stored). Open interest, put/call, and dealer gamma (net GEX, regime, flip, index) are sampled every 5 minutes into `data/options.db` while the backend runs — they start empty on a fresh process. Older rows from before this column existed have no gamma until the next sample.
- `moves` compares the latest 1h / 4h / 24h change to the median of prior same-length windows. A finding fires only when the move is at least a floor (e.g. 5% OI / 0.10 PCR / 1.5 DVOL pts in 1h) **and** at least 2× typical. One finding per series (hottest window). Dealer-gamma regime changes and flip-side crossings (`gamma_to_short` / `gamma_to_long` / `flip_crossed_up` / `flip_crossed_down`) fire when the latest print differs from the print at that window — they do not use a typical-size floor.
- Informational only — not financial advice.
