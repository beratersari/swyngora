import json

import httpx
import pytest

from swyngora_ai.config import Settings
from swyngora_ai.tools.market_http import build_market_tools


class _Transport(httpx.BaseTransport):
    def handle_request(self, request: httpx.Request) -> httpx.Response:
        if request.url.path == "/health":
            return httpx.Response(200, json={"status": "ok"})
        if request.url.path.endswith("/ticker/24h"):
            return httpx.Response(
                200,
                json={"symbol": request.url.params.get("symbol"), "lastPrice": "100"},
            )
        if request.url.path.endswith("/orderbook"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "groupSize": request.url.params.get("group") or "0.1",
                    "bids": [{"price": "100", "quantity": "1", "isWall": False}],
                    "asks": [{"price": "100.1", "quantity": "1", "isWall": True}],
                },
            )
        if request.url.path.endswith("/exchanges"):
            return httpx.Response(200, json={"exchanges": ["binance"], "default": "binance"})
        return httpx.Response(404, json={"error": "not found"})


def test_market_tools_hit_api(monkeypatch):
    # Patch httpx.Client to use mock transport
    real_client = httpx.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = _Transport()
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {t.name: t for t in tools}
    assert "get_ticker" in by_name
    out = by_name["get_ticker"].invoke({"symbol": "BTCUSDT", "exchange": "binance"})
    data = json.loads(out)
    assert data["symbol"] == "BTCUSDT"
    assert data["lastPrice"] == "100"

    assert "get_spot_orderbook" in by_name
    book = json.loads(
        by_name["get_spot_orderbook"].invoke(
            {"symbol": "BTCUSDT", "exchange": "binance", "group": "0.1"}
        )
    )
    assert book["symbol"] == "BTCUSDT"
    assert book["asks"][0]["isWall"] is True

    assert "analyze_spot_orderbook" in by_name
    analysis = json.loads(
        by_name["analyze_spot_orderbook"].invoke(
            {"symbol": "BTCUSDT", "exchange": "binance", "range_pct": 2}
        )
    )
    assert analysis["symbol"] == "BTCUSDT"
    assert "bids" not in analysis

    health = by_name["health"].invoke({})
    assert "ok" in health
