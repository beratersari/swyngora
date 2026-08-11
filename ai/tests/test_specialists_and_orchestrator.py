"""Orchestrator tests with a fake chat model (no live LLM)."""

from __future__ import annotations

from typing import Any

from langchain_core.language_models.chat_models import BaseChatModel
from langchain_core.messages import AIMessage
from langchain_core.outputs import ChatGeneration, ChatResult

from swyngora_ai.agents.specialists import build_specialist_tools
from swyngora_ai.config import Settings
from swyngora_ai.graph.orchestrator import Orchestrator, SessionMemory


class ScriptedModel(BaseChatModel):
    """Minimal chat model that returns scripted AIMessages (optionally with tool calls)."""

    responses: list[AIMessage]
    _i: int = 0

    @property
    def _llm_type(self) -> str:
        return "scripted"

    def _generate(
        self,
        messages: list[Any],
        stop: list[str] | None = None,
        run_manager: Any = None,
        **kwargs: Any,
    ) -> ChatResult:
        if self._i >= len(self.responses):
            msg = AIMessage(content="done")
        else:
            msg = self.responses[self._i]
            self._i += 1
        return ChatResult(generations=[ChatGeneration(message=msg)])

    def bind_tools(self, tools: Any, **kwargs: Any) -> ScriptedModel:
        # Return self so create_react_agent can bind tools without changing behaviour.
        return self


def test_build_specialist_tools_names():
    model = ScriptedModel(responses=[AIMessage(content="ok")])
    tools = build_specialist_tools(model, Settings(_env_file=None))
    names = {t.name for t in tools}
    assert names == {"market_agent", "web_agent", "x_agent", "analyst_agent"}


def test_orchestrator_chat_with_scripted_model():
    model = ScriptedModel(
        responses=[
            AIMessage(content="BTC is around 100 (scripted). Not financial advice."),
        ]
    )
    orch = Orchestrator(
        settings=Settings(_env_file=None),
        memory=SessionMemory(),
        model=model,
    )
    out = orch.chat("What is BTC price?", session_id="t1")
    assert out.reply
    assert isinstance(out.tools, list)
    assert isinstance(out.thinking, list)
    assert isinstance(out.references, list)


def test_orchestrator_emits_live_process_events():
    events: list[dict] = []
    model = ScriptedModel(
        responses=[
            AIMessage(content="BTC is around 100 (scripted). Not financial advice."),
        ]
    )
    orch = Orchestrator(
        settings=Settings(_env_file=None),
        memory=SessionMemory(),
        model=model,
    )
    out = orch.chat("What is BTC price?", session_id="t-live", on_event=events.append)
    assert out.reply
    types = [e.get("type") for e in events]
    assert "status" in types
    assert "final" not in types  # structured final is added by the HTTP layer
    texts = " ".join(str(e.get("text") or "") for e in events)
    assert "Planning" in texts or "Orchestrator" in texts or "Composing" in texts


def test_run_agent_with_progress_streams_then_emits():
    from langchain_core.messages import AIMessage, HumanMessage, ToolMessage
    from swyngora_ai.graph.orchestrator import run_agent_with_progress
    from swyngora_ai.progress import reset_progress, set_progress

    human = HumanMessage(content="q")
    think = AIMessage(
        content="I'll check market data",
        tool_calls=[{"name": "market_agent", "args": {"task": "BTC"}, "id": "1", "type": "tool_call"}],
    )
    tool = ToolMessage(content='{"lastPrice":"1"}', name="market_agent", tool_call_id="1")
    final = AIMessage(content="done")

    class FakeGraph:
        def stream(self, _input, config=None, stream_mode=None):
            assert stream_mode == "values"
            yield {"messages": [human]}
            yield {"messages": [human, think]}
            yield {"messages": [human, think, tool, final]}

        def invoke(self, _input, config=None):
            raise AssertionError("invoke should not run when stream works")

    events: list[dict] = []
    token = set_progress(events.append)
    try:
        out = run_agent_with_progress(FakeGraph(), [human], {"recursion_limit": 4})
    finally:
        reset_progress(token)
    assert out[-1].content == "done"
    types = [e.get("type") for e in events]
    assert "thinking" in types
    assert "tool" in types
    assert "tool_result" in types
    assert "status" in types


def test_extract_trace_tools():
    from langchain_core.messages import AIMessage, HumanMessage, ToolMessage

    from swyngora_ai.graph.orchestrator import extract_trace

    msgs = [
        HumanMessage(content="q"),
        AIMessage(
            content="I'll check market data",
            tool_calls=[
                {"name": "market_agent", "args": {"task": "JUV"}, "id": "1", "type": "tool_call"}
            ],
        ),
        ToolMessage(content='{"lastPrice":"1"}', name="market_agent", tool_call_id="1"),
        AIMessage(content="JUV looks quiet."),
    ]
    reply, tools, thinking = extract_trace(msgs)
    assert "quiet" in reply
    assert any("market_agent" in t for t in tools)
    assert thinking


def test_session_memory():
    mem = SessionMemory(max_messages=4)
    from langchain_core.messages import AIMessage, HumanMessage

    mem.append("s", [HumanMessage(content="a"), AIMessage(content="b")])
    mem.append("s", [HumanMessage(content="c"), AIMessage(content="d")])
    mem.append("s", [HumanMessage(content="e"), AIMessage(content="f")])
    assert len(mem.get("s")) == 4  # capped
