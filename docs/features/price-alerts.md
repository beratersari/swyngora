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

`liquidation_notional` sums Binance + Bybit when `exchange=all`. Repeating fires when the rolling window first crosses the dollar line, stays quiet while that wave stays above it, and re-arms when the window drops so the next wave can fire. `window` is `1m` / `5m` / `15m` / `1h` (default 5m).

Creating a coin alert (`liquidation_notional` or `liquidation_cascade`) **subscribes** that pair on the live Bybit stream if it is not already in the automatic set. It also fills the lookback from each venue's own history (the alert window, or 6h for cascade) so a 5m alert does not wait 5 minutes for live prints. Live and history prints that share venue + symbol + side + time + price + qty count once. A missing history API still leaves the live subscribe in place.

MCP: `create_orderbook_alert`, `create_liquidation_feed_alert`, `create_liquidation_cascade_alert`, `create_liquidation_notional_alert`. Informational only.

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