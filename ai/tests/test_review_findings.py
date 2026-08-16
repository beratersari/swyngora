"""Verification tests for the 2026-08-16 review findings (AI surface)."""

from __future__ import annotations

import json

import httpx

from swyngora_ai.config import Settings
from swyngora_ai.graph.facts import extract_market_facts
from swyngora_ai.graph.orchestrator import Orchestrator, SessionMemory
from swyngora_ai.grounding import apply_grounding
from swyngora_ai.tools.market_http import (
    begin_tool_json_turn,
    bind_client_id,
    bind_tool_scope,
    build_market_tools,
    collected_tool_json,
    record_tool_json,
    reset_bound_client_id,
    reset_tool_json_turn,
    reset_tool_scope,
)


class _RecordingTransport(httpx.BaseTransport):
    def __init__(self) -> None:
        self.requests: list[httpx.Request] = []

    def handle_request(self, request: httpx.Request) -> httpx.Response:
        self.requests.append(request)
        if request.url.path == "/api/v1/account/close":
            return httpx.Response(200, json={"status": "closed", "clientId": "victim"})
        return httpx.Response(200, json={"ok": True, "path": request.url.path})


def _install_tools(monkeypatch, transport: _RecordingTransport):
    real_client = httpx.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = transport
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=transport)

    monkeypatch.setattr(httpx, "Client", fake_client)
    return build_market_tools(Settings(api_base_url="http://backend.test", api_auth_token="master"))


def test_finding4_cancel_import_dotdot_does_not_hit_account_close(monkeypatch):
    """Finding 4: LLM-controlled import_id can normalize to POST /api/v1/account/close."""
    transport = _RecordingTransport()
    tools = _install_tools(monkeypatch, transport)
    by_name = {t.name: t for t in tools}
    bind_tok = bind_client_id("victim")
    scope = bind_tool_scope(can_trade=True, can_manage_keys=False)
    try:
        out = by_name["cancel_import"].invoke(
            {"client_id": "ignored", "import_id": "../account/close?"}
        )
    finally:
        reset_tool_scope(scope)
        reset_bound_client_id(bind_tok)

    paths = [r.url.path for r in transport.requests]
    assert "ERROR 400" in out
    assert "/api/v1/account/close" not in paths
    assert all("/account/close" not in p for p in paths), paths


def test_finding4_cancel_export_and_pause_recurring_also_normalized(monkeypatch):
    transport = _RecordingTransport()
    tools = _install_tools(monkeypatch, transport)
    by_name = {t.name: t for t in tools}
    bind_tok = bind_client_id("victim")
    scope = bind_tool_scope(can_trade=True, can_manage_keys=False)
    try:
        by_name["cancel_export"].invoke(
            {"client_id": "ignored", "export_id": "x/../../account/close?"}
        )
        by_name["pause_recurring_buy"].invoke(
            {"client_id": "ignored", "plan_id": "../../account/close?"}
        )
    finally:
        reset_tool_scope(scope)
        reset_bound_client_id(bind_tok)

    paths = [r.url.path for r in transport.requests]
    assert "/api/v1/account/close" not in paths
    assert all("/account/close" not in p for p in paths), paths


def test_finding5_specialist_prose_is_not_grounded():
    """Finding 5: orchestrator facts come from specialist prose, so apply_grounding is skipped."""
    ticker = json.dumps({"symbol": "BTCUSDT", "exchange": "binance", "lastPrice": "100.25"})
    tok = begin_tool_json_turn()
    try:
        record_tool_json(ticker)
        facts = extract_market_facts(*collected_tool_json())
    finally:
        reset_tool_json_turn(tok)
    assert facts.last_price == "100.25"
    out = apply_grounding("BTC last is 99999.12 on binance.", facts, [ticker])
    assert "Unverified" in out


def test_finding5_orchestrator_skips_grounding_without_json_tool_messages():
    """Chat path only inspects orchestrator-level ToolMessages (specialist return text)."""
    from typing import Any

    from langchain_core.language_models.chat_models import BaseChatModel
    from langchain_core.messages import AIMessage
    from langchain_core.outputs import ChatGeneration, ChatResult

    class ScriptedModel(BaseChatModel):
        responses: list[AIMessage]
        _i: int = 0

        @property
        def _llm_type(self) -> str:
            return "scripted"

        def _generate(self, messages, stop=None, run_manager=None, **kwargs: Any) -> ChatResult:
            if self._i >= len(self.responses):
                msg = self.responses[-1] if self.responses else AIMessage(content="done")
            else:
                msg = self.responses[self._i]
                self._i += 1
            return ChatResult(generations=[ChatGeneration(message=msg)])

        def bind_tools(self, tools, **kwargs):
            return self

    model = ScriptedModel(
        responses=[
            AIMessage(content="BTC last is 99999.12. Not financial advice."),
        ]
    )
    orch = Orchestrator(
        settings=Settings(_env_file=None),
        memory=SessionMemory(),
        model=model,
    )
    out = orch.chat("What is BTC price?", session_id="ground-1")
    assert out.reply
    # No tool JSON this turn — nothing to ground against.
    assert "Unverified" not in (out.reply or "")
