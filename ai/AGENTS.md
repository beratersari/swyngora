# AGENTS.md — `ai/`

Closest wins for files under `ai/`.

## Scope

Python multi-agent assistant (LangGraph orchestrator + specialists). LLM providers: **Ollama** and **Grok (xAI)** only.

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
```

Package manager is **uv** (not pip). `uv sync` reads `pyproject.toml` / `uv.lock` and creates `ai/.venv`. Activate that venv, then run `pytest`, `ruff`, `ty`, and `swyngora-ai` as usual. Recreate a stale pip venv with `rm -rf .venv && uv sync`.

Ruff and ty config live in `pyproject.toml`. Run from `ai/`. Use `ruff check --fix . && ruff format .` to apply auto-fixes.
