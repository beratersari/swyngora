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

## Limitations / follow-ups

- No durable conversation store or multi-user auth yet
- X agent is not official X API
- Telegram AI mode (`/ask`) not wired yet
- Streaming UI not shipped
