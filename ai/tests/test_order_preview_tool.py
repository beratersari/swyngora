import json

import httpx2

from swyngora_ai.config import Settings
from swyngora_ai.tools.market_http import bind_tool_scope, build_market_tools, reset_tool_scope


def test_compare_order_execution_posts_three_prices(monkeypatch):
    seen: list[tuple[str, str, dict]] = []
    real_client = httpx2.Client

    class _Transport(httpx2.BaseTransport):
        def handle_request(self, request: httpx2.Request) -> httpx2.Response:
            raw = request.content.decode() if request.content else "{}"
            seen.append((request.method, request.url.path, json.loads(raw or "{}")))
            return httpx2.Response(200, json={"rows": [{"style": "market", "status": "filled"}]})

    def fake_client(*args, **kwargs):
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {tool.name: tool for tool in tools}
    scope = bind_tool_scope(can_trade=False, can_manage_keys=False)
    try:
        out = by_name["compare_order_execution"].invoke(
            {
                "client_id": "c1",
                "symbol": "BTCUSDT",
                "side": "buy",
                "quantity": 2,
                "exchange": "binance",
                "spread_minutes": 10,
                "slice_count": 2,
            }
        )
    finally:
        reset_tool_scope(scope)
    assert len(seen) == 1
    method, path, body = seen[0]
    assert method == "POST"
    assert path == "/api/v1/portfolio/orders/execution-compare"
    assert body["symbol"] == "BTCUSDT"
    assert body["side"] == "buy"
    assert body["quantity"] == 2
    assert body["exchange"] == "binance"
    assert body["spreadMinutes"] == 10
    assert body["sliceCount"] == 2
    assert "rows" in out


def test_place_portfolio_order_sends_execution_quote_id(monkeypatch):
    seen: list[tuple[str, str, dict]] = []
    real_client = httpx2.Client

    class _Transport(httpx2.BaseTransport):
        def handle_request(self, request: httpx2.Request) -> httpx2.Response:
            raw = request.content.decode() if request.content else "{}"
            seen.append((request.method, request.url.path, json.loads(raw or "{}")))
            return httpx2.Response(200, json={"trade": {"id": "t1"}})

    def fake_client(*args, **kwargs):
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {tool.name: tool for tool in tools}
    scope = bind_tool_scope(can_trade=True, can_manage_keys=False)
    try:
        by_name["place_portfolio_order"].invoke(
            {
                "client_id": "c1",
                "symbol": "BTCUSDT",
                "side": "buy",
                "quantity": 1,
                "exchange": "binance",
                "execution_quote_id": "quote-1",
            }
        )
    finally:
        reset_tool_scope(scope)
    assert seen[0][1] == "/api/v1/portfolio/orders"
    assert seen[0][2]["executionQuoteId"] == "quote-1"


def test_set_emergency_stop_puts_the_switch(monkeypatch):
    seen: list[tuple[str, str, dict]] = []
    real_client = httpx2.Client

    class _Transport(httpx2.BaseTransport):
        def handle_request(self, request: httpx2.Request) -> httpx2.Response:
            raw = request.content.decode() if request.content else "{}"
            seen.append((request.method, request.url.path, json.loads(raw or "{}")))
            return httpx2.Response(200, json={"enabled": True, "lossPct": 5, "hit": False})

    def fake_client(*args, **kwargs):
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {tool.name: tool for tool in tools}
    scope = bind_tool_scope(can_trade=True, can_manage_keys=False)
    try:
        by_name["set_emergency_stop"].invoke(
            {"client_id": "c1", "enabled": True, "loss_pct": 5}
        )
    finally:
        reset_tool_scope(scope)
    assert seen[0][0] == "PUT"
    assert seen[0][1].startswith("/api/v1/portfolio/emergency-stop")
    assert seen[0][2] == {"enabled": True, "lossPct": 5}


def test_preview_portfolio_order_posts_preview_only(monkeypatch):
    seen: list[tuple[str, str, dict]] = []
    real_client = httpx2.Client

    class _Transport(httpx2.BaseTransport):
        def handle_request(self, request: httpx2.Request) -> httpx2.Response:
            raw = request.content.decode() if request.content else "{}"
            seen.append((request.method, request.url.path, json.loads(raw or "{}")))
            return httpx2.Response(
                200,
                json={"fillsNow": True, "cashDebit": 100, "fee": 0, "risk": {"blocked": False}},
            )

    def fake_client(*args, **kwargs):
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {tool.name: tool for tool in tools}
    # A read-only session may price an order. It still cannot place one.
    scope = bind_tool_scope(can_trade=False, can_manage_keys=False)
    try:
        out = by_name["preview_portfolio_order"].invoke(
            {
                "client_id": "c1",
                "symbol": "BTCUSDT",
                "side": "buy",
                "quantity": 1,
                "order_type": "market",
                "product": "spot",
            }
        )
    finally:
        reset_tool_scope(scope)
    assert len(seen) == 1
    method, path, body = seen[0]
    assert method == "POST"
    assert path == "/api/v1/portfolio/orders/preview"
    assert body["symbol"] == "BTCUSDT"
    assert body["side"] == "buy"
    assert body["quantity"] == 1
    assert body["type"] == "market"
    assert "riskPercent" not in body
    assert path != "/api/v1/portfolio/orders"
    assert "cashDebit" in out


def test_preview_portfolio_order_sends_risk_percent_without_quantity(monkeypatch):
    seen: list[dict] = []
    real_client = httpx2.Client

    class _Transport(httpx2.BaseTransport):
        def handle_request(self, request: httpx2.Request) -> httpx2.Response:
            raw = request.content.decode() if request.content else "{}"
            seen.append(json.loads(raw or "{}"))
            return httpx2.Response(200, json={"quantity": 10, "sizing": {"riskBudget": 100}})

    def fake_client(*args, **kwargs):
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {tool.name: tool for tool in tools}
    scope = bind_tool_scope(can_trade=False, can_manage_keys=False)
    try:
        by_name["preview_portfolio_order"].invoke(
            {
                "client_id": "c1",
                "symbol": "BTCUSDT",
                "side": "buy",
                "risk_percent": 1,
                "stop_loss_price": 90,
                "product": "spot",
            }
        )
    finally:
        reset_tool_scope(scope)
    assert seen[0]["riskPercent"] == 1
    assert seen[0]["stopLossPrice"] == 90
    assert "quantity" not in seen[0]


def test_preview_portfolio_order_sends_best_cost_route(monkeypatch):
    seen: list[dict] = []
    real_client = httpx2.Client

    class _Transport(httpx2.BaseTransport):
        def handle_request(self, request: httpx2.Request) -> httpx2.Response:
            raw = request.content.decode() if request.content else "{}"
            seen.append(json.loads(raw or "{}"))
            return httpx2.Response(200, json={"routed": True, "routeLegs": []})

    def fake_client(*args, **kwargs):
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {tool.name: tool for tool in tools}
    scope = bind_tool_scope(can_trade=False, can_manage_keys=False)
    try:
        by_name["preview_portfolio_order"].invoke(
            {
                "client_id": "c1",
                "symbol": "BTCUSDT",
                "side": "buy",
                "quantity": 2,
                "exchange": "all",
                "compare_exchange": "binance",
                "max_slippage_pct": 0.5,
            }
        )
    finally:
        reset_tool_scope(scope)
    assert seen[0]["exchange"] == "all"
    assert seen[0]["compareExchange"] == "binance"
    assert seen[0]["maxSlippagePct"] == 0.5


def test_preview_portfolio_order_sends_spread(monkeypatch):
    seen: list[dict] = []
    real_client = httpx2.Client

    class _Transport(httpx2.BaseTransport):
        def handle_request(self, request: httpx2.Request) -> httpx2.Response:
            raw = request.content.decode() if request.content else "{}"
            seen.append(json.loads(raw or "{}"))
            return httpx2.Response(200, json={"spread": True, "sliceCount": 10})

    def fake_client(*args, **kwargs):
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {tool.name: tool for tool in tools}
    scope = bind_tool_scope(can_trade=False, can_manage_keys=False)
    try:
        by_name["preview_portfolio_order"].invoke(
            {
                "client_id": "c1",
                "symbol": "BTCUSDT",
                "side": "buy",
                "quantity": 1,
                "exchange": "all",
                "spread": True,
                "spread_minutes": 30,
                "slice_count": 15,
            }
        )
    finally:
        reset_tool_scope(scope)
    assert seen[0]["spread"] is True
    assert seen[0]["spreadMinutes"] == 30
    assert seen[0]["sliceCount"] == 15
    assert seen[0]["exchange"] == "all"
    assert seen[0]["quantity"] == 1
