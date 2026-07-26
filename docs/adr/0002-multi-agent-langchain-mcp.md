# ADR 0002: Multi-agent LangGraph + Go MCP

## Status

Accepted — 2026-07-26

## Context

Swyngora needs an AI assistant that can combine market data, web context, and weak social signals. Research (LangChain multi-agent architecture notes, 2025–2026) favors:

- **Supervisor / subagents-as-tools** for multi-domain tasks with centralized control
- **LangGraph** for production orchestration over legacy `AgentExecutor`
- **MCP** as a stable tool boundary for host-agnostic tool exposure

Constraints from AGENTS.md: Ollama + Grok only; free/public data preference; backend N-layered.

## Decision

1. Implement a Python package `ai/` with a LangGraph **orchestrator** that calls specialist agents as tools: market, web, X, analyst.
2. Implement market tools as a **Go MCP server** (`backend/cmd/mcp`) that adapts the public HTTP API (no business-logic duplication).
3. Python also binds the same market tools via **HTTP** so tests and local runs work without spawning MCP.
4. Mandate: new agent-useful features ship with MCP tools in the same MR.

## Consequences

- Clear separation: Go owns data plane; Python owns reasoning plane.
- Extra process for stdio MCP (optional for HTTP-tool path).
- Specialist fan-out increases latency/token cost vs single ReAct agent — acceptable for quality.
