import json

import httpx2

from swyngora_ai.config import Settings
from swyngora_ai.tools.market_http import bind_tool_scope, build_market_tools, reset_tool_scope


def _tools(monkeypatch):
    seen: list[tuple[str, str, dict]] = []
    real_client = httpx2.Client

    class _Transport(httpx2.BaseTransport):
        def handle_request(self, request: httpx2.Request) -> httpx2.Response:
            raw = request.content.decode() if request.content else "{}"
            seen.append((request.method, request.url.path, json.loads(raw or "{}")))
            return httpx2.Response(200, json={"equity": 10000, "shockedEquity": 9800, "replay": {"interval": "1d"}})

    def fake_client(*args, **kwargs):
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    return {tool.name: tool for tool in tools}, seen


def test_stress_portfolio_replays_a_period_when_read_only(monkeypatch):
    by_name, seen = _tools(monkeypatch)
    scope = bind_tool_scope(can_trade=False, can_manage_keys=False)
    try:
        out = by_name["stress_portfolio"].invoke(
            {
                "client_id": "c1",
                "period_from": "2024-08-05",
                "period_to": "2024-08-06",
                "portfolio_id": "book-1",
            }
        )
    finally:
        reset_tool_scope(scope)
    assert "9800" in out
    assert len(seen) == 1
    method, path, body = seen[0]
    assert method == "POST"
    assert path == "/api/v1/portfolio/stress"
    assert body["from"] == "2024-08-05"
    assert body["to"] == "2024-08-06"
    assert body["portfolioId"] == "book-1"
    assert "moves" not in body


def test_stress_portfolio_sends_typed_moves(monkeypatch):
    by_name, seen = _tools(monkeypatch)
    by_name["stress_portfolio"].invoke({"client_id": "c1", "moves": "BTC:-10,ETH:-15"})
    assert seen[0][2]["moves"] == [
        {"asset": "BTC", "movePct": -10},
        {"asset": "ETH", "movePct": -15},
    ]


def test_stress_portfolio_refuses_moves_and_a_period(monkeypatch):
    by_name, seen = _tools(monkeypatch)
    out = by_name["stress_portfolio"].invoke(
        {"client_id": "c1", "moves": "BTC:-10", "period_from": "2024-08-05", "period_to": "2024-08-06"}
    )
    assert out.startswith("ERROR 400")
    assert seen == []
