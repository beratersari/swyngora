# Feature: Multi-agent AI assistant

## Problem / goal

Users need explanations and multi-step market analysis, not only raw JSON from the API. Swyngora provides a **hierarchical multi-agent** assistant that plans, gathers tool-backed data, and synthesizes answers — without inventing prices.

## Architecture

Based on LangChain multi-agent research (supervisor / **subagents-as-tools**): centralized control, specialists for distinct domains, bounded ReAct loops.

```text
User → Orchestrator (LangGraph create_react_agent)
         ├─ market_agent  → HTTP tools ≡ Go MCP tools
         ├─ web_agent     → DuckDuckGo web + news (free)
         ├─ x_agent       → X/Twitter-indexed search (weak signal)
         └─ analyst_agent → synthesis (no tools)
```

| Component | Path |
|-----------|------|
| Orchestrator | `ai/src/swyngora_ai/graph/orchestrator.py` |
| Specialists | `ai/src/swyngora_ai/agents/` |
| Market tools | `ai/src/swyngora_ai/tools/market_http.py` |
| Go MCP server | `backend/cmd/mcp`, `backend/internal/transport/mcp` |
| LLM factory | `ai/src/swyngora_ai/llm/factory.py` (Ollama \| Grok) |

## Behavior

- Market numbers must come from tools (ticker, candles, supply, indicators, spot).
- Paper sell realized PnL uses tax lots (`list_portfolio_lots`, `lotMethod` fifo|lifo) after the sell fee; buy lot cost includes the buy fee. Per-exchange rates: `get_paper_trading_costs`.
- Live UI updates use the backend WebSocket (`realtime_stream_info` / `GET /api/v1/realtime`); tools remain request/response.
- Social/X results are labeled weak and incomplete.
- Coin/project questions dispatch **web_agent** (`web_research` + `web_news`) and optionally **x_agent**; public **URLs** return as `references` on the chat payload and render as source cards in the web UI.
- Answers include “not financial advice” framing for market questions.
- Session memory is in-process (not durable across restarts).
- HTTP/Telegram pass `clientId`; Python tools bind that id (and send `SWYNGORA_API_TOKEN` / `X-Client-Id`) so the model cannot switch tenants. Reserved ids (`ai-assistant`, `http-default`, `anonymous`) are rejected by the backend.
- Web `/ai` streams `POST /api/v1/ai/chat/stream` and shows a **Process** timeline (status / think / tools / results) as each step happens. The list stays open while working, then collapses to a one-line summary so the answer stays readable. Non-stream `POST /api/v1/ai/chat` remains as fallback.
- Telegram `/ask` uses the same stream for live progress edits.

## How to run

```bash
# One backend process: REST + MCP (/mcp)
cd backend && go run ./cmd/server

# AI CLI (separate process — needs LLM)
cd ai && uv sync && source .venv/bin/activate
export AI_LLM_PROVIDER=ollama   # or grok + XAI_API_KEY
export SWYNGORA_API_URL=http://localhost:8080
# export SWYNGORA_API_TOKEN=...   # same as backend API_AUTH_TOKEN when set
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

Market specialist tools mirror Go MCP (ticker, **spot order book** / **order-book analysis** / **market-wide book** / **market impact**, **order-book alerts**, candles, indicators, pumps, paper portfolio, scanner, **watchlist sharing**, **API keys**).  
Impact: `estimate_market_impact` walks live depth for a market buy/sell (`quantity` or `notional`). Liquidity: `get_market_liquidity` (0–100 + weaker side, per venue + market-wide). Liquidations: `get_liquidations` (5m/1h/4h/24h long vs short, Binance USD-M + Bybit linear). Open interest: `get_open_interest` (current + 5m/1h/4h/24h change, Binance USD-M + Bybit linear). Walls include `behavior` (`short` / `persistent` / `suspicious`). See `docs/features/order-book.md`, `docs/features/liquidations.md`, and `docs/features/open-interest.md`.  
Sharing tools: `share_watchlist`, `update_watchlist_share`, `revoke_watchlist_share`, `list_watchlist_shares`, `list_shared_watchlists`, `list_watchlist_audit`. See `docs/features/watchlist-sharing.md`.  
Export tools: `start_export`, `get_export`, `list_exports`, `cancel_export` (includes paper `portfolios`). See `docs/features/user-data-export.md`.  
Import tools: `preview_import`, `confirm_import`, `get_import`, `list_imports`, `cancel_import` (merge or replace portfolios). See `docs/features/user-data-import.md`.  
API keys: `create_api_key`, `list_api_keys`, `revoke_api_key`. See `docs/features/api-keys.md`.  
Paper orders: `list_portfolios`, `create_portfolio`, `rename_portfolio`, `delete_portfolio`, `share_portfolio`, `update_portfolio_share`, `revoke_portfolio_share`, `list_portfolio_shares`, `list_shared_portfolios`, `get_portfolio`, `get_paper_trading_costs`, `place_portfolio_order`, `place_portfolio_pending_order`, `place_portfolio_oco_order`, `place_portfolio_bracket_order`, `list_portfolio_orders`, `get_portfolio_order`, `amend_portfolio_order`, `cancel_all_portfolio_orders`, `cancel_portfolio_order`, `list_portfolio_trades`, `get_portfolio_performance`, `deposit_portfolio_cash`, `withdraw_portfolio_cash`, `transfer_portfolio_cash`, `list_portfolio_cash_movements` (optional `portfolioId` / `portfolio_id` when multiple books exist; place tools accept `idempotencyKey` so retries do not double-fill). See `docs/features/paper-trading.md`.  
Recurring buys: `create_recurring_buy`, `update_recurring_buy`, `list_recurring_buys`, `get_recurring_buy`, `pause_recurring_buy`, `resume_recurring_buy`, `delete_recurring_buy`, `list_recurring_buy_runs`. See `docs/features/recurring-buys.md`.  
Allocation baskets: `create_portfolio_basket`, `list_portfolio_baskets`, `get_portfolio_basket`, `update_portfolio_basket`, `delete_portfolio_basket`, `preview_portfolio_rebalance`, `rebalance_portfolio_basket` (manual only). See `docs/features/allocation-baskets.md`.  
Risk limits: `get_portfolio_risk_limits`, `set_portfolio_risk_limits`, `clear_portfolio_risk_limits`. See `docs/features/risk-limits.md`.  
Paper margin: `place_margin_order`, `list_margin_positions`, `close_margin_position`, `set_margin_brackets`, `list_margin_orders`, `cancel_margin_order`, `list_margin_trades` (`place_margin_order` / `close_margin_position` accept `idempotencyKey`). See `docs/features/paper-margin.md`.  
Price-diff: `create_price_diff_watch`, `list_price_diff_watches`, `get_price_diff_watch`, `delete_price_diff_watch`, `list_price_diff_opportunities`, `get_price_diff_opportunity`. See `docs/features/price-diff.md`.

## Limitations / follow-ups

- No durable conversation store or multi-user auth yet
- X agent is not official X API
- Web research prefers Wikipedia, Google News RSS, CoinGecko, and Hacker News (free). DuckDuckGo is optional and often times out.
- X is StockTwits/HN proxies, not the official API

## Mobile client

Product mobile **Ask** tab: `docs/features/mobile-ai-chat.md` (OpenAPI `postAiChat`, RTK `aiApi`).

