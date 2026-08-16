import json

import httpx
from langchain_core.messages import AIMessage, HumanMessage

from swyngora_ai.config import Settings
from swyngora_ai.graph.orchestrator import SessionMemory, memory_key
from swyngora_ai.tools.market_http import (
    bind_tool_scope,
    build_market_tools,
    reset_tool_scope,
)


class _Transport(httpx.BaseTransport):
    def handle_request(self, request: httpx.Request) -> httpx.Response:
        if request.url.path == "/health":
            return httpx.Response(200, json={"status": "ok"})
        if request.url.path.endswith("/ticker/24h"):
            return httpx.Response(
                200,
                json={"symbol": request.url.params.get("symbol"), "lastPrice": "100"},
            )
        if request.url.path.endswith("/liquidations"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "windows": [
                        {"window": "24h", "longNotional": "100", "shortNotional": "50", "count": 3}
                    ],
                },
            )
        if request.url.path.endswith("/orderbook/liquidity"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "venueCount": 3,
                    "market": {"score": 72, "grade": "high", "weakerSide": "balanced"},
                },
            )
        if request.url.path.endswith("/orderbook/impact"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "side": request.url.params.get("side") or "buy",
                    "averagePrice": "100.25",
                    "slippagePct": 0.25,
                    "exhausted": False,
                    "filledQuantity": request.url.params.get("quantity") or "1",
                },
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

    assert "get_liquidations" in by_name
    liqs = json.loads(by_name["get_liquidations"].invoke({"symbol": "BTCUSDT"}))
    assert liqs["windows"][0]["window"] == "24h"

    assert "get_market_liquidity" in by_name
    liq = json.loads(by_name["get_market_liquidity"].invoke({"symbol": "BTCUSDT"}))
    assert liq["market"]["score"] == 72

    assert "estimate_market_impact" in by_name
    impact = json.loads(
        by_name["estimate_market_impact"].invoke(
            {"symbol": "BTCUSDT", "quantity": 5, "side": "buy"}
        )
    )
    assert impact["symbol"] == "BTCUSDT"
    assert impact["averagePrice"] == "100.25"

    health = by_name["health"].invoke({})
    assert "ok" in health


def test_market_http_reuses_one_client(monkeypatch):
    created: list[httpx.Client] = []
    real_client = httpx.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = _Transport()
        kwargs.pop("timeout", None)
        client = real_client(*args, timeout=5.0, transport=_Transport())
        created.append(client)
        return client

    monkeypatch.setattr(httpx, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {t.name: t for t in tools}
    by_name["get_ticker"].invoke({"symbol": "BTCUSDT", "exchange": "binance"})
    by_name["health"].invoke({})
    assert len(created) == 1


def test_read_only_scope_blocks_mutations(monkeypatch):
    posted: list[str] = []

    class _MutTransport(httpx.BaseTransport):
        def handle_request(self, request: httpx.Request) -> httpx.Response:
            posted.append(f"{request.method} {request.url.path}")
            return httpx.Response(200, json={"ok": True})

    real_client = httpx.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = _MutTransport()
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_MutTransport())

    monkeypatch.setattr(httpx, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {t.name: t for t in tools}

    toks = bind_tool_scope(can_trade=False, can_manage_keys=False)
    try:
        out = by_name["place_portfolio_order"].invoke(
            {
                "client_id": "c1",
                "symbol": "BTCUSDT",
                "side": "buy",
                "quantity": 1,
            }
        )
        assert "403" in out
        assert "read-only" in out
        assert posted == []

        keys = by_name["create_api_key"].invoke(
            {"client_id": "c1", "name": "x", "permission": "trade"}
        )
        assert "403" in keys
        assert "not available" in keys
    finally:
        reset_tool_scope(toks)


def test_memory_key_isolates_tenants():
    mem = SessionMemory()
    a = memory_key("alice", "shared")
    b = memory_key("bob", "shared")
    assert a != b
    mem.append(a, [HumanMessage(content="alice secret"), AIMessage(content="ok")])
    assert mem.get(b) == []
    assert len(mem.get(a)) == 2
