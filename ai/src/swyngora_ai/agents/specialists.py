"""Specialist ReAct agents (market, web, X, analyst)."""

from __future__ import annotations

from collections.abc import Sequence
from typing import Any

from langchain.agents import create_agent
from langchain_core.language_models.chat_models import BaseChatModel
from langchain_core.messages import HumanMessage, SystemMessage, ToolMessage
from langchain_core.tools import BaseTool, StructuredTool
from pydantic import BaseModel, Field

from swyngora_ai.agents.prompts import (
    ANALYST_SYSTEM,
    MARKET_SYSTEM,
    WEB_SYSTEM,
    X_SYSTEM,
)
from swyngora_ai.config import Settings, get_settings
from swyngora_ai.progress import emit
from swyngora_ai.references import extract_references
from swyngora_ai.tools.market_http import build_market_tools
from swyngora_ai.tools.web_search import build_web_tools
from swyngora_ai.tools.x_search import build_x_tools


def _last_text(result: dict[str, Any]) -> str:
    msgs = result.get("messages") or []
    if not msgs:
        return ""
    content = getattr(msgs[-1], "content", msgs[-1])
    if isinstance(content, list):
        parts = []
        for block in content:
            if isinstance(block, dict) and block.get("type") == "text":
                parts.append(block.get("text", ""))
            else:
                parts.append(str(block))
        return "\n".join(parts)
    return str(content)


def _wrap_tools_with_progress(tools: Sequence[BaseTool], specialist: str) -> list[BaseTool]:
    """Emit live tool events when leaf tools run."""
    wrapped: list[BaseTool] = []
    for t in tools:
        original = t

        def make_fn(tool: BaseTool):
            def _fn(**kwargs: Any) -> str:
                arg_bits = []
                for k, v in list(kwargs.items())[:5]:
                    vs = str(v)
                    if len(vs) > 60:
                        vs = vs[:57] + "…"
                    arg_bits.append(f"{k}={vs}")
                emit("tool", f"{specialist} → {tool.name}({', '.join(arg_bits)})")
                try:
                    out = tool.invoke(kwargs)
                except Exception as e:  # noqa: BLE001
                    emit("tool_error", f"{tool.name} failed: {e}")
                    raise
                preview = str(out).replace("\n", " ").strip()
                if len(preview) > 100:
                    preview = preview[:97] + "…"
                emit("tool_result", f"{tool.name} ✓ {preview}")
                return out if isinstance(out, str) else str(out)

            return _fn

        # Rebuild StructuredTool with same name/description/schema
        try:
            schema = original.args_schema
        except Exception:
            schema = None
        wrapped.append(
            StructuredTool.from_function(
                make_fn(original),
                name=original.name,
                description=original.description or original.name,
                args_schema=schema,
            )
        )
    return wrapped


def _run_react(
    model: BaseChatModel,
    tools: Sequence[BaseTool],
    system: str,
    task: str,
    specialist: str,
    recursion_limit: int = 12,
) -> str:
    from swyngora_ai.graph.orchestrator import extract_trace

    emit("status", f"Running {specialist}…")
    emit("thinking", f"{specialist}: {task[:200]}")
    tools = _wrap_tools_with_progress(tools, specialist)
    agent = create_agent(model, list(tools), system_prompt=system)
    out = agent.invoke(
        {"messages": [HumanMessage(content=task)]},
        config={"recursion_limit": recursion_limit},
    )
    msgs = list(out.get("messages") or [])
    reply, nested_tools, _thinking = extract_trace(msgs)
    if not reply:
        reply = _last_text(out)
    # Re-emit nested tools for live UIs that missed leaf-tool wraps.
    for t in nested_tools[:30]:
        emit("tool", f"{specialist}: {t}")
    emit("status", f"{specialist} finished")
    # Keep URLs on the specialist return so the orchestrator can list Sources.
    blobs = [
        _content_text_local(getattr(m, "content", "")) for m in msgs if isinstance(m, ToolMessage)
    ]
    refs = extract_references(*blobs, reply)
    if refs:
        lines = "\n".join(f"- [{r.title}]({r.url})" for r in refs[:12])
        reply = f"{reply.rstrip()}\n\nSOURCES:\n{lines}"
    return reply


def _content_text_local(content: Any) -> str:
    if content is None:
        return ""
    if isinstance(content, str):
        return content
    return str(content)


class TaskInput(BaseModel):
    task: str = Field(description="Clear task for the specialist agent")


def build_specialist_tools(
    model: BaseChatModel,
    settings: Settings | None = None,
) -> list[StructuredTool]:
    """Expose specialists as tools the orchestrator can call (subagents-as-tools)."""
    cfg = settings or get_settings()
    client_id = cfg.default_client_id
    market_tools = build_market_tools(cfg)
    web_tools = build_web_tools()
    x_tools = build_x_tools()

    def market_agent(task: str) -> str:
        emit("tool", f"market_agent(task={task[:80]})")
        return _run_react(
            model,
            market_tools,
            MARKET_SYSTEM.format(client_id=client_id),
            task,
            specialist="market_agent",
            recursion_limit=cfg.max_agent_iterations + 4,
        )

    def web_agent(task: str) -> str:
        emit("tool", f"web_agent(task={task[:80]})")
        return _run_react(model, web_tools, WEB_SYSTEM, task, specialist="web_agent")

    def x_agent(task: str) -> str:
        emit("tool", f"x_agent(task={task[:80]})")
        return _run_react(model, x_tools, X_SYSTEM, task, specialist="x_agent")

    def analyst_agent(task: str) -> str:
        emit("tool", "analyst_agent(synthesize)")
        emit("thinking", "Synthesizing findings…")
        msgs = [
            SystemMessage(content=ANALYST_SYSTEM),
            HumanMessage(content=task),
        ]
        resp = model.invoke(msgs)
        content = resp.content
        if isinstance(content, list):
            return " ".join(str(c) for c in content)
        return str(content)

    return [
        StructuredTool.from_function(
            market_agent,
            name="market_agent",
            description=(
                "Swyngora market specialist with MCP/HTTP tools: prices, candles, "
                "supply, spot rankings, RSI/EMA, watchlist. Use for all market numbers."
            ),
            args_schema=TaskInput,
        ),
        StructuredTool.from_function(
            web_agent,
            name="web_agent",
            description="Web/news research specialist. Background, news, project info.",
            args_schema=TaskInput,
        ),
        StructuredTool.from_function(
            x_agent,
            name="x_agent",
            description="X/Twitter weak-signal specialist. Social chatter only.",
            args_schema=TaskInput,
        ),
        StructuredTool.from_function(
            analyst_agent,
            name="analyst_agent",
            description=(
                "Lead analyst: synthesize prior specialist outputs into a final structured answer. "
                "Pass all relevant findings in the task text."
            ),
            args_schema=TaskInput,
        ),
    ]
