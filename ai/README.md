# Swyngora AI (`ai/`)

Multi-agent crypto analysis assistant built with **LangGraph + LangChain**.

## Architecture

Research-backed **supervisor / subagents-as-tools** pattern (LangChain multi-agent guidance 2025–2026):

```text
User
  └─► Orchestrator (ReAct, plans & routes)
        ├─ market_agent  → Swyngora MCP/HTTP tools (prices, candles, supply, RSI/EMA, watchlist)
        ├─ web_agent     → free DuckDuckGo web + news search
        ├─ x_agent       → free X/Twitter-indexed search (weak social signal)
        └─ analyst_agent → synthesis + risk framing (no tools)
```

| Principle | How we apply it |
|-----------|-----------------|
| Centralized control | Orchestrator owns routing; specialists do not talk to each other |
| Tool-grounded numbers | Market agent must call tools; no invented prices |
| Fail soft | Tool errors returned as text; orchestrator degrades |
| Provider constraint | **Ollama** (local default) or **Grok/xAI** only |
| No paid data tiers | DuckDuckGo free search; market data via our backend |

## Prerequisites

1. Backend API running: `cd backend && go run ./cmd/server` (`:8080`)
2. Optional MCP stdio server: `cd backend && go run ./cmd/mcp`
3. LLM:
   - **Ollama**: `ollama pull llama3.2` (or set `OLLAMA_MODEL`)
   - **Grok**: set `XAI_API_KEY` and `AI_LLM_PROVIDER=grok`

## Setup

Requires [uv](https://docs.astral.sh/uv/) ≥ 0.12 (one-time: `curl -LsSf https://astral.sh/uv/install.sh | sh`).

```bash
cd ai
uv sync                      # creates .venv, editable install, runtime + dev from uv.lock
source .venv/bin/activate
```

Python 3.11+ (pinned in `.python-version`). Recreate an old pip venv with `rm -rf .venv && uv sync`.

## Run

```bash
# REPL
export AI_LLM_PROVIDER=ollama
export SWYNGORA_API_URL=http://localhost:8080
swyngora-ai

# One-shot
swyngora-ai "What is BTC RSI on binance 1h?"
```

## Environment

Copy [`ai/.env.example`](.env.example) → `ai/.env` (or use repo-root / `backend/.env`).

| Variable | Default | Meaning |
|----------|---------|---------|
| `AI_LLM_PROVIDER` | `ollama` | `ollama` or `grok` |
| `OLLAMA_BASE_URL` | `http://127.0.0.1:11434` | Ollama host |
| `OLLAMA_MODEL` | `llama3.2` | Local model name |
| `XAI_API_KEY` | — | **Required for Grok** (https://console.x.ai/) |
| `GROK_MODEL` | `grok-3-mini` | xAI model id |
| `SWYNGORA_API_URL` | `http://localhost:8080` | Backend API |
| `AI_DEFAULT_CLIENT_ID` | `ai-assistant` | Watchlist client id |
| `AI_MAX_ITERATIONS` | `8` | Specialist ReAct depth hint |
| `AI_TEMPERATURE` | `0.2` | Sampling temperature |

## Tests

```bash
cd ai
source .venv/bin/activate   # after uv sync
pytest -q
ty check
```

Unit tests use scripted/fake models and mocked HTTP — no live LLM or network required.

## Lint and format

[Ruff](https://docs.astral.sh/ruff/) is the linter and formatter (config in `pyproject.toml`). After `uv sync` and activating `.venv`:

```bash
cd ai
source .venv/bin/activate
ruff check .              # lint (see pyproject.toml [tool.ruff.lint].select)
ruff format --check .     # formatter dry-run
ruff check --fix . && ruff format .   # apply auto-fixes + format
ty check                  # type checker (Astral ty; default error rules)
```

Target Python 3.11+, line length 100. Prompt/URL strings may exceed 100 (`E501` ignored).

## MCP

MCP is **integrated into the backend process** (`go run ./cmd/server`):

- REST: `http://localhost:8080/api/v1/...`
- MCP streamable HTTP: `http://localhost:8080/mcp`

Python market tools call the REST API (same contracts as MCP tool names). Optional stdio binary `backend/cmd/mcp` is only for hosts that cannot use HTTP MCP.

**Rule:** when product features are added, expose them via MCP tools so AI can use them (see root `AGENTS.md`).

## Layout

```text
ai/
├── pyproject.toml    # package + uv_build + ruff + ty + pytest
├── uv.lock           # committed lockfile (uv sync --frozen)
├── .python-version   # 3.11
├── src/swyngora_ai/
│   ├── agents/       # prompts + specialist builders
│   ├── graph/        # orchestrator
│   ├── llm/          # Ollama / Grok factory
│   ├── tools/        # market HTTP, web, X
│   └── cli.py
└── tests/
```
