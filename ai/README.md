# Swyngora AI (`ai/`)

Multi-agent crypto analysis assistant built with **LangGraph + LangChain**.

## Architecture

Research-backed **supervisor / subagents-as-tools** pattern (LangChain multi-agent guidance 2025–2026):

```text
User
  └─► Orchestrator (ReAct — calls a specialist only if the question needs it)
        ├─ market_tape_agent   → ticker, candles, RSI/EMA, supply, holders, FX, lists, delists
        ├─ market_book_agent   → book, liqs, impact, pumps, swing
        ├─ paper_desk_agent    → paper portfolios / OCO / bracket / amend / margin
        ├─ account_agent       → watchlist, alerts, keys, export/import
        ├─ web_agent           → allowlisted RSS + wiki + Gecko + EDGAR/KAP
        ├─ x_agent             → StockTwits / HN (weak social)
        └─ analyst_agent       → optional synthesis
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
   - **Ollama**: `ollama pull qwen2.5` (or set `OLLAMA_MODEL`; avoid llama3.2 — weak tool calling)
   - **Grok**: set `XAI_API_KEY` and `AI_LLM_PROVIDER=grok`
   - After the primary’s 4 tries fail, the agent tries the **other** provider (Grok ↔ Ollama). A leftover `XAI_API_KEY` while on Ollama will call Grok if Ollama is down.

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

# HTTP (used by the Go proxy + web Process panel)
python -m swyngora_ai.serve --host 127.0.0.1 --port 8090
# POST /v1/chat          → JSON {reply, tools, thinking, references}
# POST /v1/chat/stream   → NDJSON status/thinking/tool/final/done

# Flags: --session, --provider ollama|grok; swyngora-ai --help
```

## Environment

Copy [`ai/.env.example`](.env.example) → `ai/.env` (or use repo-root / `backend/.env`).

| Variable | Default | Meaning |
|----------|---------|---------|
| `AI_LLM_PROVIDER` | `ollama` | `ollama` or `grok` |
| `OLLAMA_BASE_URL` | `http://127.0.0.1:11434` | Ollama host |
| `OLLAMA_MODEL` | `qwen2.5` | Local model (needs tool calling) |
| `AI_SERVICE_TOKEN` | empty | Shared secret with the Go proxy; empty = open localhost |
| `AI_MEMORY_PATH` | empty | FinMem SQLite path (`data/ai-memory.db` or `:memory:`) |
| `XAI_API_KEY` | — | **Required for Grok** (https://console.x.ai/). Also enables Grok as the Ollama fallback. |
| `GROK_MODEL` | `grok-4.3` | xAI model id (cheapest general chat) |
| `GROK_REASONING_EFFORT` | `low` | Grok only: `none` \| `low` \| `medium` \| `high` |
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
