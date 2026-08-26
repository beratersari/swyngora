"""Hierarchical multi-agent orchestrator (LangGraph supervisor / subagents-as-tools)."""

from __future__ import annotations

import warnings
from collections.abc import Sequence
from dataclasses import dataclass, field
from typing import Any

from langchain.agents import create_agent
from langchain_core.messages import AIMessage, BaseMessage, HumanMessage, SystemMessage, ToolMessage
from langchain_core.runnables import RunnableConfig

from swyngora_ai.agents.prompts import ORCHESTRATOR_SYSTEM
from swyngora_ai.agents.specialists import SpecialistRunner
from swyngora_ai.config import Settings, get_settings
from swyngora_ai.graph.facts import extract_market_facts
from swyngora_ai.graph.route import classify_route
from swyngora_ai.graph.tape_fetch import prefetch_tape
from swyngora_ai.grounding import apply_grounding, grounded_references
from swyngora_ai.language import disclaimer_for, empty_reply_for, has_advice_disclaimer
from swyngora_ai.llm.factory import build_chat_model
from swyngora_ai.llm.retry import MODEL_RETRY, build_agent_middleware
from swyngora_ai.memory.finmem import FinMem, SessionMemory, as_human_ai, memory_key
from swyngora_ai.progress import (
    emit,
    emit_tool_outcome,
    is_tool_error_text,
    reset_progress,
    set_progress,
)
from swyngora_ai.tools.market_http import (
    begin_tool_json_turn,
    bind_client_id,
    bind_tool_scope,
    collected_tool_json,
    record_tool_json,
    reset_bound_client_id,
    reset_tool_json_turn,
    reset_tool_scope,
)


@dataclass
class ChatResult:
    """Final answer plus transparent agent trace for UIs (Telegram, etc.)."""

    reply: str
    tools: list[str] = field(default_factory=list)
    thinking: list[str] = field(default_factory=list)
    session_id: str = "default"
    references: list[dict[str, str]] = field(default_factory=list)


# SessionMemory + memory_key live in swyngora_ai.memory.finmem (re-exported here).


def _should_refresh_tape(memory: Any, client_id: str, message: str) -> bool:
    """True when the user asked for market numbers and FinMem tape is older than TTL."""
    route = classify_route(message)
    if not (route.tape or route.book):
        return False
    check = getattr(memory, "tape_is_stale", None)
    if not callable(check):
        return False
    return bool(check(client_id))


def _content_text(content: Any) -> str:
    if content is None:
        return ""
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        parts: list[str] = []
        for block in content:
            if isinstance(block, str):
                parts.append(block)
            elif isinstance(block, dict):
                btype = block.get("type")
                if btype in {"reasoning", "tool_call", "tool_use"}:
                    continue
                text = block.get("text")
                if btype == "text" or text:
                    parts.append(str(text or ""))
        return "\n".join(p for p in parts if p)
    return str(content)


def extract_trace(messages: Sequence[BaseMessage]) -> tuple[str, list[str], list[str]]:
    """Return (final_reply, tools_used, thinking_steps) from agent messages."""
    tools: list[str] = []
    thinking: list[str] = []
    final = ""

    for msg in messages:
        if isinstance(msg, AIMessage):
            text = _content_text(msg.content).strip()
            tcalls = getattr(msg, "tool_calls", None) or []
            if tcalls:
                if text:
                    thinking.append(text[:500])
                for tc in tcalls:
                    name = tc.get("name") if isinstance(tc, dict) else getattr(tc, "name", "?")
                    args = tc.get("args") if isinstance(tc, dict) else getattr(tc, "args", {})
                    # Compact args for display
                    arg_s = ""
                    if isinstance(args, dict):
                        bits = []
                        for k, v in list(args.items())[:6]:
                            vs = str(v)
                            if len(vs) > 80:
                                vs = vs[:77] + "…"
                            bits.append(f"{k}={vs}")
                        arg_s = ", ".join(bits)
                    tools.append(f"{name}({arg_s})" if arg_s else str(name))
            elif text:
                final = text
                thinking.append(text[:500])
        elif isinstance(msg, ToolMessage):
            name = getattr(msg, "name", None) or "tool"
            preview = _content_text(msg.content).replace("\n", " ").strip()
            if len(preview) > 120:
                preview = preview[:117] + "…"
            tools.append(f"↳ {name} → {preview}" if preview else f"↳ {name}")
        elif isinstance(msg, HumanMessage):
            continue

    if not final:
        # last non-empty AI text
        for msg in reversed(messages):
            if isinstance(msg, AIMessage):
                t = _content_text(msg.content).strip()
                if t:
                    final = t
                    break
    # Deduplicate tools preserving order
    seen: set[str] = set()
    uniq_tools: list[str] = []
    for t in tools:
        if t not in seen:
            seen.add(t)
            uniq_tools.append(t)
    # Thinking: drop exact duplicates of final reply
    uniq_think: list[str] = []
    for t in thinking:
        if t == final:
            continue
        if t not in uniq_think:
            uniq_think.append(t)
    return final, uniq_tools, uniq_think[:12]


def _fmt_tool_args(args: dict[str, Any]) -> str:
    bits: list[str] = []
    for key, value in list(args.items())[:5]:
        text = str(value)
        if len(text) > 60:
            text = text[:57] + "…"
        bits.append(f"{key}={text}")
    return ", ".join(bits)


def _preview_output(result: object) -> str:
    content = getattr(result, "content", result)
    preview = _content_text(content).replace("\n", " ").strip()
    if len(preview) > 100:
        preview = preview[:97] + "…"
    return preview


def _emit_tool_call(item: Any, specialist: str | None) -> None:
    name = str(getattr(item, "tool_name", None) or "unnamed_tool")
    raw_args = getattr(item, "input", None) or {}
    args = raw_args if isinstance(raw_args, dict) else {}
    label = f"{specialist} → {name}" if specialist else name
    emit("tool", f"{label}({_fmt_tool_args(args)})" if args else label)
    emit("status", f"Calling {name}…")
    # v3 is pull-based: item.output stays None until output_deltas is consumed.
    # Our tools return one complete string, so discard chunks and just wait.
    for _ in getattr(item, "output_deltas", ()) or ():
        pass
    preview = _preview_output(getattr(item, "output", None))
    emit_tool_outcome(name, preview)
    emit("status", f"{name} failed" if is_tool_error_text(preview) else f"{name} done")


def run_agent_with_progress(
    graph: Any,
    messages: list[BaseMessage],
    config: RunnableConfig,
    *,
    specialist: str | None = None,
) -> list[BaseMessage]:
    """Run a create_agent graph via stream_events v3 and emit Process events."""
    with warnings.catch_warnings():
        warnings.filterwarnings(
            "ignore",
            message="The v3 streaming protocol on Pregel is experimental.",
        )
        stream = graph.stream_events({"messages": messages}, config, version="v3")
        for kind, item in stream.interleave("tool_calls", "messages"):
            if kind == "tool_calls":
                _emit_tool_call(item, specialist)
            elif kind == "messages":
                text = str(getattr(item, "text", None) or "").strip()
                if text:
                    emit("status", "Composing answer…")
        out = stream.output or {}
    return list(out.get("messages") or [])


class Orchestrator:
    """Top-level Swyngora AI orchestrator (LangGraph desk state machine)."""

    def __init__(
        self,
        settings: Settings | None = None,
        memory: SessionMemory | None = None,
        model: Any | None = None,
    ) -> None:
        self.settings = settings or get_settings()
        if memory is not None:
            self.memory = memory
        else:
            self.memory = FinMem(path=self.settings.memory_path)
        injected = model is not None
        self.model = model or build_chat_model(self.settings)
        self._middleware = [MODEL_RETRY] if injected else build_agent_middleware(self.settings)
        self._runner = SpecialistRunner(self.model, self.settings, middleware=self._middleware)
        tools = self._runner.as_orchestrator_tools()
        system = ORCHESTRATOR_SYSTEM.format(client_id=self.settings.default_client_id)
        self._graph = create_agent(
            self.model,
            tools,
            system_prompt=system,
            middleware=self._middleware,
        )

    def chat(
        self,
        user_message: str,
        session_id: str = "default",
        on_event=None,
        client_id: str = "",
        *,
        can_trade: bool = True,
        can_manage_keys: bool = True,
    ) -> ChatResult:
        """Run one user turn; returns answer + tool/thinking trace.

        on_event: optional callback(dict) for live updates (type/text).
        can_trade / can_manage_keys gate HTTP tools when the Go proxy uses master auth.
        """
        tools_acc: list[str] = []
        thinking_acc: list[str] = []

        def _cb(ev: dict[str, Any]) -> None:
            t = ev.get("type") or ""
            text = (ev.get("text") or "").strip()
            if t in ("tool", "tool_result", "tool_error") and text:
                tools_acc.append(text)
            if t in ("thinking", "status") and text:
                thinking_acc.append(text)
            if on_event is not None:
                on_event(ev)

        token = set_progress(_cb)
        cid = (client_id or "").strip() or (self.settings.default_client_id or "").strip()
        public_session = (session_id or "default").strip() or "default"
        mem_key = memory_key(cid, public_session)
        bind_tok = bind_client_id(cid)
        scope_toks = bind_tool_scope(can_trade=can_trade, can_manage_keys=can_manage_keys)
        json_tok = begin_tool_json_turn()
        try:
            emit("status", "Planning…")
            history = self.memory.get(mem_key)
            messages: list[BaseMessage] = list(history) + [HumanMessage(content=user_message)]
            if _should_refresh_tape(self.memory, cid, user_message):
                emit("status", "Refreshing stale tape…")
                tape_blobs = prefetch_tape(self.settings, user_message)
                for blob in tape_blobs:
                    record_tool_json(blob)
                if tape_blobs:
                    messages.insert(
                        -1,
                        SystemMessage(
                            content=(
                                "Fresh market tape JSON (prefer over older chat numbers):\n"
                                + "\n".join(tape_blobs[:6])
                            )
                        ),
                    )
                    if isinstance(self.memory, FinMem):
                        self.memory.note_tape(cid)
            config: RunnableConfig = {
                "recursion_limit": max(24, self.settings.max_agent_iterations * 4)
            }

            emit("status", "Orchestrator running…")
            out_msgs = run_agent_with_progress(self._graph, messages, config)
            turn_slice = out_msgs[len(history) :] if history else out_msgs
            turn_msgs = turn_slice if turn_slice else out_msgs
            reply, tools_msg, thinking_msg = extract_trace(turn_msgs if turn_msgs else out_msgs)
            if not reply:
                reply = empty_reply_for(user_message)

            if not tools_acc:
                tools_acc.extend(tools_msg)
            for t in thinking_msg:
                if t not in thinking_acc:
                    thinking_acc.append(t)

            blobs = collected_tool_json()
            if not blobs:
                blobs = [
                    _content_text(getattr(m, "content", ""))
                    for m in turn_msgs
                    if isinstance(m, ToolMessage)
                ]
            facts = extract_market_facts(*blobs)
            if facts.last_price or facts.rsi:
                reply = apply_grounding(reply, facts, blobs, user_message=user_message)

            route = classify_route(user_message)
            if route.desk_note and not has_advice_disclaimer(reply):
                reply = f"{reply}\n\n{disclaimer_for(user_message)}"

            self.memory.append(mem_key, as_human_ai(user_message, reply))
            if isinstance(self.memory, FinMem):
                if facts.symbol or facts.last_price:
                    self.memory.note_tape(cid, facts.symbol, facts.exchange)
                self.memory.persist_turn(cid, user_message, reply)

            refs = grounded_references(*blobs, reply=reply)
            emit("status", "Composing answer…")
            return ChatResult(
                reply=reply,
                tools=tools_acc,
                thinking=thinking_acc,
                session_id=public_session,
                references=[r.as_dict() for r in refs],
            )
        finally:
            reset_tool_json_turn(json_tok)
            reset_tool_scope(scope_toks)
            reset_bound_client_id(bind_tok)
            reset_progress(token)

    def reset(self, session_id: str = "default", client_id: str = "") -> None:
        cid = (client_id or "").strip() or (self.settings.default_client_id or "").strip()
        public_session = (session_id or "default").strip() or "default"
        self.memory.clear(memory_key(cid, public_session))


def build_orchestrator(settings: Settings | None = None) -> Orchestrator:
    return Orchestrator(settings=settings)
