# Squeeze risk

## Goal

For any coin, score **long-squeeze** and **short-squeeze** risk on Binance
USD-M and Bybit linear, say which side is crowded, explain **why** risk is high
or low, and provide one **combined** view from both venues.

A long squeeze is forced long liquidations cascading lower. A short squeeze is
forced short liquidations cascading higher. This is a **model**, not a
prediction or financial advice.

## Behavior

`GET /api/v1/market/squeeze-risk?symbol=BTCUSDT`

- `exchange=all` (default): **Binance** and **Bybit** separately, plus
  `combined` (OI-weighted scores).
- `exchange=binance` or `bybit`: one venue only (no combined block).

Each venue includes:

| Field | Meaning |
|---|---|
| `longSqueeze` / `shortSqueeze` | Score 0–100, level, weighted factors, plain-language reasons |
| `crowdedSide` | `long` / `short` / `balanced` from account long/short share |
| `higherRisk` | Which squeeze score is clearly higher |
| `summary` | One-line read |

### Score drivers (weights)

| Factor | Weight | Signal |
|---|---|---|
| Crowding | 30% | Account long/short share (and whether it is rising) |
| Funding | 25% | Who pays; longs pay fuels long-squeeze, shorts pay fuels short-squeeze |
| OI build | 20% | 1h/4h open-interest % change; rising OI with price direction adds leverage narrative |
| Liq heat | 15% | Recent same-side liquidations as % of OI (cascade already live) |
| Near pocket | 10% | Share of estimated side OI within ~2% (stylized leverage mix) |

Levels: `low` (under 35), `moderate` (35–55), `elevated` (55–70), `high` (70–85), `extreme` (85+).

### Combined

OI-weighted average of venue scores. `dominantVenue` is the larger OI book.
Reasons start with the stronger venue for that side.

### Limits

- Long/short is **account count**, not position size.
- Nearby pockets use an **assumed** leverage mix (same family as liquidation hunt).
- Liquidation heat only counts events this process has seen (and restored from SQLite).
- Works for any futures pair the venues publish; it is not a market-wide scan of every listing in one call.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/squeeze_risk.go` |
| Service | `backend/internal/service/market/squeeze.go` |
| HTTP | `GET /api/v1/market/squeeze-risk` |
| MCP / AI | `get_squeeze_risk` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/
curl "http://localhost:8080/api/v1/market/squeeze-risk?symbol=BTCUSDT"
```

Read `venues[].longSqueeze` / `shortSqueeze` and `combined`.
