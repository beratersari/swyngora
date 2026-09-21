# Fear and greed index

## Goal

Show a single 0–100 read of whether the crypto tape looks fearful or greedy,
the same kind of headline Binance and CoinGlass publish, calculated from
market data Swyngora already collects. Say which input moved the score the
most, and keep the past scores.

## Behavior

`GET /api/v1/market/fear-greed`

The score is a weighted average of whichever of these inputs are available.
A missing input is left out and the remaining weights are scaled so they
still sum to 100. At least three inputs are required.

| Input | Nominal weight | Fear ← → greed |
|---|---|---|
| 24h breadth (share of liquid coins that are up) | 18 | 0% up → 0, 50% → 50, 100% → 100 |
| BTC 24h move | 16 | −8% → 0, 0% → 50, +8% → 100 |
| BTC 24h range versus its usual range | 14 | 2× usual → 0, usual → 50, calm → 100 |
| OI-weighted predicted funding | 14 | −0.05% → 0, 0 → 50, +0.05% → 100 (longs paying is greed) |
| Binance account long share | 12 | 30% long → 0, 50% → 50, 70% → 100 |
| 24h liquidations (only after this process has watched the full day) | 12 | all longs liquidated → 0, all shorts → 100 |
| Perp bid share within 1% of mid | 8 | 25% bids → 0, 50% → 50, 75% → 100 |
| Aggressive taker buy share | 6 | 25% buys → 0, 50% → 50, 75% → 100 |

`pull` on each factor is how many points that input added to a neutral 50.
`top` is the largest absolute pull. `sharePct` is that input's weight after
missing inputs were dropped.

Levels:

| Score | Level |
|---|---|
| 0–24 | Extreme fear |
| 25–44 | Fear |
| 45–55 | Neutral |
| 56–74 | Greed |
| 75–100 | Extreme greed |

`history` is the stored series (about every 15 minutes, last 7 days returned).
`windows` is the live score versus the stored score 1h / 4h / 24h / 7d ago.
Samples live in `data/feargreed.db` and start when the backend is running.
A fresh process has a live score and an empty history until the first samples land.

Web: a compact chip on Markets (`/markets`). Clicking it opens drivers and past scores. The note that this is our own tape, not the Binance or CoinGlass index, sits behind an info icon in that dialog.
Telegram: `/fear`. MCP: `get_fear_greed`.

Informational only — not financial advice, and not a copy of another vendor's index.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/fear_greed.go` |
| Store | `backend/internal/adapter/feargreedstore` |
| Service | `backend/internal/service/market/feargreed.go` |
| HTTP | `GET /api/v1/market/fear-greed` |
| MCP / AI | `get_fear_greed` |
| Web | Markets page `FearGreedPanel` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/feargreedstore/ ./internal/service/market/ ./internal/transport/http/handler/ ./internal/transport/telegram/ -count=1 -run 'FearGreed|FormatFearGreed'
curl -s 'http://localhost:8080/api/v1/market/fear-greed'
```

Open `http://localhost:5174/markets` and confirm the score, the level, the input that moved it, and (after the server has been up) the past scores.

## Limits

- Crypto tape only (Binance breadth and volatility, combined futures for funding, liquidations, perp depth, and taker flow). Nasdaq and BIST are not part of the score.
- History exists only for the time this process has been sampling. It is not a multi-year vendor series.
- Liquidations and Bybit taker flow are thin until those sockets have been collecting.
- The bands include a neutral middle (45–55) so a score of 50 is not labeled greed or fear.
