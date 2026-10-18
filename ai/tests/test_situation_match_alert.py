import json

import httpx2
from swyngora_ai.config import Settings
from swyngora_ai.tools.market_http import bind_tool_scope, build_market_tools, reset_tool_scope


def test_create_situation_match_alert_posts_selected_clauses(monkeypatch):
    seen: list[httpx2.Request] = []

    class _Transport(httpx2.BaseTransport):
        def handle_request(self, request: httpx2.Request) -> httpx2.Response:
            seen.append(request)
            return httpx2.Response(201, json={"id": "a1", "condition": "matches"})

    real_client = httpx2.Client

    def fake_client(*args, **kwargs):
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
    tools = {t.name: t for t in build_market_tools(Settings(api_base_url="http://test"))}
    scope = bind_tool_scope(can_trade=True, can_manage_keys=False)
    try:
        out = json.loads(
            tools["create_situation_match_alert"].invoke(
                {
                    "client_id": "c1",
                    "symbol": "BTCUSDT",
                    "price_cvd": "price_up_cvd_down",
                    "funding": "long",
                    "prints": "",
                }
            )
        )
    finally:
        reset_tool_scope(scope)
    assert out["condition"] == "matches"
    body = json.loads(seen[-1].content.decode())
    assert seen[-1].url.path == "/api/v1/alerts"
    assert body["kind"] == "situation"
    assert body["condition"] == "matches"
    assert body["situation"] == {"priceCvd": "price_up_cvd_down", "funding": "long"}
    assert "prints" not in body["situation"]
