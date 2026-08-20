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
            msg = self.responses[-1] if self.responses else AIMessage(content="done")
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
    assert names == {
        "market_tape_agent",
        "market_book_agent",
        "paper_desk_agent",
        "account_agent",
        "web_agent",
        "x_agent",
        "analyst_agent",
    }


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
    events: list[dict[str, Any]] = []
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


def _scripted_ticker_agent(result: str, *, final: str = "done"):
    from langchain.agents import create_agent
    from langchain_core.messages import HumanMessage
    from langchain_core.tools import StructuredTool

    from swyngora_ai.graph.orchestrator import run_agent_with_progress
    from swyngora_ai.progress import reset_progress, set_progress

    def get_ticker(symbol: str = "BTCUSDT") -> str:
        return result

    model = ScriptedModel(
        responses=[
            AIMessage(
                content="",
                tool_calls=[
                    {
                        "name": "get_ticker",
                        "args": {"symbol": "BTCUSDT"},
                        "id": "1",
                        "type": "tool_call",
                    }
                ],
            ),
            AIMessage(content=final),
        ]
    )
    graph = create_agent(
        model,
        [StructuredTool.from_function(get_ticker, name="get_ticker", description="ticker")],
        system_prompt="tape",
    )
    events: list[dict[str, Any]] = []
    token = set_progress(events.append)
    try:
        out = run_agent_with_progress(
            graph,
            [HumanMessage(content="btc")],
            {"recursion_limit": 8},
            specialist="market_tape_agent",
        )
    finally:
        reset_progress(token)
    return out, events


def test_content_text_skips_reasoning_blocks():
    from swyngora_ai.graph.orchestrator import _content_text

    raw = [
        {"type": "reasoning", "reasoning": "The user asked…", "index": 0},
        {"type": "tool_call", "name": "market_tape_agent", "args": {"task": "BTC"}},
        {"type": "text", "text": "BTCUSDT last is 100."},
    ]
    assert _content_text(raw) == "BTCUSDT last is 100."


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


def test_run_agent_emits_one_prefixed_tool_result():
    out, events = _scripted_ticker_agent('{"lastPrice":"100"}')
    assert out
    assert out[-1].content == "done"
    tool_lines = [e.get("text") or "" for e in events if e.get("type") == "tool"]
    result_lines = [e.get("text") or "" for e in events if e.get("type") == "tool_result"]
    assert tool_lines == ["market_tape_agent → get_ticker(symbol=BTCUSDT)"]
    assert len(result_lines) == 1
    assert "get_ticker ✓" in result_lines[0]
    assert any(e.get("type") == "status" for e in events)


def test_run_agent_treats_error_string_as_failure():
    _out, events = _scripted_ticker_agent(
        "ERROR connect: [Errno 111] Connection refused",
        final="down",
    )
    tool_lines = [e.get("text") or "" for e in events if e.get("type") == "tool"]
    assert tool_lines == ["market_tape_agent → get_ticker(symbol=BTCUSDT)"]
    err_lines = [e.get("text") or "" for e in events if e.get("type") == "tool_error"]
    assert len(err_lines) == 1
    assert "get_ticker failed:" in err_lines[0]
    assert "ERROR connect" in err_lines[0]
    assert "✓" not in err_lines[0]
    assert any((e.get("text") or "") == "get_ticker failed" for e in events)
    assert not any("get_ticker ✓" in (e.get("text") or "") for e in events)


def test_specialist_runner_keeps_original_leaf_tools():
    from swyngora_ai.agents.specialists import SpecialistRunner
    from swyngora_ai.tools.market_http import build_market_tools

    model = ScriptedModel(responses=[AIMessage(content="ok")])
    settings = Settings(_env_file=None)
    runner = SpecialistRunner(model, settings)
    expected = {t.name for t in build_market_tools(settings, pack="tape")}
    assert {t.name for t in runner._tools["market_tape_agent"]} == expected
