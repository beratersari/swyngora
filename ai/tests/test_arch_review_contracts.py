"""Architecture-review contract checks (AI HTTP tools vs Go HTTP/MCP)."""

from __future__ import annotations

import inspect
import json

import httpx

from swyngora_ai.config import Settings
from swyngora_ai.tools.market_http import (
    PortfolioCashMoveInput,
    PortfolioOrderInput,
    PortfolioPendingOrderInput,
    bind_client_id,
    build_market_tools,
    reset_bound_client_id,
)


class _RecordingTransport(httpx.BaseTransport):
    def __init__(self) -> None:
        self.requests: list[httpx.Request] = []

    def handle_request(self, request: httpx.Request) -> httpx.Response:
        self.requests.append(request)
        return httpx.Response(200, json={"ok": True})


def _install(monkeypatch, transport: _RecordingTransport):
    real = httpx.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = transport
        kwargs.pop("timeout", None)
        return real(*args, timeout=5.0, transport=transport)

    monkeypatch.setattr(httpx, "Client", fake_client)
    return {t.name: t for t in build_market_tools(Settings(api_base_url="http://backend.test"))}


def test_arch_review_mutating_paper_tools_have_portfolio_id():
    for cls in (PortfolioOrderInput, PortfolioCashMoveInput, PortfolioPendingOrderInput):
        assert "portfolio_id" in cls.model_fields, cls.__name__


def test_arch_review_place_order_http_sends_portfolio_id(monkeypatch):
    transport = _RecordingTransport()
    tools = _install(monkeypatch, transport)
    tok = bind_client_id("arch-ai")
    try:
        out = tools["place_portfolio_order"].invoke(
            {
                "client_id": "arch-ai",
                "symbol": "BTCUSDT",
                "side": "buy",
                "quantity": 1,
                "portfolio_id": "book-1",
            }
        )
    finally:
        reset_bound_client_id(tok)

    assert "ERROR" not in out
    last = transport.requests[-1]
    assert last.url.path == "/api/v1/portfolio/orders"
    body = json.loads(last.content.decode())
    assert body.get("portfolioId") == "book-1"


def test_arch_review_deposit_http_sends_portfolio_id(monkeypatch):
    transport = _RecordingTransport()
    tools = _install(monkeypatch, transport)
    tok = bind_client_id("arch-ai")
    try:
        tools["deposit_portfolio_cash"].invoke(
            {"client_id": "arch-ai", "amount": 100, "portfolio_id": "book-1"}
        )
    finally:
        reset_bound_client_id(tok)
    last = transport.requests[-1]
    body = json.loads(last.content.decode())
    assert last.url.path == "/api/v1/portfolio/deposits"
    assert body.get("portfolioId") == "book-1"


def test_arch_review_get_asset_profile_bound():
    names = {t.name for t in build_market_tools(Settings(api_base_url="http://test"))}
    assert "get_asset_profile" in names


def test_arch_review_pending_order_can_place_trailing_stop():
    fields = PortfolioPendingOrderInput.model_fields
    desc = str(fields["order_type"].description)
    assert "trailing_stop" in desc
    assert "trail_type" in fields
    assert "trail_value" in fields


def test_arch_review_place_order_signature_has_portfolio_id():
    from swyngora_ai.tools import market_http as m

    src = inspect.getsource(m.build_market_tools)
    start = src.index("def place_portfolio_order(")
    end = src.index("def place_portfolio_pending_order(")
    block = src[start:end]
    assert "portfolio_id" in block
