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

MCP: `create_orderbook_alert`. Informational only.

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