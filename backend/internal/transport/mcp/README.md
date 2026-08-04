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
| `scan_pump_events` | Scan top-volume symbols for recent pumps (`maxTotalEvents` caps total events; response includes resolved defaults) |
| `get_watchlist` | Read watchlist (optional `ownerClientId` for shared lists) |
| `add_watchlist_item` | Add/update watch item (owner/editor; optional `ownerClientId`) |
| `remove_watchlist_item` | Remove watch item (owner/editor; optional `ownerClientId`) |
| `share_watchlist` | Grant viewer/editor access (owner only; no duplicate grantee) |
| `update_watchlist_share` | Change share role (owner only) |
| `revoke_watchlist_share` | Remove access (owner only) |
| `list_watchlist_shares` | List outgoing shares |
| `list_shared_watchlists` | Lists shared with this client |
| `list_watchlist_audit` | Who changed the list and when |
| `list_price_alerts` | List price alerts for a client |
| `create_price_alert` | Create one-shot above/below price alert |
| `delete_price_alert` | Delete a price alert by id |
| `get_alert_webhook` | Get client webhook URL for alert notifications |
| `set_alert_webhook` | Set http(s) webhook URL (durable outbox on trigger) |
| `delete_alert_webhook` | Clear client webhook URL |
| `create_portfolio` | Create paper portfolio with starting balance |
| `get_portfolio` | Cash, positions, P&L snapshot |
| `place_portfolio_order` | Paper market buy/sell |
| `place_portfolio_pending_order` | Paper limit_buy / limit_sell / stop_loss |
| `list_portfolio_orders` | List pending paper orders |
| `cancel_portfolio_order` | Cancel open pending paper order |
| `list_portfolio_trades` | Paper trade history |
| `create_recurring_buy` | Create paper recurring buy (DCA) plan |
| `list_recurring_buys` | List recurring buy plans |
| `get_recurring_buy` | Get one recurring buy plan |
| `pause_recurring_buy` | Pause a plan |
| `resume_recurring_buy` | Resume a paused plan |
| `delete_recurring_buy` | Delete a plan and its runs |
| `list_recurring_buy_runs` | Execution history for a plan |
| `create_scanner_rule` | Create RSI / MA / volume watchlist scanner rule |
| `list_scanner_rules` | List scanner rules |
| `delete_scanner_rule` | Delete scanner rule |
| `list_scanner_results` | Scanner match history |
| `start_export` | Start JSON/CSV export of watchlist, shares, alerts, backtests |
| `get_export` | Export job status / progress |
| `list_exports` | List recent export jobs |
| `cancel_export` | Cancel pending/running export |
| `preview_import` | Preview restore of an export file (counts only) |
| `confirm_import` | Apply preview with merge or replace |
| `get_import` | Import job status / progress |
| `list_imports` | List recent import jobs |
| `cancel_import` | Cancel previewed/pending/running import |

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