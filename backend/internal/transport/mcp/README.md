# MCP transport (`internal/transport/mcp`)

Exposes Swyngora market and watchlist HTTP APIs as **MCP tools** for AI agents.

## Tools

| Tool | Purpose |
|------|---------|
| `health` | API liveness |
| `realtime_stream_info` | WebSocket subscribe/unsubscribe protocol for live prices + paper portfolio |
| `list_exchanges` | Configured venues |
| `get_fx_rates` | Spot FX (USD base) for display conversion |
| `get_ticker` | 24h ticker |
| `get_spot_orderbook` | Live grouped spot bids/asks + walls; `analysis` from ±rangePct of mid; wall `behavior` |
| `analyze_spot_orderbook` | Buy/sell pressure, imbalance, and large walls (short / persistent / suspicious) |
| `analyze_market_orderbook` | Combined Binance+Coinbase+Bybit pressure in one shared price band |
| `get_market_liquidity` | 0–100 liquidity score from ±0.1/0.5/1% depth; per venue + market-wide |
| `get_liquidations` | Rolling 5m/1h/4h/24h long/short futures liquidations (Binance USD-M + Bybit linear) |
| `estimate_market_impact` | Walk live depth for a simulated market buy/sell; impact only when a new best remains |
| `get_candles` | OHLCV |
| `get_supply` | Supply snapshot |
| `list_spot_markets` | Search/sort spot/equity list (`binance` / `coinbase` / `bybit` / `nasdaq` / `bist`) |
| `list_delist_schedule` | Binance scheduled spot delists |
| `get_indicators` | RSI/EMA |
| `analyze_swing` | 4h+1d swing engine (quality gates, ATR stop/TP) |
| `scan_swing_setups` | Watchlist swing scan |
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
| `create_orderbook_alert` | Live imbalance or wall alert (edge re-arm, no re-fire while still true) |
| `delete_price_alert` | Delete a price alert by id |
| `get_alert_webhook` | Get client webhook URL for alert notifications |
| `set_alert_webhook` | Set http(s) webhook URL (durable outbox on trigger) |
| `delete_alert_webhook` | Clear client webhook URL |
| `create_api_key` | Create named `read`/`trade` API key (secret once) |
| `list_api_keys` | List key metadata (no secrets) |
| `revoke_api_key` | Revoke a named API key |
| `create_portfolio` | Create a named paper book (starting balance + optional name) |
| `list_portfolios` | List paper books for a client |
| `rename_portfolio` | Rename a paper book |
| `delete_portfolio` | Delete a paper book and its data |
| `share_portfolio` | Share a book with another client (`viewer` or `trader`) |
| `update_portfolio_share` | Change share role |
| `revoke_portfolio_share` | Remove a share |
| `list_portfolio_shares` | Outgoing shares (owner) |
| `list_shared_portfolios` | Books shared with you |
| `get_portfolio` | Cash, positions, P&L snapshot (optional `portfolioId`) |
| `deposit_portfolio_cash` | Add virtual cash (not trading P&L) |
| `transfer_portfolio_cash` | Move cash between your own books (owner only; not deposit/withdrawal) |
| `withdraw_portfolio_cash` | Withdraw available virtual cash |
| `list_portfolio_cash_movements` | Deposit/withdraw history |
| `get_portfolio_performance` | Equity history + period P&L (1d/1w/1m/3m) |
| `get_portfolio_risk_limits` | Optional risk limits + live status |
| `set_portfolio_risk_limits` | Set daily-loss % and/or max coin weight % |
| `clear_portfolio_risk_limits` | Remove all risk limits |
| `get_paper_trading_costs` | Per-exchange paper fee and slippage rates |
| `place_portfolio_order` | Paper market buy/sell (slipped fill + taker fee; optional idempotencyKey) |
| `place_portfolio_pending_order` | Paper limit_buy / limit_sell / stop_loss / trailing_stop (optional idempotencyKey) |
| `place_portfolio_oco_order` | Paper OCO take-profit + stop-loss (linked; optional idempotencyKey) |
| `place_portfolio_bracket_order` | Paper bracket: entry limit buy + pending TP/SL (optional idempotencyKey) |
| `list_portfolio_orders` | List pending paper orders |
| `get_portfolio_order` | Get one pending order + last price + amend hints |
| `amend_portfolio_order` | Amend open GTC limit/stop price and/or remaining size |
| `cancel_all_portfolio_orders` | Cancel all open paper orders, or one market |
| `cancel_portfolio_order` | Cancel open pending paper order |
| `list_portfolio_trades` | Paper trade history |
| `list_portfolio_lots` | Open (or closed) tax lots for FIFO/LIFO sells |
| `create_recurring_buy` | Create named paper recurring buy (daily/weekly/monthly/interval) |
| `update_recurring_buy` | Update recurring buy name, amount, or schedule |
| `list_recurring_buys` | List recurring buy plans |
| `get_recurring_buy` | Get one recurring buy plan |
| `pause_recurring_buy` | Pause a plan |
| `resume_recurring_buy` | Resume a paused plan |
| `delete_recurring_buy` | Delete a plan and its runs |
| `list_recurring_buy_runs` | Execution history for a plan |
| `create_portfolio_basket` | Named target mix (does not trade) |
| `list_portfolio_baskets` | List saved baskets |
| `get_portfolio_basket` | Basket + live drift preview |
| `update_portfolio_basket` | Update name/targets (no trades) |
| `delete_portfolio_basket` | Delete basket |
| `preview_portfolio_rebalance` | Proposed sells/buys (no trades) |
| `rebalance_portfolio_basket` | User-triggered rebalance |
| `set_margin_mode` | Set isolated or cross (locked if open pos/orders) |
| `place_margin_order` | Paper margin long/short open (market/limit, 1x–10x; optional idempotencyKey) |
| `list_margin_positions` | Open margin positions |
| `close_margin_position` | Full/partial margin close (optional idempotencyKey) |
| `adjust_margin` | Add/remove isolated position margin |
| `repay_margin_debt` | Pay interest then principal without closing |
| `set_margin_brackets` | Stop-loss / take-profit |
| `list_margin_orders` | Margin limit orders |
| `cancel_margin_order` | Cancel margin limit (releases reserve) |
| `list_margin_trades` | Margin trade history |
| `create_price_diff_watch` | Track cross-exchange price gaps after fees |
| `list_price_diff_watches` | List price-diff watches |
| `get_price_diff_watch` | Get one price-diff watch |
| `delete_price_diff_watch` | Delete watch and opportunities |
| `list_price_diff_opportunities` | List open/closed opportunities |
| `get_price_diff_opportunity` | Get one opportunity |
| `create_scanner_rule` | Create RSI / MA / volume watchlist scanner rule |
| `list_scanner_rules` | List scanner rules |
| `delete_scanner_rule` | Delete scanner rule |
| `list_scanner_results` | Scanner match history |
| `start_export` | Start JSON/CSV export of watchlist, shares, alerts, backtests, portfolios |
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

### Security

| Control | Env | Notes |
|---------|-----|--------|
| Shared API token | `API_AUTH_TOKEN` | When set, `/mcp` requires master token or a user `trade` key (`Authorization: Bearer` / `X-API-Key`) |
| Closed account | (in-process tools) | Tools that send `clientId` call `RequireActive` — closed clients get a tool error (HTTP AccountGate cannot read JSON body on `/mcp`) |
| User API key | HTTP identity | `bindMCPTenant` forces tool `clientId` to the key binding (mismatch → error); `create_api_key` / `list_api_keys` / `revoke_api_key` denied for user keys (mirror REST) |
| Disable MCP | `MCP_ENABLED=false` | Do not mount `/mcp` at all |
| Webhooks | (service) | `set_alert_webhook` rejects private/local targets unless `WEBHOOK_ALLOW_PRIVATE=true` |

Leave `API_AUTH_TOKEN` empty only for local single-user development. Do not expose `/mcp` on a public interface without a token.

Optional **stdio** adapter only for hosts that cannot speak HTTP MCP:

```bash
# only if you need pure stdio — API should already be running, or use in-process via server
SWYNGORA_API_URL=http://localhost:8080 go run ./cmd/mcp
# If the API has API_AUTH_TOKEN set, the HTTP client path must send the same token (not yet wired in stdio client — prefer in-process /mcp with auth).
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