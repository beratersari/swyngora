import json

import httpx2

from swyngora_ai.config import Settings
from swyngora_ai.tools.market_http import bind_tool_scope, build_market_tools, reset_tool_scope


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
    assert path != "/api/v1/portfolio/orders"
    assert "cashDebit" in out
