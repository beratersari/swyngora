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
- Social/X results are labeled weak and incomplete.
- Answers include “not financial advice” framing for market questions.
- Session memory is in-process (not durable across restarts).

## How to run

```bash
# One backend process: REST + MCP (/mcp)
cd backend && go run ./cmd/server

# AI CLI (separate process — needs LLM)
cd ai && pip install -e ".[dev]"
export AI_LLM_PROVIDER=ollama   # or grok + XAI_API_KEY
export SWYNGORA_API_URL=http://localhost:8080
swyngora-ai "BTC RSI on binance 1h and recent news"
```

## Tests

```bash
cd backend && go test ./internal/transport/mcp/...
cd ai && pytest -q
```

## Market / watchlist tools

Market specialist tools mirror Go MCP (ticker, candles, indicators, pumps, paper portfolio, scanner, **watchlist sharing**, **API keys**).  
Sharing tools: `share_watchlist`, `update_watchlist_share`, `revoke_watchlist_share`, `list_watchlist_shares`, `list_shared_watchlists`, `list_watchlist_audit`. See `docs/features/watchlist-sharing.md`.  
Export tools: `start_export`, `get_export`, `list_exports`, `cancel_export`. See `docs/features/user-data-export.md`.  
Import tools: `preview_import`, `confirm_import`, `get_import`, `list_imports`, `cancel_import`. See `docs/features/user-data-import.md`.  
API keys: `create_api_key`, `list_api_keys`, `revoke_api_key`. See `docs/features/api-keys.md`.  
Paper orders: `list_portfolios`, `create_portfolio`, `rename_portfolio`, `delete_portfolio`, `share_portfolio`, `update_portfolio_share`, `revoke_portfolio_share`, `list_portfolio_shares`, `list_shared_portfolios`, `get_portfolio`, `place_portfolio_order`, `place_portfolio_pending_order`, `place_portfolio_oco_order`, `place_portfolio_bracket_order`, `list_portfolio_orders`, `get_portfolio_order`, `amend_portfolio_order`, `cancel_all_portfolio_orders`, `cancel_portfolio_order`, `list_portfolio_trades`, `get_portfolio_performance`, `deposit_portfolio_cash`, `withdraw_portfolio_cash`, `list_portfolio_cash_movements` (optional `portfolioId` / `portfolio_id` when multiple books exist). See `docs/features/paper-trading.md`.  
Recurring buys: `create_recurring_buy`, `update_recurring_buy`, `list_recurring_buys`, `get_recurring_buy`, `pause_recurring_buy`, `resume_recurring_buy`, `delete_recurring_buy`, `list_recurring_buy_runs`. See `docs/features/recurring-buys.md`.  
Allocation baskets: `create_portfolio_basket`, `list_portfolio_baskets`, `get_portfolio_basket`, `update_portfolio_basket`, `delete_portfolio_basket`, `preview_portfolio_rebalance`, `rebalance_portfolio_basket` (manual only). See `docs/features/allocation-baskets.md`.  
Risk limits: `get_portfolio_risk_limits`, `set_portfolio_risk_limits`, `clear_portfolio_risk_limits`. See `docs/features/risk-limits.md`.  
Paper margin: `place_margin_order`, `list_margin_positions`, `close_margin_position`, `set_margin_brackets`, `list_margin_orders`, `cancel_margin_order`, `list_margin_trades`. See `docs/features/paper-margin.md`.  
Price-diff: `create_price_diff_watch`, `list_price_diff_watches`, `get_price_diff_watch`, `delete_price_diff_watch`, `list_price_diff_opportunities`, `get_price_diff_opportunity`. See `docs/features/price-diff.md`.

## Limitations / follow-ups

- No durable conversation store or multi-user auth yet
- X agent is not official X API
- Telegram AI mode (`/ask`) not wired yet
- Streaming UI not shipped

## Mobile client

Product mobile **Ask** tab: `docs/features/mobile-ai-chat.md` (OpenAPI `postAiChat`, RTK `aiApi`).

