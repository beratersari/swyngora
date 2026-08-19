# Feature: Multi-agent AI assistant

## Problem / goal

Users need explanations and multi-step market analysis, not only raw JSON from the API. Swyngora provides a **hierarchical multi-agent** assistant that plans, gathers tool-backed data, and synthesizes answers — without inventing prices.

## Architecture

LangChain **ReAct orchestrator**: the model chooses specialists from the user prompt. **No tool is mandatory.** Greeting or a self-contained question → zero tools.

```text
User → Orchestrator (create_agent)
         optional, only if the question needs it:
         ├─ market_tape_agent  → ticker, candles, indicators, FX, lists, delists, holders
         ├─ market_book_agent  → book, liquidations, impact, pumps, swing
         ├─ paper_desk_agent   → paper books / orders / margin
         ├─ account_agent      → watchlist, alerts, keys, export/import
         ├─ web_agent          → allowlisted RSS + wiki + Gecko + SEC EDGAR / KAP
         ├─ x_agent            → StockTwits / HN (weak)
         └─ analyst_agent      → synthesis only after several specialists
```

| Component | Path |
|-----------|------|
| Orchestrator / desk graph | `ai/src/swyngora_ai/graph/` |
| Specialists | `ai/src/swyngora_ai/agents/` |
| Market tools (packed) | `ai/src/swyngora_ai/tools/market_http.py`, `packs.py` |
| Sources / grounding | `ai/src/swyngora_ai/sources/`, `grounding.py` |
| FinMem | `ai/src/swyngora_ai/memory/` |
| Go MCP server | `backend/cmd/mcp`, `backend/internal/transport/mcp` |
| LLM factory | `ai/src/swyngora_ai/llm/factory.py` (Ollama \| Grok) |

## Behavior

- Market numbers must come from tools (ticker, candles, supply, holders, indicators, spot).
- Paper sell realized PnL uses tax lots (`list_portfolio_lots`, `lotMethod` fifo|lifo) after the sell fee; buy lot cost includes the buy fee. Per-exchange rates: `get_paper_trading_costs`.
- Live UI updates use the backend WebSocket (`realtime_stream_info` / `GET /api/v1/realtime`); tools remain request/response.
- Social/X results are labeled weak and incomplete.
- Coin/project questions dispatch **web_agent** (`web_research` + `web_news`) and optionally **x_agent**; public **URLs** return as `references` on the chat payload and render as source cards in the web UI.
- Answers include “not financial advice” framing for market questions.
- Session memory is **namespaced by `clientId` + `sessionId`**. Optional FinMem SQLite (`AI_MEMORY_PATH`) stores daily notes + last tape timestamp (TTL 5m). `reset` uses the same tenant key.
- HTTP/Telegram pass `clientId`; Python tools bind that id (and send `SWYNGORA_API_TOKEN` / `X-Client-Id`) so the model cannot switch tenants. Reserved ids (`ai-assistant`, `http-default`, `anonymous`) are rejected by the backend.
- The Go AI proxy also passes `canTrade` / `canManageKeys` from the authenticated identity. User **read** keys keep AI chat but mutating tools return 403; user keys never get key-admin tools even with trade permission.
- Web `/ai` streams `POST /api/v1/ai/chat/stream` and shows a **Process** timeline (status / think / tools / results) as each step happens. The list stays open while working, then collapses to a one-line summary so the answer stays readable. Non-stream `POST /api/v1/ai/chat` remains as fallback.
- Telegram `/ask` uses the same stream for live progress edits.

## How to run

```bash
# One backend process: REST + MCP (/mcp)
cd backend && go run ./cmd/server

# AI CLI (separate process — needs LLM)
cd ai && uv sync && source .venv/bin/activate
export AI_LLM_PROVIDER=ollama   # or grok + XAI_API_KEY
export OLLAMA_MODEL=qwen2.5     # default; llama3.2 skips tools
# export GROK_REASONING_EFFORT=low   # Grok only: none|low|medium|high
export SWYNGORA_API_URL=http://localhost:8080
# export SWYNGORA_API_TOKEN=...   # same as backend API_AUTH_TOKEN when set
# export AI_SERVICE_TOKEN=...     # shared with Go proxy for :8090
# export AI_MEMORY_PATH=data/ai-memory.db
swyngora-ai "BTC RSI on binance 1h and recent news"
```

## Tests

```bash
cd backend && go test ./internal/transport/mcp/...

cd ai
source .venv/bin/activate   # after uv sync
pytest -q
ruff check . && ruff format --check .
ty check
```

## Market / watchlist tools

Market specialist tools mirror Go MCP (ticker, **FX rates** `get_fx_rates`, **spot order book** / **order-book analysis** / **market-wide book** / **market impact** / **order heatmap**, **order-book alerts**, candles, indicators, pumps, paper portfolio, scanner, **watchlist sharing**, **API keys**).  
Impact: `estimate_market_impact` walks live depth for a market buy/sell (`quantity` or `notional`). Liquidity: `get_market_liquidity` (0–100 + weaker side, per venue + market-wide). Order heatmap: `get_orderbook_heatmap` (resting bid/ask size over the last few minutes). Liquidations: `get_liquidations` (5m/1h/4h/24h long vs short, Binance USD-M + Bybit linear). Open interest: `get_open_interest` (current + 5m/1h/4h/24h change, Binance USD-M + Bybit linear; includes `funding`). Funding: `get_funding_rate` (predicted next rate + recent settlements). Long/short: `get_long_short_ratio` (share of accounts long vs short). History: `get_futures_history` (durable stored OI / funding / long-short / liquidations). Hunt: `estimate_liquidation_hunt` (hypothetical per-venue spot size to reach liq zones + rough desk result; not evidence of exchange behavior). Squeeze: `get_squeeze_risk` (long/short squeeze scores 0–100 per venue + combined). Positioning: `get_positioning` (price+OI long/short buildup, unwinding, covering + market combined). Divergence: `get_venue_divergence` (Binance vs Bybit same/opposite). Taker flow: `get_taker_flow` (aggressive buy vs sell volume 5m/1h/4h). CVD: `get_cvd` (spot and futures CVD versus price; 15m/1h/4h/24h change; venue split, spot-vs-futures split, divergence duration). Basis: `get_basis` (perp vs spot premium/discount). Correlation: `get_price_correlation` (vs BTC and ETH, 1h/4h/24h). Breadth: `get_market_breadth` (how many followed coins are up vs down, 1h/4h/24h). Volatility: `get_price_volatility` (range vs normal and vs BTC/ETH). Snapshot: `get_market_snapshot` (price, volume, mcap, OI, funding, LS, taker together). Levels: `get_support_resistance` (S/R + breakout score). Whales: `get_whale_trades` (largest recent buys/sells/longs/shorts, biggest first, vs market cap). Book history: `get_orderbook_history` / `compare_orderbook_history` (stored books, which levels gained or lost liquidity). Icebergs: `get_orderbook_icebergs` (same-price clip eaten then refilled, bid and ask). Walls include `behavior` (`short` / `persistent` / `suspicious` / `iceberg`). See `docs/features/order-book.md`, `docs/features/icebergs.md`, `docs/features/liquidations.md`, `docs/features/open-interest.md`, `docs/features/funding-rate.md`, `docs/features/long-short-ratio.md`, `docs/features/futures-history.md`, `docs/features/liquidation-hunt.md`, `docs/features/squeeze-risk.md`, `docs/features/positioning.md`, `docs/features/venue-divergence.md`, `docs/features/taker-flow.md`, `docs/features/cvd.md`, `docs/features/basis.md`, `docs/features/correlation.md`, `docs/features/breadth.md`, `docs/features/volatility.md`, `docs/features/snapshot.md`, `docs/features/levels.md`, `docs/features/whales.md`, and `docs/features/book-history.md`.
Sharing tools: `share_watchlist`, `update_watchlist_share`, `revoke_watchlist_share`, `list_watchlist_shares`, `list_shared_watchlists`, `list_watchlist_audit`. See `docs/features/watchlist-sharing.md`.  
Export tools: `start_export`, `get_export`, `list_exports`, `cancel_export` (includes paper `portfolios`). See `docs/features/user-data-export.md`.  
Import tools: `preview_import`, `confirm_import`, `get_import`, `list_imports`, `cancel_import` (merge or replace portfolios). See `docs/features/user-data-import.md`.  
API keys: `create_api_key`, `list_api_keys`, `revoke_api_key`. Tool JSON redacts `secret` / token fields (and `swy_…` patterns) so one-time secrets are not fed into the model or chat trace. See `docs/features/api-keys.md`.  
Paper orders: `list_portfolios`, `create_portfolio`, `rename_portfolio`, `delete_portfolio`, `share_portfolio`, `update_portfolio_share`, `revoke_portfolio_share`, `list_portfolio_shares`, `list_shared_portfolios`, `get_portfolio`, `get_paper_trading_costs`, `place_portfolio_order`, `place_portfolio_pending_order`, `place_portfolio_oco_order`, `place_portfolio_bracket_order`, `list_portfolio_orders`, `get_portfolio_order`, `amend_portfolio_order`, `cancel_all_portfolio_orders`, `cancel_portfolio_order`, `list_portfolio_trades`, `get_portfolio_performance`, `deposit_portfolio_cash`, `withdraw_portfolio_cash`, `transfer_portfolio_cash`, `list_portfolio_cash_movements` (optional `portfolioId` / `portfolio_id` when multiple books exist; place tools accept `idempotencyKey` so retries do not double-fill). See `docs/features/paper-trading.md`.  
Recurring buys: `create_recurring_buy`, `update_recurring_buy`, `list_recurring_buys`, `get_recurring_buy`, `pause_recurring_buy`, `resume_recurring_buy`, `delete_recurring_buy`, `list_recurring_buy_runs`. See `docs/features/recurring-buys.md`.  
Allocation baskets: `create_portfolio_basket`, `list_portfolio_baskets`, `get_portfolio_basket`, `update_portfolio_basket`, `delete_portfolio_basket`, `preview_portfolio_rebalance`, `rebalance_portfolio_basket` (manual only). See `docs/features/allocation-baskets.md`.  
Risk limits: `get_portfolio_risk_limits`, `set_portfolio_risk_limits`, `clear_portfolio_risk_limits`. See `docs/features/risk-limits.md`.  
Paper margin: `place_margin_order`, `list_margin_positions`, `close_margin_position`, `set_margin_brackets`, `list_margin_orders`, `cancel_margin_order`, `list_margin_trades` (`place_margin_order` / `close_margin_position` accept `idempotencyKey`). See `docs/features/paper-margin.md`.  
Price-diff: `create_price_diff_watch`, `list_price_diff_watches`, `get_price_diff_watch`, `delete_price_diff_watch`, `list_price_diff_opportunities`, `get_price_diff_opportunity`. See `docs/features/price-diff.md`.

## Limitations / follow-ups

- X agent is not the official X API (StockTwits + HN + optional DDG)
- DuckDuckGo is last-resort and often times out; publisher RSS / EDGAR / KAP are preferred
- Default CLI `client_id` `ai-assistant` is reserved by the backend — set a real id for watchlist/paper tools
- `:8090` is open unless `AI_SERVICE_TOKEN` is set on both Python and Go

## Mobile client

Product mobile **Ask** tab: `docs/features/mobile-ai-chat.md` (OpenAPI `postAiChat`, RTK `aiApi`).

