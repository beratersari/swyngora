"""Verification tests for the 2026-08-16 critical-review AI findings."""

from __future__ import annotations

import httpx

from swyngora_ai.config import Settings
from swyngora_ai.tools.market_http import build_market_tools


def test_finding11_unbound_mutating_tool_is_denied(monkeypatch):
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
    # No bind — unbound workers must fail closed (not inherit another chat).
    out = tools["place_portfolio_order"].invoke(
        {"client_id": "c1", "symbol": "BTCUSDT", "side": "buy", "quantity": 1}
    )
    assert "403" in out
    assert "read-only" in out
    assert not posted
