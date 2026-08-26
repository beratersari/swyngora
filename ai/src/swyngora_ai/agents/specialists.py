"""Specialist ReAct agents (tape / book / paper / account / web / X / analyst)."""

from __future__ import annotations

from collections.abc import Callable, Sequence
from typing import Any

from langchain.agents import create_agent
from langchain_core.language_models.chat_models import BaseChatModel
from langchain_core.messages import HumanMessage, SystemMessage, ToolMessage
from langchain_core.tools import BaseTool, StructuredTool
from pydantic import BaseModel, Field

from swyngora_ai.agents.prompts import (
    ACCOUNT_SYSTEM,
    ANALYST_SYSTEM,
    BOOK_SYSTEM,
    PAPER_SYSTEM,
    TAPE_SYSTEM,
    WEB_SYSTEM,
    X_SYSTEM,
)
from swyngora_ai.config import Settings, get_settings
from swyngora_ai.constants import (
    SPECIALIST_ACCOUNT,
    SPECIALIST_ANALYST,
    SPECIALIST_BOOK,
    SPECIALIST_PAPER,
    SPECIALIST_TAPE,
    SPECIALIST_WEB,
    SPECIALIST_X,
)
from swyngora_ai.llm.retry import MODEL_RETRY
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


def _content_text_local(content: Any) -> str:
    if content is None:
        return ""
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        parts: list[str] = []
        for block in content:
            if isinstance(block, str):
                parts.append(block)
            elif isinstance(block, dict) and block.get("type") not in {
                "reasoning",
                "tool_call",
                "tool_use",
            }:
                text = block.get("text")
                if text:
                    parts.append(str(text))
        return "\n".join(p for p in parts if p)
    return str(content)


def _run_react(
    model: BaseChatModel,
    tools: Sequence[BaseTool],
    system: str,
    task: str,
    specialist: str,
    recursion_limit: int = 12,
    compiled: Any | None = None,
    middleware: Sequence[Any] | None = None,
) -> str:
    from swyngora_ai.graph.orchestrator import extract_trace, run_agent_with_progress

    emit("status", f"Running {specialist}…")
    emit("thinking", f"{specialist}: {task[:200]}")
    agent = compiled
    if agent is None:
        agent = create_agent(
            model,
            list(tools),
            system_prompt=system,
            middleware=list(middleware) if middleware is not None else [MODEL_RETRY],
        )
    msgs = run_agent_with_progress(
        agent,
        [HumanMessage(content=task)],
        {"recursion_limit": recursion_limit},
        specialist=specialist,
    )
    reply, _nested_tools, _thinking = extract_trace(msgs)
    if not reply:
        reply = _last_text({"messages": msgs})
    emit("status", f"{specialist} finished")
    blobs = [
        _content_text_local(getattr(m, "content", "")) for m in msgs if isinstance(m, ToolMessage)
    ]
    refs = extract_references(*blobs, reply)
    if refs:
        lines = "\n".join(f"- [{r.title}]({r.url})" for r in refs[:12])
        reply = f"{reply.rstrip()}\n\nSOURCES:\n{lines}"
    return reply


class TaskInput(BaseModel):
    task: str = Field(description="Clear task for the specialist agent")


class SpecialistRunner:
    """Compile each specialist once; run ReAct with a small tool pack."""

    def __init__(
        self,
        model: BaseChatModel,
        settings: Settings | None = None,
        middleware: Sequence[Any] | None = None,
    ) -> None:
        self.model = model
        self.settings = settings or get_settings()
        self.client_id = self.settings.default_client_id
        self._middleware = list(middleware) if middleware is not None else [MODEL_RETRY]
        self._compiled: dict[str, Any] = {}
        self._tools: dict[str, list[BaseTool]] = {}
        self._systems: dict[str, str] = {}
        self._prepare()

    def _prepare(self) -> None:
        cid = self.client_id
        packs: list[tuple[str, str, Sequence[BaseTool]]] = [
            (
                SPECIALIST_TAPE,
                TAPE_SYSTEM.format(client_id=cid),
                build_market_tools(self.settings, pack="tape"),
            ),
            (
                SPECIALIST_BOOK,
                BOOK_SYSTEM.format(client_id=cid),
                build_market_tools(self.settings, pack="book"),
            ),
            (
                SPECIALIST_PAPER,
                PAPER_SYSTEM.format(client_id=cid),
                build_market_tools(self.settings, pack="paper"),
            ),
            (
                SPECIALIST_ACCOUNT,
                ACCOUNT_SYSTEM.format(client_id=cid),
                build_market_tools(self.settings, pack="account"),
            ),
            (SPECIALIST_WEB, WEB_SYSTEM, build_web_tools()),
            (SPECIALIST_X, X_SYSTEM, build_x_tools()),
        ]
        for name, system, tools in packs:
            leaf = list(tools)
            self._tools[name] = leaf
            self._systems[name] = system
            self._compiled[name] = create_agent(
                self.model,
                leaf,
                system_prompt=system,
                middleware=self._middleware,
            )

    def run(self, name: str, task: str) -> str:
        limit = self.settings.max_agent_iterations + 4
        if name == SPECIALIST_ANALYST:
            return self.analyze(task)
        compiled = self._compiled.get(name)
        tools = self._tools.get(name) or []
        system = self._systems.get(name) or ""
        return _run_react(
            self.model,
            tools,
            system,
            task,
            specialist=name,
            recursion_limit=limit,
            compiled=compiled,
            middleware=self._middleware,
        )

    def analyze(self, task: str) -> str:
        emit("thinking", "Synthesizing findings…")
        msgs = [
            SystemMessage(content=ANALYST_SYSTEM),
            HumanMessage(content=task),
        ]
        resp = self.model.invoke(msgs)
        return _content_text_local(resp.content)

    def as_orchestrator_tools(self) -> list[StructuredTool]:
        """Legacy subagents-as-tools surface (tests + fallback)."""

        def make(name: str) -> Callable[[str], str]:
            def _fn(task: str) -> str:
                return self.run(name, task)

            return _fn

        specs = [
            (
                SPECIALIST_TAPE,
                "Call ONLY if the user asked for live price, RSI/EMA, candles, supply, FX, or delists. "
                "Venues binance|coinbase|bybit|nasdaq|bist.",
            ),
            (
                SPECIALIST_BOOK,
                "Call ONLY if the user asked for order book, liquidations, liquidity, impact, pumps, swing, "
                "funding, open interest, CVD, whales, levels, snapshot, or related flow analytics.",
            ),
            (
                SPECIALIST_PAPER,
                "Call ONLY if the user asked about a paper portfolio, order, or margin action.",
            ),
            (
                SPECIALIST_ACCOUNT,
                "Call ONLY if the user asked about watchlist, alerts, API keys, export/import, or scanner.",
            ),
            (
                SPECIALIST_WEB,
                "Call ONLY if the user asked for news, filings, or project/background research.",
            ),
            (
                SPECIALIST_X,
                "Call ONLY if the user asked for Twitter/X/StockTwits/social sentiment.",
            ),
            (
                SPECIALIST_ANALYST,
                "Optional synthesis after several specialists. Do not call for a simple question.",
            ),
        ]
        return [
            StructuredTool.from_function(
                make(name),
                name=name,
                description=desc,
                args_schema=TaskInput,
            )
            for name, desc in specs
        ]


def build_specialist_tools(
    model: BaseChatModel,
    settings: Settings | None = None,
) -> list[StructuredTool]:
    """Expose specialists as tools the orchestrator can call (subagents-as-tools)."""
    return SpecialistRunner(model, settings).as_orchestrator_tools()
