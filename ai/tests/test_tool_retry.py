"""ToolRetryMiddleware retries ToolTransientError then fail-softs as ERROR …"""

from __future__ import annotations

import pytest
from langchain.agents import create_agent
from langchain.agents.middleware import ToolRetryMiddleware
from langchain_core.messages import AIMessage, HumanMessage, ToolMessage
from langchain_core.tools import StructuredTool

from swyngora_ai.llm.retry import TOOL_RETRY, format_tool_retry_failure
from swyngora_ai.tools.errors import ToolTransientError
from test_specialists_and_orchestrator import ScriptedModel


def test_format_tool_retry_failure_prefixes_error() -> None:
    assert format_tool_retry_failure(ToolTransientError("timeout: blip")).startswith(
        "ERROR timeout"
    )
    assert format_tool_retry_failure(ToolTransientError("ERROR 502: down")).startswith("ERROR 502")


def test_tool_retry_constant() -> None:
    assert isinstance(TOOL_RETRY, ToolRetryMiddleware)
    assert TOOL_RETRY.max_retries == 2
    assert TOOL_RETRY.retry_on == (ToolTransientError,)


def _fast_tool_retry() -> ToolRetryMiddleware:
    return ToolRetryMiddleware(
        max_retries=2,
        retry_on=(ToolTransientError,),
        on_failure=format_tool_retry_failure,
        initial_delay=0.0,
        jitter=False,
    )


def _flaky_ticker(fails: int, *, error: Exception | None = None):
    state = {"n": 0}

    def get_ticker(symbol: str = "BTCUSDT") -> str:
        state["n"] += 1
        if state["n"] <= fails:
            raise error or ToolTransientError("timeout: blip")
        return '{"lastPrice":"100"}'

    return get_ticker, state


def _ticker_agent(fn, final: str = "done"):
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
    return create_agent(
        model,
        [StructuredTool.from_function(fn, name="get_ticker", description="ticker")],
        system_prompt="t",
        middleware=[_fast_tool_retry()],
    )


def test_create_agent_retries_then_returns_json() -> None:
    fn, state = _flaky_ticker(2)
    result = _ticker_agent(fn, final="recovered").invoke(
        {"messages": [HumanMessage(content="q")]}, {"recursion_limit": 8}
    )
    msgs = list((result or {}).get("messages") or [])
    tools = [m for m in msgs if isinstance(m, ToolMessage)]
    assert state["n"] == 3
    assert tools
    assert "100" in str(tools[0].content)


def test_create_agent_exhausted_returns_error_string() -> None:
    fn, state = _flaky_ticker(10)
    result = _ticker_agent(fn, final="could not fetch").invoke(
        {"messages": [HumanMessage(content="q")]}, {"recursion_limit": 8}
    )
    msgs = list((result or {}).get("messages") or [])
    tools = [m for m in msgs if isinstance(m, ToolMessage)]
    assert state["n"] == 3
    assert tools
    assert str(tools[0].content).startswith("ERROR timeout")
    texts = [str(getattr(m, "content", "")) for m in msgs]
    assert any("could not fetch" in t for t in texts)


def test_non_transient_error_is_not_retried() -> None:
    fn, state = _flaky_ticker(10, error=ValueError("bad symbol"))
    with pytest.raises(ValueError, match="bad symbol"):
        _ticker_agent(fn).invoke({"messages": [HumanMessage(content="q")]}, {"recursion_limit": 8})
    assert state["n"] == 1
