# MCP transport (`internal/transport/mcp`)

Exposes Swyngora market and watchlist HTTP APIs as **MCP tools** for AI agents.

## Tools

| Tool | Purpose |
|------|---------|
| `health` | API liveness |
| `list_exchanges` | Configured venues |
| `get_ticker` | 24h ticker |
| `get_candles` | OHLCV |
| `get_supply` | Supply snapshot |
| `list_spot_markets` | Search/sort spot list |
| `get_indicators` | RSI/EMA |
| `detect_pump_events` | Pump/dump events on one symbol (threshold, interval, lookback) |
| `scan_pump_events` | Scan top-volume symbols for recent pumps |
| `get_watchlist` | Read watchlist |
| `add_watchlist_item` | Add/update watch item |
| `remove_watchlist_item` | Remove watch item |
| `list_price_alerts` | List price alerts for a client |
| `create_price_alert` | Create one-shot above/below price alert |
| `delete_price_alert` | Delete a price alert by id |

## Run (integrated — preferred)

MCP is **embedded in the main API process**. One command:

```bash
cd backend && go run ./cmd/server
# REST:  http://localhost:8080/api/v1/...
# MCP:   http://localhost:8080/mcp   (streamable HTTP)
```

Optional **stdio** adapter only for hosts that cannot speak HTTP MCP:

```bash
# only if you need pure stdio — API should already be running, or use in-process via server
SWYNGORA_API_URL=http://localhost:8080 go run ./cmd/mcp
```

## Tests

```bash
go test ./internal/transport/mcp/...
```

## Design

- **In-process** tools call market/watchlist services (no second process, no self-HTTP hop).
- HTTP mount uses mark3labs streamable MCP at `/mcp`.
- Failures return MCP tool errors (not panics).
- New product features that AI should use **must** add a matching MCP tool (see root `AGENTS.md`).