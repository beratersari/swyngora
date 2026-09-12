"""ToolRetryMiddleware retries ToolTransientError then fail-softs as ERROR …"""

from __future__ import annotations

from typing import Any

import pytest
from langchain.agents import create_agent
from langchain.agents.middleware import ToolRetryMiddleware
from langchain_core.messages import AIMessage, HumanMessage, ToolMessage
from langchain_core.tools import StructuredTool

from swyngora_ai.graph.orchestrator import run_agent_with_progress
from swyngora_ai.llm.errors import NoteToolError
from swyngora_ai.llm.retry import TOOL_RETRY, format_tool_retry_failure
from swyngora_ai.progress import (
    as_tool_error_preview,
    emit_tool_outcome,
    reset_progress,
    set_progress,
)
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


def _run_progress(graph) -> list[dict[str, Any]]:
    events: list[dict[str, Any]] = []
    token = set_progress(events.append)
    try:
        run_agent_with_progress(
            graph,
            [HumanMessage(content="btc")],
            {"recursion_limit": 8},
            specialist="market_tape_agent",
        )
    finally:
        reset_progress(token)
    return events


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


def test_note_tool_error_emits_retry_then_exhausted(monkeypatch) -> None:
    seen: list[tuple[str, str]] = []
    monkeypatch.setattr("swyngora_ai.llm.errors.emit", lambda t, text="": seen.append((t, text)))
    note = NoteToolError()
    note.on_tool_start({"name": "get_ticker"}, "")
    note.on_tool_error(ToolTransientError("timeout: blip"))
    note.on_tool_error(ToolTransientError("timeout: blip"))
    assert seen == [
        ("tool_retry", "get_ticker"),
        ("status", "Retrying get_ticker (2/3)…"),
        ("tool_retry", "get_ticker"),
        ("status", "Retrying get_ticker (3/3)…"),
    ]
    note.on_tool_error(ToolTransientError("timeout: blip"))
    assert seen[-1] == ("status", "get_ticker failed after 3 tries")


def test_note_tool_error_ignores_non_transient(monkeypatch) -> None:
    seen: list[str] = []
    monkeypatch.setattr("swyngora_ai.llm.errors.emit", lambda _t, text="": seen.append(text))
    note = NoteToolError()
    note.on_tool_start({"name": "get_ticker"}, "")
    note.on_tool_error(ValueError("bad symbol"))
    assert seen == []
    assert note.attempts == 0


def test_note_tool_error_tracks_parallel_tool_call_ids(monkeypatch) -> None:
    seen: list[str] = []
    monkeypatch.setattr("swyngora_ai.llm.errors.emit", lambda _t, text="": seen.append(text))
    note = NoteToolError()
    note.on_tool_start({"name": "get_ticker"}, "", tool_call_id="a")
    note.on_tool_start({"name": "get_indicators"}, "", tool_call_id="b")
    note.on_tool_error(ToolTransientError("timeout: t"), tool_call_id="a")
    note.on_tool_error(ToolTransientError("timeout: i"), tool_call_id="b")
    assert seen.count("Retrying get_ticker (2/3)…") == 1
    assert seen.count("Retrying get_indicators (2/3)…") == 1


def test_note_tool_error_resets_when_tool_name_changes(monkeypatch) -> None:
    seen: list[tuple[str, str]] = []
    monkeypatch.setattr("swyngora_ai.llm.errors.emit", lambda t, text="": seen.append((t, text)))
    note = NoteToolError()
    note.on_tool_start({"name": "get_ticker"}, "")
    note.on_tool_error(ToolTransientError("timeout: a"))
    note.on_tool_start({"name": "get_indicators"}, "")
    note.on_tool_error(ToolTransientError("timeout: b"))
    assert seen == [
        ("tool_retry", "get_ticker"),
        ("status", "Retrying get_ticker (2/3)…"),
        ("tool_retry", "get_indicators"),
        ("status", "Retrying get_indicators (2/3)…"),
    ]


def test_as_tool_error_preview_prefixes() -> None:
    assert as_tool_error_preview("connect: refused") == "ERROR connect: refused"
    assert as_tool_error_preview("ERROR 502") == "ERROR 502"
    assert as_tool_error_preview("") == "ERROR"


def test_emit_tool_outcome_empty_is_failure(monkeypatch) -> None:
    seen: list[tuple[str, str]] = []
    monkeypatch.setattr("swyngora_ai.progress.emit", lambda t, text="": seen.append((t, text)))
    emit_tool_outcome("get_ticker", "")
    assert seen == [("tool_error", "get_ticker failed")]


def test_run_agent_exhausted_raise_is_failure() -> None:
    fn, state = _flaky_ticker(10)
    events = _run_progress(_ticker_agent(fn, final="down"))
    tool_lines = [e.get("text") or "" for e in events if e.get("type") == "tool"]
    retry_lines = [e.get("text") or "" for e in events if e.get("type") == "tool_retry"]
    err_lines = [e.get("text") or "" for e in events if e.get("type") == "tool_error"]
    assert tool_lines.count("market_tape_agent → get_ticker(symbol=BTCUSDT)") == 1
    assert retry_lines.count("get_ticker") == 2
    assert len(err_lines) == 1
    assert "get_ticker failed:" in err_lines[0]
    assert "ERROR" in err_lines[0]
    assert "✓" not in err_lines[0]
    assert not any("get_ticker ✓" in (e.get("text") or "") for e in events)
    assert any((e.get("text") or "") == "get_ticker failed" for e in events)
    assert [e.get("text") for e in events if e.get("text") == "Calling get_ticker…"] == [
        "Calling get_ticker…"
    ]
    assert state["n"] == 3


def test_run_agent_recover_after_retry_one_tick() -> None:
    fn, state = _flaky_ticker(2)
    events = _run_progress(_ticker_agent(fn, final="ok"))
    tool_lines = [e.get("text") or "" for e in events if e.get("type") == "tool"]
    retry_lines = [e.get("text") or "" for e in events if e.get("type") == "tool_retry"]
    result_lines = [e.get("text") or "" for e in events if e.get("type") == "tool_result"]
    assert tool_lines.count("market_tape_agent → get_ticker(symbol=BTCUSDT)") == 1
    assert retry_lines.count("get_ticker") == 2
    assert len(result_lines) == 1
    assert "get_ticker ✓" in result_lines[0]
    assert not any(e.get("type") == "tool_error" for e in events)
    assert state["n"] == 3


def test_run_agent_exhausted_parallel_tools_no_ticks() -> None:
    ticker_n = {"n": 0}
    indicators_n = {"n": 0}

    def get_ticker(symbol: str = "BTCUSDT") -> str:
        ticker_n["n"] += 1
        raise ToolTransientError("timeout: t")

    def get_indicators(symbol: str = "BTCUSDT") -> str:
        indicators_n["n"] += 1
        raise ToolTransientError("timeout: i")

    model = ScriptedModel(
        responses=[
            AIMessage(
                content="",
                tool_calls=[
                    {
                        "name": "get_ticker",
                        "args": {"symbol": "BTCUSDT"},
                        "id": "a",
                        "type": "tool_call",
                    },
                    {
                        "name": "get_indicators",
                        "args": {"symbol": "BTCUSDT"},
                        "id": "b",
                        "type": "tool_call",
                    },
                ],
            ),
            AIMessage(content="down"),
        ]
    )
    graph = create_agent(
        model,
        [
            StructuredTool.from_function(get_ticker, name="get_ticker", description="ticker"),
            StructuredTool.from_function(
                get_indicators, name="get_indicators", description="indicators"
            ),
        ],
        system_prompt="t",
        middleware=[_fast_tool_retry()],
    )
    events = _run_progress(graph)
    tool_lines = [e.get("text") or "" for e in events if e.get("type") == "tool"]
    retry_lines = [e.get("text") or "" for e in events if e.get("type") == "tool_retry"]
    err_lines = [e.get("text") or "" for e in events if e.get("type") == "tool_error"]
    assert tool_lines.count("market_tape_agent → get_ticker(symbol=BTCUSDT)") == 1
    assert tool_lines.count("market_tape_agent → get_indicators(symbol=BTCUSDT)") == 1
    assert retry_lines.count("get_ticker") == 2
    assert retry_lines.count("get_indicators") == 2
    assert len(err_lines) == 2
    assert any("get_ticker failed:" in line for line in err_lines)
    assert any("get_indicators failed:" in line for line in err_lines)
    assert not any("✓" in (e.get("text") or "") for e in events)
    assert ticker_n["n"] == 3
    assert indicators_n["n"] == 3


def test_run_agent_emits_tool_retry_status() -> None:
    fn, state = _flaky_ticker(2)
    events: list[dict[str, Any]] = []
    token = set_progress(events.append)
    try:
        run_agent_with_progress(
            _ticker_agent(fn, final="ok"),
            [HumanMessage(content="btc")],
            {"recursion_limit": 8},
            specialist="market_tape_agent",
        )
    finally:
        reset_progress(token)
    texts = [e.get("text") or "" for e in events]
    assert "Retrying get_ticker (2/3)…" in texts
    assert "Retrying get_ticker (3/3)…" in texts
    assert any(
        e.get("type") == "status" and "Retrying get_ticker (2/3)" in (e.get("text") or "")
        for e in events
    )
    assert not any(
        e.get("type") == "tool" and "Retrying get_ticker" in (e.get("text") or "") for e in events
    )
    assert state["n"] == 3


def test_run_react_prefixes_error_when_all_leaf_tools_fail() -> None:
    from swyngora_ai.agents.specialists import _run_react

    fn, state = _flaky_ticker(10)
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
            AIMessage(content="Tool failure. No live price."),
        ]
    )
    out = _run_react(
        model,
        [StructuredTool.from_function(fn, name="get_ticker", description="ticker")],
        "tape",
        "btc",
        specialist="market_tape_agent",
        recursion_limit=8,
        middleware=[_fast_tool_retry()],
    )
    assert out.startswith("ERROR")
    assert "Tool failure" in out
    assert state["n"] == 3


def test_run_react_keeps_reply_when_a_leaf_tool_recovers() -> None:
    from swyngora_ai.agents.specialists import _run_react

    fn, state = _flaky_ticker(2)
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
            AIMessage(content="BTC last is 100."),
        ]
    )
    out = _run_react(
        model,
        [StructuredTool.from_function(fn, name="get_ticker", description="ticker")],
        "tape",
        "btc",
        specialist="market_tape_agent",
        recursion_limit=8,
        middleware=[_fast_tool_retry()],
    )
    assert not out.startswith("ERROR")
    assert "100" in out
    assert state["n"] == 3


def test_nested_exhausted_specialist_is_not_ticked() -> None:
    from swyngora_ai.agents.specialists import _run_react

    fn, state = _flaky_ticker(10)

    def market_tape_agent(task: str) -> str:
        return _run_react(
            ScriptedModel(
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
                    AIMessage(content="Tool failure. No live price."),
                ]
            ),
            [StructuredTool.from_function(fn, name="get_ticker", description="ticker")],
            "tape",
            task,
            specialist="market_tape_agent",
            recursion_limit=8,
            middleware=[_fast_tool_retry()],
        )

    model = ScriptedModel(
        responses=[
            AIMessage(
                content="",
                tool_calls=[
                    {
                        "name": "market_tape_agent",
                        "args": {"task": "btc"},
                        "id": "1",
                        "type": "tool_call",
                    }
                ],
            ),
            AIMessage(content="down"),
        ]
    )
    outer = create_agent(
        model,
        [
            StructuredTool.from_function(
                market_tape_agent, name="market_tape_agent", description="tape"
            )
        ],
        system_prompt="t",
    )
    events: list[dict[str, Any]] = []
    token = set_progress(events.append)
    try:
        run_agent_with_progress(outer, [HumanMessage(content="btc")], {"recursion_limit": 8})
    finally:
        reset_progress(token)
    assert not any("market_tape_agent ✓" in (e.get("text") or "") for e in events)
    err_lines = [e.get("text") or "" for e in events if e.get("type") == "tool_error"]
    assert any("market_tape_agent failed:" in line for line in err_lines)
    assert state["n"] == 3


def test_nested_run_agent_emits_retry_once() -> None:
    fn, state = _flaky_ticker(2)
    inner = _ticker_agent(fn, final="ok")

    def market_tape_agent(task: str) -> str:
        run_agent_with_progress(
            inner,
            [HumanMessage(content=task)],
            {"recursion_limit": 8},
            specialist="market_tape_agent",
        )
        return "ok"

    model = ScriptedModel(
        responses=[
            AIMessage(
                content="",
                tool_calls=[
                    {
                        "name": "market_tape_agent",
                        "args": {"task": "btc"},
                        "id": "1",
                        "type": "tool_call",
                    }
                ],
            ),
            AIMessage(content="done"),
        ]
    )
    outer = create_agent(
        model,
        [
            StructuredTool.from_function(
                market_tape_agent, name="market_tape_agent", description="tape"
            )
        ],
        system_prompt="t",
    )
    events: list[dict[str, Any]] = []
    token = set_progress(events.append)
    try:
        run_agent_with_progress(outer, [HumanMessage(content="btc")], {"recursion_limit": 8})
    finally:
        reset_progress(token)
    retry = [
        (e.get("type"), e.get("text") or "")
        for e in events
        if "Retrying get_ticker (2/3)…" in (e.get("text") or "")
    ]
    assert retry == [("status", "Retrying get_ticker (2/3)…")]
    assert state["n"] == 3


def test_non_transient_error_is_not_retried() -> None:
    fn, state = _flaky_ticker(10, error=ValueError("bad symbol"))
    with pytest.raises(ValueError, match="bad symbol"):
        _ticker_agent(fn).invoke({"messages": [HumanMessage(content="q")]}, {"recursion_limit": 8})
    assert state["n"] == 1
