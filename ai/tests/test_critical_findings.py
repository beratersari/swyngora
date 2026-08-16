"""Verification tests for the 2026-08-16 critical-review AI findings."""

from __future__ import annotations

import httpx

from swyngora_ai.config import Settings
from swyngora_ai.agents.specialists import build_specialist_tools
from swyngora_ai.serve import is_service_authorized
from swyngora_ai.tools.market_http import (
    bind_tool_scope,
    build_market_tools,
    reset_tool_scope,
)

from test_specialists_and_orchestrator import ScriptedModel
from langchain_core.messages import AIMessage


def test_finding11_empty_service_token_is_explicit_default():
    """Documented local-dev default, not an accidental missing check."""
    assert Settings(_env_file=None).service_token == ""
    assert is_service_authorized({}, "") is True
    assert is_service_authorized({}, "") is True


def test_finding11_specialists_always_include_paper_and_account():
    model = ScriptedModel(responses=[AIMessage(content="ok")])
    names = {t.name for t in build_specialist_tools(model, Settings(_env_file=None))}
    assert "paper_desk_agent" in names
    assert "account_agent" in names


def test_finding11_mutating_tool_blocked_when_can_trade_false(monkeypatch):
    posted: list[str] = []

    class _Transport(httpx.BaseTransport):
        def handle_request(self, request: httpx.Request) -> httpx.Response:
            posted.append(f"{request.method} {request.url.path}")
            return httpx.Response(200, json={"ok": True})

    real = httpx.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = _Transport()
        kwargs.pop("timeout", None)
        return real(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx, "Client", fake_client)
    tools = {t.name: t for t in build_market_tools(Settings(api_base_url="http://test"))}
    toks = bind_tool_scope(can_trade=False, can_manage_keys=False)
    try:
        out = tools["place_portfolio_order"].invoke(
            {"client_id": "c1", "symbol": "BTCUSDT", "side": "buy", "quantity": 1}
        )
    finally:
        reset_tool_scope(toks)
    assert "403" in out
    assert posted == []


def test_finding11_mutating_tool_allowed_when_can_trade_defaults_true(monkeypatch):
    posted: list[str] = []

    class _Transport(httpx.BaseTransport):
        def handle_request(self, request: httpx.Request) -> httpx.Response:
            posted.append(f"{request.method} {request.url.path}")
            return httpx.Response(200, json={"ok": True})

    real = httpx.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = _Transport()
        kwargs.pop("timeout", None)
        return real(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx, "Client", fake_client)
    tools = {t.name: t for t in build_market_tools(Settings(api_base_url="http://test"))}
    # No bind_tool_scope — ContextVar default is True (CLI / missing flags).
    out = tools["place_portfolio_order"].invoke(
        {"client_id": "c1", "symbol": "BTCUSDT", "side": "buy", "quantity": 1}
    )
    assert "403" not in out
    assert any("portfolio" in p for p in posted)
