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
| `get_orderbook_heatmap` | Resting bid/ask size over time (pre-warmed for live crypto pairs; not executed volume) |
| `get_liquidations` | Rolling 5m/1h/4h/24h long/short futures liquidations (Binance USD-M + Bybit linear) |
| `get_liquidation_overview` | Market-wide 1h/4h/12h/24h long/short totals + coins ranked for a treemap |
| `get_liquidation_levels` | CoinGlass-style price-level bars for one coin, or time bars of total liquidations (`symbol=all`) |
| `get_liquidation_cascade` | Short-burst long/short cascade vs typical (1m/5m/15m); `episodes` = start, duration, long/short notional, price; combined when both venues fire together |
| `scan_liquidation_cascades` | Market cascade plus coins currently bursting; `both` when the same side fires on both venues |
| `get_open_interest` | Current futures OI plus 5m/1h/4h/24h change (Binance USD-M + Bybit linear); includes `funding` |
| `get_funding_rate` | Predicted next perpetual funding plus recent settlements (Binance USD-M + Bybit linear) |
| `get_funding_arb` | Long cheaper-funding venue / short richer venue; sized after-fee payout + spot-perp gap |
| `scan_funding_arb` | Rank liquid USDT coins that beat fees on published settlements |
| `get_funding_arb_history` | Past after-fee winning stretches from settled Binance/Bybit prints (first clock is entry only; flip clock still pays the old sides) |
| `create_funding_arb_watch` | Follow the scan (omit symbol) or one pair; notify when after-fee net ≥ minProfit |
| `list_funding_arb_watches` | List funding-arb follow watches |
| `get_funding_arb_watch` | Get one funding-arb follow watch |
| `update_funding_arb_watch` | Change minProfit and other settings without deleting |
| `pause_funding_arb_watch` | Pause a follow without deleting it |
| `resume_funding_arb_watch` | Resume a paused follow |
| `delete_funding_arb_watch` | Delete watch and its signals |
| `list_funding_arb_signals` | List min-profit crossings (open/closed/all) |
| `get_long_short_ratio` | Account long/short ratio plus recent 5m history (Binance USD-M + Bybit linear) |
| `get_futures_history` | Durable stored OI / funding / long-short / liquidation history |
| `estimate_liquidation_hunt` | Hypothetical per-venue hunt: spot size to reach liq zones + rough desk result |
| `get_liquidation_heatmap` | Price × time liquidation intensity (12h/24h/3d/7d; Binance, Bybit, combined; own-venue prices; 1h/4h/12h hit review) |
| `get_squeeze_risk` | Long/short squeeze risk scores (0–100) per venue + OI-weighted combined |
| `get_positioning` | Price+OI regime: long/short buildup, unwinding, covering + market combined |
| `get_venue_divergence` | Binance vs Bybit: same/opposite, which metrics differ and why |
| `get_taker_flow` | Aggressive futures buy vs sell volume (5m/1h/4h) per venue + combined |
| `get_cvd` | Spot and futures CVD versus price; 15m/1h/4h/24h change; venue split, spot-vs-futures split, divergence duration |
| `get_volume_profile` | Volume by price (POC + 70% value area, buy/sell); Binance and Bybit separately plus combined; `window` or `startTime`/`endTime` |
| `get_around` | What happened around a time: before / during / after (price, volume, VWAP, vs typical, POC, sweeps, stored book/futures) |
| `compare_around` | Compare two times for the same coin: how the two moves differed (price, volume, book, OI, sweeps) |
| `find_around_moves` | Find strong up/down legs and show what happened during each (around tape) |
| `find_around_precursors` | What often changed before those moves, including conditions that fire together and whether they lean up or down |
| `find_around_similar` | Past important-move setups; matching-bar after horizons with similarity bands |
| `get_vwap` | Volume-weighted average price from a start time (or window) to now; last vs VWAP; Binance, Bybit, combined |
| `get_absorption` | Large market buys/sells vs little price move; absorbing side + strength (15m/1h/4h/24h + current run) |
| `get_liquidity_sweeps` | Poke through a prior high/low that comes back; level, excursion, reclaim time, volume |
| `get_volume_surge` | Current 5m/15m/1h volume vs that coin's typical (median); buy/sell split |
| `scan_volume_surges` | Rank coins whose volume is much higher than typical |
| `get_rsi_heatmap` | Ranked Wilder RSI scatter for top listed pairs (stables omitted) |
| `get_basis` | Perp vs spot/index premium or discount, trend, funding/OI read, venue agreement |
| `get_price_correlation` | How similarly a coin moves with BTC and ETH (1h / 4h / 24h) |
| `get_market_breadth` | How many followed coins are up vs down (1h / 4h / 24h), plus BTC/ETH vs the pack |
| `get_price_volatility` | How much a coin moved over 1h / 4h / 24h vs its normal range and vs BTC/ETH |
| `get_market_snapshot` | Price, volume, mcap, OI, funding, LS, and taker buy/sell together (1h / 4h / 24h) |
| `get_support_resistance` | Support/resistance from price, volume, and the order book, plus breakout score |
| `get_whale_trades` | Largest recent aggressive buys/sells and liquidations, biggest first |
| `get_orderbook_history` | Stored spot book at a time (levels, spread, liquidity, walls) |
| `compare_orderbook_history` | Which price levels gained or lost liquidity between two times |
| `get_orderbook_icebergs` | Same-price clip eaten then refilled (bid and ask) |
| `estimate_market_impact` | Walk live depth for a simulated market buy/sell; impact only when a new best remains |
| `get_candles` | OHLCV |
| `get_supply` | Supply snapshot |
| `get_holders` | Holder count, concentration, top wallets (CMC → Coin Metrics → GeckoTerminal → Ethplorer → Routescan → Tronscan; CryptoID for some UTXO coins). `label` is a public attribution when known |
| `get_asset_profile` | Logo URL, listing date, and published token contracts |
| `list_spot_markets` | Search/sort spot/equity list (`binance` / `coinbase` / `bybit` / `nasdaq` / `bist`) |
| `list_delist_schedule` | Scheduled spot delists per venue (halt `delistTime` + `announcedAt` when the notice was published) |
| `get_post_delist` | Off-venue last + candles after this venue halted the pair (other listed venue or CoinGecko USD) |
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
| `get_portfolio` | Cash, positions, P&L snapshot (optional `portfolioId`; required when the client has 2+ books) |
| `deposit_portfolio_cash` | Add virtual cash (not trading P&L) |
| `transfer_portfolio_cash` | Move cash between your own books (owner only; not deposit/withdrawal) |
| `withdraw_portfolio_cash` | Withdraw available virtual cash |
| `list_portfolio_cash_movements` | Deposit/withdraw history (optional `portfolioId`) |
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
| `create_recurring_buy` | Create named paper recurring buy (schedule, DST-aware timeZone, maxPrice, budget, endDate) |
| `update_recurring_buy` | Update recurring buy name, amount, schedule, timezone/clock, maxPrice, budget, or end |
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
| `create_price_diff_watch` | Track a coin on chosen venues; open only when a fresh-book fill meets minProfit for minDurationSec |
| `list_price_diff_watches` | List price-diff watches |
| `get_price_diff_watch` | Get one price-diff watch |
| `update_price_diff_watch` | Change notional, minProfit, minDurationSec, fees, or exchanges; resets duration timer |
| `pause_price_diff_watch` | Pause; close open opportunities; stop searching |
| `resume_price_diff_watch` | Resume; duration timer starts from zero |
| `delete_price_diff_watch` | Delete watch and opportunities |
| `list_price_diff_opportunities` | List open/closed opportunities |
| `get_price_diff_opportunity` | Get one opportunity |
| `quote_price_diff` | Walk two live spot books for a size: avg buy/sell, slippage, profit after fees, usable money, max size |
| `quote_price_diff_opportunity` | Same quote for a stored opportunity (uses that watch's fees) |
| `scan_price_diff_quotes` | Rank every venue pair at one size; missing books listed as unavailable; optional minProfitPct / minProfitAmount |
| `quote_price_diff_watch` | Same scan using a stored watch's symbol and fees |
| `create_scanner_rule` | Create watchlist scanner rule (conditions + matchMode all\|any) |
| `list_scanner_rules` | List scanner rules |
| `update_scanner_rule` | Enable/disable a rule or edit conditions, periods, and thresholds |
| `delete_scanner_rule` | Delete scanner rule |
| `list_scanner_results` | Scanner match history, confluence setups, hits24h |
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