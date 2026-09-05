# AGENTS.md — `ai/`

Closest wins for files under `ai/`.

## Scope

Python multi-agent assistant (LangGraph orchestrator + specialists). LLM providers: **Ollama** and **Grok (xAI)** only. `create_agent` retries the primary, then falls back to the other provider via `build_agent_middleware`. The same list includes `ModelCallLimitMiddleware` (`run_limit=6` per invoke, orchestrator and each specialist) and `ToolRetryMiddleware` for transient **read** market-tool HTTP errors (`ToolTransientError`: timeout, connect, 429, 5xx). `NoteToolError` (`on_tool_error`) logs tool-retry lines on the thinking trace (same `status` channel as model retry). The tool column classifies the final v3 stream item per `tool_call_id` (`output` / `error`); an exhausted raise is `tool_error`, not a ✓. A specialist return is prefixed `ERROR` when every nested tool failed, so `market_tape_agent` is not ticked. Mutating paper/account calls stay fail-soft strings and are not retried.

## Rules

1. Do not add OpenAI/Anthropic/Gemini SDKs as defaults.
2. Market numbers come from Swyngora tools (HTTP/MCP), never hallucinated. `Orchestrator.chat` collects tool JSON from the turn (including nested specialist HTTP calls) into `MarketFacts` and flags unverified figures.
3. Web research uses the allowlist in `sources/allowlist.py` (no paid search). New publishers need a reliability tier.
4. New backend capabilities that AI should use must ship with MCP tools in the same change (`backend/internal/transport/mcp`).
5. Tests required for new tools, routing helpers, and prompts that change behaviour.
6. User-facing answers must avoid absolute financial guarantees; keep disclaimer.
7. Prefer free/public search; no paid data-vendor plan switches.

## Commands

```bash
cd ai
uv sync                          # .venv + editable install + lock + dev tools
source .venv/bin/activate
pytest -q
ruff check . && ruff format --check .
ty check
swyngora-ai "your question"
uv run langgraph dev           # local Agent Chat UI: :2024 + graph id swyngora
```

Package manager is **uv** (not pip). `uv sync` reads `pyproject.toml` / `uv.lock` and creates `ai/.venv`. Activate that venv, then run `pytest`, `ruff`, `ty`, and `swyngora-ai` as usual. Recreate a stale pip venv with `rm -rf .venv && uv sync`.

Ruff and ty config live in `pyproject.toml`. Run from `ai/`. Use `ruff check --fix . && ruff format .` to apply auto-fixes.

Optional LangSmith: copy `LANGSMITH_TRACING` / `LANGSMITH_PROJECT` / `LANGSMITH_API_KEY` / `LANGSMITH_ENDPOINT` from `.env.example` into `ai/.env` and set `LANGSMITH_API_KEY`. Endpoint defaults to APAC; other regions: [LangSmith tracing docs](https://docs.langchain.com/langsmith/trace-with-langchain). The CLI **and** `swyngora-ai-serve` load `ai/.env` into `os.environ` (mobile Ask / web `/ai` go through serve). Restart the process after editing `.env`. No `--trace` flag. Empty key = no traces.
