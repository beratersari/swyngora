"""Architecture-review contract checks (AI HTTP tools vs Go HTTP/MCP)."""

from __future__ import annotations

import inspect
import json

import httpx

from swyngora_ai.config import Settings
from swyngora_ai.tools.market_http import (
    BasketCreateInput,
    MarginOrderInput,
    PortfolioCancelAllInput,
    PortfolioCashMoveInput,
    PortfolioOrderInput,
    PortfolioPendingOrderInput,
    RecurringBuyCreateInput,
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
    for cls in (
        PortfolioOrderInput,
        PortfolioCashMoveInput,
        PortfolioPendingOrderInput,
        PortfolioCancelAllInput,
        RecurringBuyCreateInput,
        BasketCreateInput,
        MarginOrderInput,
    ):
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


# Book-scoped tools whose args_schema advertises portfolio_id. HTTP/MCP require
# that field once a tenant has more than one paper book. Tenant-level lists
# (keys, scanner, price-diff, list_portfolios) use ClientIdInput instead.
_SCHEMA_HAS_PORTFOLIO_ID = (
    "list_portfolio_trades",
    "list_portfolio_orders",
    "list_portfolio_lots",
    "list_recurring_buys",
    "list_portfolio_baskets",
    "list_margin_positions",
    "list_margin_orders",
    "list_margin_trades",
    "place_portfolio_oco_order",
    "place_portfolio_bracket_order",
    "cancel_portfolio_order",
    "cancel_margin_order",
    "place_margin_order",
    "create_recurring_buy",
    "create_portfolio_basket",
    "cancel_all_portfolio_orders",
)

# HTTP handlers require portfolioId with 2+ books, but the Python schema has no field.
_HTTP_REQUIRES_PORTFOLIO_ID_SCHEMA_OMITS = (
    "place_margin_order",
    "create_recurring_buy",
    "create_portfolio_basket",
    "cancel_all_portfolio_orders",
)


def test_arch_review_schema_portfolio_id_matches_function_signature():
    """LangChain dumps the full args_schema into func(**kwargs). Extra keys TypeError."""
    tools = {t.name: t for t in build_market_tools(Settings(api_base_url="http://test"))}
    mismatches: list[str] = []
    for name in _SCHEMA_HAS_PORTFOLIO_ID:
        tool = tools[name]
        schema = tool.args_schema
        assert schema is not None, name
        fields = getattr(schema, "model_fields", {})
        assert "portfolio_id" in fields, f"{name} schema lost portfolio_id"
        func = getattr(tool, "func", None)
        assert callable(func), name
        params = inspect.signature(func).parameters
        if "portfolio_id" not in params and "kwargs" not in params:
            mismatches.append(name)
    assert not mismatches, (
        "args_schema has portfolio_id but the HTTP function does not accept it "
        f"(LangChain invoke TypeError / dropped book): {mismatches}"
    )


def test_arch_review_list_trades_default_invoke_does_not_typeerror(monkeypatch):
    """Pydantic model_dump includes portfolio_id='' even when the model omits it."""
    transport = _RecordingTransport()
    tools = _install(monkeypatch, transport)
    tok = bind_client_id("arch-ai")
    try:
        out = tools["list_portfolio_trades"].invoke({"client_id": "arch-ai"})
    finally:
        reset_bound_client_id(tok)
    assert "ERROR" not in out, out
    assert transport.requests, f"tool never reached HTTP: {out!r}"
    assert transport.requests[-1].url.path == "/api/v1/portfolio/trades"


def test_arch_review_list_trades_sends_portfolio_id(monkeypatch):
    transport = _RecordingTransport()
    tools = _install(monkeypatch, transport)
    tok = bind_client_id("arch-ai")
    try:
        out = tools["list_portfolio_trades"].invoke(
            {"client_id": "arch-ai", "portfolio_id": "book-1"}
        )
    finally:
        reset_bound_client_id(tok)

    assert "ERROR" not in out, out
    assert "unexpected keyword" not in out.lower(), out
    last = transport.requests[-1]
    assert last.url.path == "/api/v1/portfolio/trades"
    sent = last.url.params.get("portfolioId")
    assert sent == "book-1", f"list_portfolio_trades dropped portfolioId (url={last.url})"


def test_arch_review_oco_sends_portfolio_id(monkeypatch):
    transport = _RecordingTransport()
    tools = _install(monkeypatch, transport)
    tok = bind_client_id("arch-ai")
    try:
        out = tools["place_portfolio_oco_order"].invoke(
            {
                "client_id": "arch-ai",
                "symbol": "BTCUSDT",
                "quantity": 1,
                "take_profit_price": 110,
                "stop_loss_price": 90,
                "portfolio_id": "book-1",
            }
        )
    finally:
        reset_bound_client_id(tok)

    assert "ERROR" not in out, out
    assert "unexpected keyword" not in out.lower(), out
    last = transport.requests[-1]
    assert last.url.path == "/api/v1/portfolio/orders"
    body = json.loads(last.content.decode())
    assert body.get("portfolioId") == "book-1", body


def test_arch_review_place_margin_http_sends_portfolio_id(monkeypatch):
    transport = _RecordingTransport()
    tools = _install(monkeypatch, transport)
    tok = bind_client_id("arch-ai")
    try:
        out = tools["place_margin_order"].invoke(
            {
                "client_id": "arch-ai",
                "symbol": "BTCUSDT",
                "side": "long",
                "quantity": 1,
                "leverage": 2,
                "portfolio_id": "book-1",
            }
        )
    finally:
        reset_bound_client_id(tok)
    assert "ERROR" not in out, out
    last = transport.requests[-1]
    assert last.url.path == "/api/v1/portfolio/margin/orders"
    body = json.loads(last.content.decode())
    assert body.get("portfolioId") == "book-1", body


def test_arch_review_margin_and_recurring_schemas_include_portfolio_id():
    tools = {t.name: t for t in build_market_tools(Settings(api_base_url="http://test"))}
    missing: list[str] = []
    for name in _HTTP_REQUIRES_PORTFOLIO_ID_SCHEMA_OMITS:
        fields = getattr(tools[name].args_schema, "model_fields", {})
        if "portfolio_id" not in fields:
            missing.append(name)
    assert not missing, (
        "HTTP 400s these tools without portfolioId once a tenant has 2+ books, "
        f"but the AI schema omits the field: {missing}"
    )
