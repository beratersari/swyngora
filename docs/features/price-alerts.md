# Price alerts

## Goal

Price alerts support one-time/repeating modes, webhooks (immediate or hourly digest), and **quiet hours**.

## Quiet hours

| Setting | Meaning |
|---------|---------|
| `timeZone` | IANA name (default `UTC`) |
| `quietHours.enabled` | When true, defer delivery |
| `quietHours.start` / `end` | Local `HH:MM` (24h). If start > end, range crosses midnight (e.g. 22:00–08:00). |

- Fires still **create** pending notifications/digest items during quiet hours.
- `nextAttemptAt` is set to the **end of quiet hours** (not lost on restart).
- Retries also skip quiet windows.
- Digests still seal on the hour; POST waits until quiet hours end if needed.

### API example

```json
PUT /api/v1/alerts/webhook
{
  "url": "https://hooks.example.com/x",
  "deliveryMode": "immediate",
  "timeZone": "Europe/Istanbul",
  "quietHours": { "enabled": true, "start": "22:00", "end": "08:00" }
}
```

## Delivery modes

| `deliveryMode` | Behavior |
|----------------|----------|
| `immediate` | Each fire enqueued; POST after quiet hours allow |
| `hourly_digest` | UTC-hour batch; one POST after seal (+ quiet hours) |

## Alert modes

| `mode` | Behavior |
|--------|----------|
| `one_time` | Fire once → `triggered` |
| `repeating` | Edge-cross re-fire with re-arm |

## Kinds

| `kind` | What it watches | `condition` | `targetPrice` | Default mode |
|--------|-----------------|-------------|---------------|--------------|
| `price` (default) | Last trade/ticker price | `above` / `below` | Price | `one_time` |
| `imbalance` | Live book notional imbalance in ±`rangePct` of mid | `above` (buy) / `below` (sell) | 0.05–0.95 | `repeating` |
| `wall` | Large clustered rest size in that band | `bid` / `ask` / `any` | Min share 0–1 (`0` = any detected wall) | `repeating` |
| `liquidation_feed` | Binance / Bybit liquidation websocket down or stalled | `down` / `gap` / empty | Min down seconds (`0` = 300) | `repeating` |
| `liquidation_cascade` | A coin (or market `symbol=all`) hits a cascade grade | `cascade` (default) / `extreme` / `elevated` | unused | `repeating` |
| `liquidation_notional` | Long / short / total USDT liquidated in a window | `long` / `short` / `both` | USDT amount (e.g. 20000000) | `repeating` |
| `liquidation_level` | Last price or later prints hit a predicted 10x/25x/50x/100x band that clears later-print confidence | `any` (default) / `fair` / `high` | Min confidence 0–1 (`0` = default for that floor) | `repeating` |
| `setup_radar` | Current tape looks like the setup before that coin's own important past moves (Setup Radar / Forensics Now) | `any` (default) / `up` / `down` | Min similarity 40–100 (`0` = 60) | `repeating` |
| `funding` | Live funding rate in percent (0.01 means 0.01%). `exchange=all` fires when **any** of Binance, Bybit, or OKX crosses the level (above uses the highest rate, below the lowest) and records that venue | `above` / `below` | −5 to 5 | `one_time` unless repeating |
| `open_interest` | Current open-interest notional in USDT | `above` / `below` | Positive USDT | `one_time` unless repeating |
| `taker` | 5-minute taker buy share, 0–100 | `above` / `below` | 0–100 | `one_time` unless repeating |
| `fear_greed` | Market-wide fear and greed score. No symbol | `above` / `below` | 0–100 | `one_time` unless repeating |
| `situation` | `leaves`: live coin read leaves the clauses stored when the newest open paper buy filled. A blank newest buy and a flat book stay quiet and are named on the row. `matches`: live coin read matches the clause kinds in `situation`. No open trade is required. A blank live clause is not a match | `leaves` (default) or `matches` | unused | `one_time` unless repeating |

A `leaves` situation alert's last fire is `situationFire` on `GET /api/v1/alerts` and on the webhook `situation_alert.triggered`, beside `changed` (the clause names). It names the entry kind, the later kind, and any missing venue. The alerts table shows that in English and Turkish with the fire time. Repeating re-arm keeps it until the next fire replaces it. The list does not load a live tape. `situationWatch` is `stored`, `blank`, or `flat` when the alert is read.

A `matches` alert stores the selected clause kinds on `situationMatch`. The checker loads the same live coin read as the coin page. Every selected clause must be present with that kind. A clause left off the selection is ignored. A blank live clause is not a match, so a missing venue does not fire the alert. Print dollars can change without a new `prints` side. The day and 4-hour print clocks are not compared. No open paper position is required. `mode=one_time` fires once. `mode=repeating` fires when the match appears, stays quiet while every selected clause still matches, and re-arms when one no longer does. The last fire uses `situationFire` with `comparison.same` true and each clause `status` `met`. The webhook also sends `matched` (the clause names). Listing the alert sets `situationWatch` to `selected` and does not read the paper book. Nasdaq and BIST are rejected. The coin card links here with the definite clauses already filled in. The caller can change them before creating the alert.

```json
POST /api/v1/alerts
{
  "kind": "situation",
  "condition": "matches",
  "exchange": "binance",
  "symbol": "BTCUSDT",
  "mode": "repeating",
  "situation": {
    "priceCvd": "price_up_cvd_down",
    "funding": "long"
  }
}
```

Book kinds are evaluated in the background from the same live local books as `GET /api/v1/market/orderbook` (Binance, Coinbase, Bybit). Repeating book alerts fire when the condition **appears**, stay quiet while it remains true, and re-arm after it clears so the next appearance can fire again.

```json
POST /api/v1/alerts
{
  "kind": "imbalance",
  "exchange": "binance",
  "symbol": "BTCUSDT",
  "condition": "above",
  "targetPrice": 0.2,
  "rangePct": 2
}
```

```json
POST /api/v1/alerts
{
  "kind": "wall",
  "exchange": "coinbase",
  "symbol": "BTC-USD",
  "condition": "bid",
  "targetPrice": 0.15,
  "rangePct": 2
}
```

Repeating book and liquidation alerts fire when the condition **appears**, stay quiet while it remains true, and re-arm after it clears so the next appearance can fire again. A liquidation feed alert does **not** keep sending while Bybit or Binance is still down. A cascade alert payload includes the **exchange** and **coin** that crossed the grade. A live feed with only an old unfilled history hole does not stay in the down state.

```json
POST /api/v1/alerts
{
  "kind": "liquidation_feed",
  "exchange": "bybit",
  "targetPrice": 300
}
```

```json
POST /api/v1/alerts
{
  "kind": "liquidation_cascade",
  "exchange": "all",
  "symbol": "BTCUSDT",
  "condition": "cascade"
}
```

```json
POST /api/v1/alerts
{
  "kind": "liquidation_notional",
  "exchange": "all",
  "symbol": "BTCUSDT",
  "condition": "both",
  "targetPrice": 20000000,
  "window": "5m"
}
```

`liquidation_notional` sums Binance + Bybit when `exchange=all`. Repeating fires when the rolling window first crosses the dollar line, stays quiet while that wave stays above it, and re-arms when the window drops so the next wave can fire. `window` is `1m` / `5m` / `15m` / `1h` (default 5m). The payload includes `importance` and an optional `level` (leverage / side / confidence) when those prints landed in a predicted band — a high-confidence 100x pocket is not the same alert as a low-confidence 10x miss.

```json
POST /api/v1/alerts
{
  "kind": "liquidation_level",
  "exchange": "all",
  "symbol": "BTCUSDT",
  "condition": "high"
}
```

`liquidation_level` fires when last price or later prints sit in a predicted 10x/25x/50x/100x band that clears later-print confidence. `condition=any` (default floor 0.35), `fair` (0.45), or `high` (0.80). An explicit `targetPrice` in 0–1 overrides the floor. Low-confidence bands that keep missing later prints do not fire. The payload `importance` is `high` / `medium` / `low` from that band's confidence.

```json
POST /api/v1/alerts
{
  "kind": "setup_radar",
  "exchange": "all",
  "symbol": "BTCUSDT",
  "condition": "any",
  "targetPrice": 60
}
```

`setup_radar` uses the same Forensics Now compare: last window vs that coin's own past important-move before-windows (3d lookback). `symbol=all` watches the top-volume scan and fires when **any** coin crosses the floor; new coins while the radar is still hot do not re-fire until zero hits pass. `condition=up` / `down` keeps only cases whose similar past after-path usually rose or fell. The checker caches the scan for about 3 minutes so a 30s tick does not rescan 12 coins. Webhook type `setup_radar.triggered` includes `similarity`, `lean`, `afterStory`, `whySimilar`, `compared` (price / volume / takers / oi / book), and `symbols`. From Forensics Now, use **Create alert** on a row (or alert any coin).

Creating a coin alert (`liquidation_notional` or `liquidation_cascade`) **subscribes** that pair on the live Bybit stream if it is not already in the automatic set. It also fills the lookback from each venue's own history (the alert window, or 6h for cascade) so a 5m alert does not wait 5 minutes for live prints. Live and history prints that share venue + symbol + side + time + price + qty count once. A missing history API still leaves the live subscribe in place.

MCP: `create_orderbook_alert`, `create_liquidation_feed_alert`, `create_liquidation_cascade_alert`, `create_liquidation_notional_alert`, `create_setup_radar_alert`. Informational only.

## Webhook security (SSRF)

- Only absolute `http`/`https` URLs without userinfo.
- By default, **private/local destinations are rejected** after DNS resolution (loopback, RFC1918, link-local, CGNAT, cloud metadata hostnames such as `metadata.google.internal`).
- Delivery **does not follow HTTP redirects**.
- URL is re-validated at delivery time.
- `WEBHOOK_ALLOW_PRIVATE=true` opts into loopback/private targets for local testing only.

## Config

| Env | Default |
|-----|---------|
| `ALERTS_DB_PATH` | `data/alerts.db` |
| `ALERT_CHECK_INTERVAL` | `30s` |
| `WEBHOOK_DELIVERY_INTERVAL` | `5s` |
| `WEBHOOK_HTTP_TIMEOUT` | `10s` |
| `WEBHOOK_MAX_ATTEMPTS` | `8` |
| `WEBHOOK_ALLOW_PRIVATE` | `false` |
| `API_AUTH_TOKEN` | empty (open local mode) — when set, protects alert APIs + MCP |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/adapter/alertstore/ ./internal/service/pricealert/ -count=1
```