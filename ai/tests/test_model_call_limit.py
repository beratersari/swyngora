"""ModelCallLimitMiddleware stops a looping create_agent after run_limit steps."""

from __future__ import annotations

from typing import Any

from langchain.agents import create_agent
from langchain.agents.middleware import ModelCallLimitMiddleware
from langchain_core.language_models.chat_models import BaseChatModel
from langchain_core.messages import AIMessage, HumanMessage
from langchain_core.outputs import ChatGeneration, ChatResult
from langchain_core.tools import StructuredTool

from swyngora_ai.agents.specialists import SpecialistRunner
from swyngora_ai.config import Settings
from swyngora_ai.graph.orchestrator import Orchestrator, SessionMemory, extract_trace
from swyngora_ai.llm.retry import (
    MODEL_CALL_LIMIT,
    MODEL_CALL_LIMIT_PREFIX,
    MODEL_CALL_LIMIT_RUN,
    MODEL_RETRY,
    TOOL_RETRY,
    default_agent_middleware,
    is_model_call_limit_text,
)
from swyngora_ai.progress import reset_progress, set_progress
from test_specialists_and_orchestrator import ScriptedModel


class _LoopModel(BaseChatModel):
    """Always requests another tool call so the agent would loop without a cap."""

    calls: int = 0

    @property
    def _llm_type(self) -> str:
        return "loop"

    def _generate(
        self,
        messages: list[Any],
        stop: list[str] | None = None,
        run_manager: Any = None,
        **kwargs: Any,
    ) -> ChatResult:
        self.calls += 1
        msg = AIMessage(
            content="",
            tool_calls=[
                {
                    "name": "ping",
                    "args": {},
                    "id": str(self.calls),
                    "type": "tool_call",
                }
            ],
        )
        return ChatResult(generations=[ChatGeneration(message=msg)])

    def bind_tools(self, tools: Any, **kwargs: Any) -> _LoopModel:
        return self


def _ping() -> str:
    return "pong"


def test_model_call_limit_constant() -> None:
    assert isinstance(MODEL_CALL_LIMIT, ModelCallLimitMiddleware)
    assert MODEL_CALL_LIMIT.run_limit == MODEL_CALL_LIMIT_RUN == 6
    assert MODEL_CALL_LIMIT.thread_limit is None
    assert MODEL_CALL_LIMIT.exit_behavior == "end"


def test_default_and_injected_middleware_include_call_limit() -> None:
    assert default_agent_middleware() == [MODEL_RETRY, TOOL_RETRY, MODEL_CALL_LIMIT]
    orch = Orchestrator(
        settings=Settings(_env_file=None),
        memory=SessionMemory(),
        model=ScriptedModel(responses=[AIMessage(content="ok. Not financial advice.")]),
    )
    assert orch._middleware == [MODEL_RETRY, TOOL_RETRY, MODEL_CALL_LIMIT]
    runner = SpecialistRunner(
        ScriptedModel(responses=[AIMessage(content="ok")]),
        Settings(_env_file=None),
    )
    assert MODEL_CALL_LIMIT in runner._middleware


def test_create_agent_stops_at_run_limit() -> None:
    model = _LoopModel()
    graph = create_agent(
        model,
        [StructuredTool.from_function(_ping, name="ping", description="ping")],
        system_prompt="t",
        middleware=[
            ModelCallLimitMiddleware(run_limit=2, exit_behavior="end"),
        ],
    )
    result = graph.invoke({"messages": [HumanMessage(content="q")]}, {"recursion_limit": 16})
    msgs = list((result or {}).get("messages") or [])
    texts = [str(getattr(m, "content", "")) for m in msgs]
    assert model.calls == 2
    assert any(t.startswith(MODEL_CALL_LIMIT_PREFIX) for t in texts)


def test_extract_trace_emits_limit_status() -> None:
    events: list[dict[str, Any]] = []
    token = set_progress(events.append)
    try:
        reply, _tools, _think = extract_trace(
            [
                HumanMessage(content="q"),
                AIMessage(content="Model call limits exceeded: run limit (6/6)"),
            ]
        )
    finally:
        reset_progress(token)
    assert is_model_call_limit_text(reply)
    assert any(
        e.get("type") == "status" and e.get("text") == "Model call limit reached…" for e in events
    )
