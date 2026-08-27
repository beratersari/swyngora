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
        if request.url.path.endswith("/long-short-ratio"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "venues": [
                        {
                            "exchange": "binance",
                            "current": {"ratio": "1.70", "bias": "long", "longPct": "63"},
                        }
                    ],
                },
            )
        if request.url.path.endswith("/funding-arb/history"):
            return httpx.Response(
                200,
                json={"symbol": request.url.params.get("symbol"), "runs": []},
            )
        if request.url.path.endswith("/funding-arb/scan"):
            return httpx.Response(
                200,
                json={"hits": [], "skipped": 0, "note": "informational only"},
            )
        if request.url.path.endswith("/funding-arb"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "trade": {
                        "longExchange": "binance",
                        "shortExchange": "bybit",
                        "worthIt": False,
                    },
                },
            )
        if request.url.path.endswith("/funding-rate"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "venues": [
                        {
                            "exchange": "binance",
                            "current": {"rate": "0.0001", "ratePct": "0.01", "payer": "long"},
                        }
                    ],
                },
            )
        if request.url.path.endswith("/open-interest"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "unit": "BTC",
                    "current": {"contracts": "100", "value": "10000"},
                    "windows": [{"window": "24h", "change": "+10", "direction": "up"}],
                },
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
        if request.url.path.endswith("/orderbook/heatmap"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "windowSeconds": int(request.url.params.get("window") or 600),
                    "columns": [{"t": "2026-08-16T12:00:00Z", "mid": "100"}],
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
        if request.url.path.endswith("/volume-surge/scan"):
            return httpx.Response(
                200,
                json={"hits": [{"symbol": "HOTUSDT", "maxRatio": "5", "hottest": "5m"}]},
            )
        if request.url.path.endswith("/volume-surge"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "venues": [
                        {
                            "hottest": "5m",
                            "maxRatio": "5",
                            "windows": [{"window": "5m", "ratio": "5"}],
                        }
                    ],
                },
            )
        if request.url.path.endswith("/liquidity-sweeps"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "summary": "swept high 65000",
                    "venues": [
                        {"exchange": "binance", "sweeps": [{"side": "high", "level": "65000"}]}
                    ],
                },
            )
        if request.url.path.endswith("/absorption"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "summary": "bids absorbing market sells",
                    "combined": {"current": {"kind": "bid", "score": 72}},
                },
            )
        if request.url.path.endswith("/around/similar"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "summary": "2 similar past setups",
                    "matches": [{"similarity": "82", "afterReturnPct": "+3.5"}],
                },
            )
        if request.url.path.endswith("/around/precursors"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "summary": "volume was elevated before most up-moves",
                    "patterns": [{"metric": "volume_elevated", "common": True, "side": "up"}],
                },
            )
        if request.url.path.endswith("/around/moves"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "summary": "3 important moves",
                    "moves": [
                        {"direction": "up", "returnPct": "+4.2", "at": "2026-08-20T14:00:00Z"}
                    ],
                },
            )
        if request.url.path.endswith("/around/compare"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "fromAt": request.url.params.get("from"),
                    "toAt": request.url.params.get("to"),
                    "summary": "later move was larger",
                    "venues": [{"exchange": "binance", "phases": [{"phase": "during"}]}],
                },
            )
        if request.url.path.endswith("/around"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "at": request.url.params.get("at"),
                    "summary": "BTC moved +2% during the window.",
                    "combined": {
                        "phases": [
                            {"phase": "before"},
                            {"phase": "during", "price": {"changePct": "+2.0"}},
                            {"phase": "after"},
                        ]
                    },
                },
            )
        if request.url.path.endswith("/vwap"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "combined": {"vwap": "65000", "distancePct": "+1.2", "volume": "1000000"},
                },
            )
        if request.url.path.endswith("/volume-profile"):
            return httpx.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "exchange": request.url.params.get("exchange") or "all",
                    "window": request.url.params.get("window") or "24h",
                    "poc": {"price": "65200", "volume": "150000"},
                    "valueArea": {"low": "64800", "high": "66100"},
                },
            )
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

    assert "get_orderbook_heatmap" in by_name
    heat = json.loads(by_name["get_orderbook_heatmap"].invoke({"symbol": "BTCUSDT"}))
    assert heat["symbol"] == "BTCUSDT"
    assert heat["windowSeconds"] == 600

    assert "get_open_interest" in by_name
    oi = json.loads(by_name["get_open_interest"].invoke({"symbol": "BTCUSDT"}))
    assert oi["current"]["contracts"] == "100"

    assert "get_funding_rate" in by_name
    fr = json.loads(by_name["get_funding_rate"].invoke({"symbol": "BTCUSDT"}))
    assert fr["venues"][0]["current"]["payer"] == "long"

    assert "get_funding_arb" in by_name
    arb = json.loads(
        by_name["get_funding_arb"].invoke({"symbol": "BTCUSDT", "notional": 10000})
    )
    assert arb["trade"]["longExchange"] == "binance"
    assert "scan_funding_arb" in by_name
    ranked = json.loads(by_name["scan_funding_arb"].invoke({"notional": 10000}))
    assert ranked["hits"] == []
    assert "get_funding_arb_history" in by_name
    hist = json.loads(
        by_name["get_funding_arb_history"].invoke(
            {"symbol": "BTCUSDT", "start": "2026-08-01", "end": "2026-08-08"}
        )
    )
    assert hist["runs"] == []

    assert "get_long_short_ratio" in by_name
    lsr = json.loads(by_name["get_long_short_ratio"].invoke({"symbol": "BTCUSDT"}))
    assert lsr["venues"][0]["current"]["bias"] == "long"

    assert "get_volume_surge" in by_name
    surge = json.loads(by_name["get_volume_surge"].invoke({"symbol": "BTCUSDT"}))
    assert surge["venues"][0]["maxRatio"] == "5"
    assert "scan_volume_surges" in by_name
    hot = json.loads(by_name["scan_volume_surges"].invoke({"exchange": "binance"}))
    assert hot["hits"][0]["symbol"] == "HOTUSDT"

    assert "get_liquidity_sweeps" in by_name
    sweeps = json.loads(by_name["get_liquidity_sweeps"].invoke({"symbol": "BTCUSDT"}))
    assert sweeps["venues"][0]["sweeps"][0]["side"] == "high"

    assert "get_absorption" in by_name
    absorb = json.loads(by_name["get_absorption"].invoke({"symbol": "BTCUSDT"}))
    assert absorb["combined"]["current"]["kind"] == "bid"

    assert "get_around" in by_name
    around = json.loads(
        by_name["get_around"].invoke({"symbol": "BTCUSDT", "at": "2026-08-20T14:00:00Z"})
    )
    assert around["combined"]["phases"][1]["phase"] == "during"

    assert "compare_around" in by_name
    compared = json.loads(
        by_name["compare_around"].invoke(
            {
                "symbol": "BTCUSDT",
                "from_time": "2026-08-20T12:00:00Z",
                "to_time": "2026-08-20T16:00:00Z",
            }
        )
    )
    assert compared["summary"] == "later move was larger"

    assert "find_around_moves" in by_name
    found = json.loads(by_name["find_around_moves"].invoke({"symbol": "BTCUSDT"}))
    assert found["moves"][0]["direction"] == "up"

    assert "find_around_precursors" in by_name
    prec = json.loads(by_name["find_around_precursors"].invoke({"symbol": "BTCUSDT"}))
    assert prec["patterns"][0]["metric"] == "volume_elevated"

    assert "find_around_similar" in by_name
    sim = json.loads(by_name["find_around_similar"].invoke({"symbol": "BTCUSDT"}))
    assert sim["matches"][0]["afterReturnPct"] == "+3.5"

    assert "get_vwap" in by_name
    vwap = json.loads(by_name["get_vwap"].invoke({"symbol": "BTCUSDT", "window": "24h"}))
    assert vwap["combined"]["vwap"] == "65000"

    assert "get_volume_profile" in by_name
    vp = json.loads(
        by_name["get_volume_profile"].invoke(
            {"symbol": "BTCUSDT", "exchange": "all", "window": "4h"}
        )
    )
    assert vp["symbol"] == "BTCUSDT"
    assert vp["poc"]["price"] == "65200"

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
