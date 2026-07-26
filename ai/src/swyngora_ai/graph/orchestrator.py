"""Hierarchical multi-agent orchestrator (LangGraph supervisor / subagents-as-tools)."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from langchain.agents import create_agent
from langchain_core.messages import AIMessage, BaseMessage, HumanMessage, ToolMessage

from swyngora_ai.agents.prompts import ORCHESTRATOR_SYSTEM
from swyngora_ai.agents.specialists import build_specialist_tools
from swyngora_ai.config import Settings, get_settings
from swyngora_ai.llm.factory import build_chat_model
from swyngora_ai.progress import emit, reset_progress, set_progress


@dataclass
class ChatResult:
    """Final answer plus transparent agent trace for UIs (Telegram, etc.)."""

    reply: str
    tools: list[str] = field(default_factory=list)
    thinking: list[str] = field(default_factory=list)
    session_id: str = "default"


@dataclass
class SessionMemory:
    """Simple per-session message buffer (not durable)."""

    sessions: dict[str, list[BaseMessage]] = field(default_factory=dict)
    max_messages: int = 40

    def get(self, session_id: str) -> list[BaseMessage]:
        return list(self.sessions.get(session_id, []))

    def append(self, session_id: str, messages: list[BaseMessage]) -> None:
        buf = self.sessions.setdefault(session_id, [])
        buf.extend(messages)
        if len(buf) > self.max_messages:
            self.sessions[session_id] = buf[-self.max_messages :]

    def clear(self, session_id: str) -> None:
        self.sessions.pop(session_id, None)


def _content_text(content: Any) -> str:
    if content is None:
        return ""
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        parts: list[str] = []
        for block in content:
            if isinstance(block, dict) and block.get("type") == "text":
                parts.append(str(block.get("text", "")))
            elif isinstance(block, str):
                parts.append(block)
            else:
                parts.append(str(block))
        return "\n".join(p for p in parts if p)
    return str(content)


def extract_trace(messages: list[BaseMessage]) -> tuple[str, list[str], list[str]]:
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


class Orchestrator:
    """Top-level Swyngora AI orchestrator."""

    def __init__(
        self,
        settings: Settings | None = None,
        memory: SessionMemory | None = None,
        model: Any | None = None,
    ) -> None:
        self.settings = settings or get_settings()
        self.memory = memory or SessionMemory()
        self.model = model or build_chat_model(self.settings)
        tools = build_specialist_tools(self.model, self.settings)
        system = ORCHESTRATOR_SYSTEM.format(client_id=self.settings.default_client_id)
        self._graph = create_agent(self.model, tools, system_prompt=system)

    def chat(
        self,
        user_message: str,
        session_id: str = "default",
        on_event=None,
    ) -> ChatResult:
        """Run one user turn; returns answer + tool/thinking trace.

        on_event: optional callback(dict) for live updates (type/text).
        """
        tools_acc: list[str] = []
        thinking_acc: list[str] = []

        def _cb(ev: dict) -> None:
            t = (ev.get("type") or "")
            text = (ev.get("text") or "").strip()
            if t in ("tool", "tool_result", "tool_error") and text:
                tools_acc.append(text)
            if t in ("thinking", "status") and text:
                thinking_acc.append(text)
            if on_event is not None:
                on_event(ev)

        token = set_progress(_cb)
        try:
            emit("status", "Planning…")
            history = self.memory.get(session_id)
            messages: list[BaseMessage] = list(history) + [HumanMessage(content=user_message)]
            config = {"recursion_limit": max(24, self.settings.max_agent_iterations * 4)}

            # invoke is the reliable path for a final answer; progress emits from
            # specialist tools + a lightweight stream-of-updates for live UI only.
            # (stream-only accumulation previously risked missing the final AI text.)
            out_msgs: list[BaseMessage] = []
            emit("status", "Orchestrator running…")

            # Optional live updates via stream in a side channel would need threads;
            # keep a single invoke so deep analysis always terminates with a reply.
            result = self._graph.invoke({"messages": messages}, config=config)
            out_msgs = list(result.get("messages") or [])
            # Emit tool/thinking from the completed transcript for UIs that missed live wraps.
            turn_slice = out_msgs[len(history) :] if history else out_msgs
            for msg in turn_slice:
                self._emit_from_message(msg)

            turn_msgs = turn_slice if turn_slice else out_msgs
            reply, tools_msg, thinking_msg = extract_trace(turn_msgs if turn_msgs else out_msgs)
            if not reply:
                reply = (
                    "I could not produce an answer. "
                    "Try rephrasing or /ask with a clearer symbol (e.g. JUVUSDT)."
                )

            # Merge live + post-hoc traces
            for t in tools_msg:
                if t not in tools_acc:
                    tools_acc.append(t)
            for t in thinking_msg:
                if t not in thinking_acc:
                    thinking_acc.append(t)

            final_msg = AIMessage(content=reply)
            self.memory.append(session_id, [HumanMessage(content=user_message), final_msg])

            if any(
                k in user_message.lower()
                for k in ("price", "btc", "eth", "buy", "sell", "mcap", "rsi", "analysis", "juv")
            ):
                if "not financial advice" not in reply.lower():
                    reply = f"{reply}\n\n{self.settings.disclaimer}"

            emit("final", reply[:200])
            return ChatResult(
                reply=reply,
                tools=tools_acc,
                thinking=thinking_acc,
                session_id=session_id,
            )
        finally:
            reset_progress(token)

    @staticmethod
    def _emit_from_message(msg: BaseMessage) -> None:
        """Push live events from streamed agent messages."""
        if isinstance(msg, AIMessage):
            text = _content_text(msg.content).strip()
            tcalls = getattr(msg, "tool_calls", None) or []
            if tcalls:
                if text:
                    emit("thinking", text[:300])
                for tc in tcalls:
                    name = tc.get("name") if isinstance(tc, dict) else getattr(tc, "name", "?")
                    args = tc.get("args") if isinstance(tc, dict) else getattr(tc, "args", {})
                    arg_s = ""
                    if isinstance(args, dict):
                        bits = []
                        for k, v in list(args.items())[:4]:
                            vs = str(v)
                            if len(vs) > 60:
                                vs = vs[:57] + "…"
                            bits.append(f"{k}={vs}")
                        arg_s = ", ".join(bits)
                    emit("tool", f"{name}({arg_s})" if arg_s else str(name))
                    emit("status", f"Calling {name}…")
            elif text:
                emit("status", "Composing answer…")
        elif isinstance(msg, ToolMessage):
            name = getattr(msg, "name", None) or "tool"
            preview = _content_text(msg.content).replace("\n", " ").strip()
            if len(preview) > 100:
                preview = preview[:97] + "…"
            emit("tool_result", f"{name} ✓ {preview}" if preview else f"{name} ✓")
            emit("status", f"{name} done")

    def reset(self, session_id: str = "default") -> None:
        self.memory.clear(session_id)


def build_orchestrator(settings: Settings | None = None) -> Orchestrator:
    return Orchestrator(settings=settings)
