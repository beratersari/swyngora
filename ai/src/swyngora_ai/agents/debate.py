"""Optional bull/bear notes — only when the user asks to lean long/short."""

from __future__ import annotations

from typing import Any

from langchain_core.messages import HumanMessage, SystemMessage

from swyngora_ai.agents.prompts import BEAR_SYSTEM, BULL_SYSTEM
from swyngora_ai.progress import emit


def _content(resp: Any) -> str:
    content = getattr(resp, "content", resp)
    if isinstance(content, list):
        return " ".join(str(c) for c in content)
    return str(content)


def run_debate(model: Any, packet: str) -> tuple[str, str]:
    emit("status", "Bull / bear debate…")
    emit("thinking", "Structured lean requested — running bull and bear notes")
    bull = _content(
        model.invoke([SystemMessage(content=BULL_SYSTEM), HumanMessage(content=packet)])
    )
    bear = _content(
        model.invoke([SystemMessage(content=BEAR_SYSTEM), HumanMessage(content=packet)])
    )
    emit("status", "Debate finished")
    return bull, bear
