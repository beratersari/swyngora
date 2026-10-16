import json

import httpx2
from langchain_core.messages import AIMessage, HumanMessage

from swyngora_ai.config import Settings
from swyngora_ai.graph.orchestrator import SessionMemory, memory_key
from swyngora_ai.tools.market_http import (
    bind_tool_scope,
    build_market_tools,
    reset_tool_scope,
)


class _Transport(httpx2.BaseTransport):
    def handle_request(self, request: httpx2.Request) -> httpx2.Response:
        if request.url.path == "/health":
            return httpx2.Response(200, json={"status": "ok"})
        if request.url.path.endswith("/ticker/24h"):
            return httpx2.Response(
                200,
                json={"symbol": request.url.params.get("symbol"), "lastPrice": "100"},
            )
        if request.url.path.endswith("/long-short-ratio"):
            return httpx2.Response(
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
            return httpx2.Response(
                200,
                json={"symbol": request.url.params.get("symbol"), "runs": []},
            )
        if request.url.path.endswith("/funding-arb/scan"):
            return httpx2.Response(
                200,
                json={"hits": [], "skipped": 0, "note": "informational only"},
            )
        if "/funding-arb/watches" in request.url.path:
            if request.method == "POST":
                return httpx2.Response(201, json={"id": "w1", "symbol": "BTCUSDT", "minProfit": 10})
            if request.method == "DELETE":
                return httpx2.Response(200, json={"deleted": True})
            return httpx2.Response(200, json={"id": "w1", "watches": []})
        if request.url.path.endswith("/funding-arb/signals"):
            return httpx2.Response(200, json={"signals": []})
        if request.url.path.endswith("/funding-arb"):
            return httpx2.Response(
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
            return httpx2.Response(
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
        if request.url.path.endswith("/options"):
            return httpx2.Response(
                200,
                json={
                    "asset": "BTC",
                    "exchange": "deribit",
                    "openInterest": "350",
                    "putCallOi": "0.4",
                    "dvol": "49.5",
                },
            )
        if request.url.path.endswith("/open-interest"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "unit": "BTC",
                    "current": {"contracts": "100", "value": "10000"},
                    "windows": [{"window": "24h", "change": "+10", "direction": "up"}],
                },
            )
        if request.url.path.endswith("/liquidation-cascade/scan"):
            return httpx2.Response(
                200,
                json={
                    "market": {"symbol": "all", "summary": "quiet"},
                    "hits": [{"symbol": "SOLUSDT", "grade": "cascade", "side": "long"}],
                },
            )
        if request.url.path.endswith("/liquidation-cascade"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol") or "all",
                    "venues": [{"exchange": "binance", "grade": "quiet"}],
                    "summary": "quiet",
                },
            )
        if request.url.path.endswith("/liquidation-levels"):
            return httpx2.Response(
                200,
                json={
                    "kind": "totals",
                    "symbol": request.url.params.get("symbol") or "all",
                    "bars": [
                        {"t": "2026-08-30T12:00:00Z", "longNotional": "10", "shortNotional": "5"}
                    ],
                },
            )
        if request.url.path.endswith("/liquidations/overview"):
            return httpx2.Response(
                200,
                json={
                    "exchange": request.url.params.get("exchange") or "all",
                    "coinWindow": request.url.params.get("window") or "24h",
                    "windows": [
                        {"window": "1h", "longNotional": "80", "shortNotional": "20", "count": 2}
                    ],
                    "coins": [{"symbol": "BTCUSDT", "totalNotional": "100"}],
                },
            )
        if request.url.path.endswith("/liquidations"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "windows": [
                        {"window": "24h", "longNotional": "100", "shortNotional": "50", "count": 3}
                    ],
                },
            )
        if request.url.path.endswith("/orderbook/heatmap"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "windowSeconds": int(request.url.params.get("window") or 600),
                    "columns": [{"t": "2026-08-16T12:00:00Z", "mid": "100"}],
                },
            )
        if request.url.path.endswith("/orderbook/liquidity"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "venueCount": 3,
                    "market": {"score": 72, "grade": "high", "weakerSide": "balanced"},
                },
            )
        if request.url.path.endswith("/orderbook/impact"):
            return httpx2.Response(
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
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "groupSize": request.url.params.get("group") or "0.1",
                    "bids": [{"price": "100", "quantity": "1", "isWall": False}],
                    "asks": [{"price": "100.1", "quantity": "1", "isWall": True}],
                },
            )
        if request.url.path.endswith("/exchanges"):
            return httpx2.Response(200, json={"exchanges": ["binance"], "default": "binance"})
        if request.url.path.endswith("/rsi-heatmap"):
            return httpx2.Response(
                200,
                json={
                    "exchange": "binance",
                    "quote": "USDT",
                    "period": 14,
                    "intervals": ["1h"],
                    "items": [
                        {
                            "symbol": "BTCUSDT",
                            "cells": [{"interval": "1h", "rsi": 42.5, "zone": "neutral"}],
                        }
                    ],
                },
            )
        if request.url.path.endswith("/volume-surge/scan"):
            return httpx2.Response(
                200,
                json={"hits": [{"symbol": "HOTUSDT", "maxRatio": "5", "hottest": "5m"}]},
            )
        if request.url.path.endswith("/volume-surge"):
            return httpx2.Response(
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
            return httpx2.Response(
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
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "summary": "bids absorbing market sells",
                    "combined": {"current": {"kind": "bid", "score": 72}},
                },
            )
        if request.url.path == "/api/v1/alerts" and request.method == "POST":
            body = json.loads(request.content.decode()) if request.content else {}
            return httpx2.Response(
                201,
                json={
                    "id": "a1",
                    "kind": body.get("kind", "price"),
                    "symbol": body.get("symbol", "ALL"),
                    "condition": body.get("condition", "any"),
                    "targetPrice": body.get("targetPrice", 60),
                    "mode": "repeating",
                    "status": "active",
                    "armed": True,
                },
            )
        if request.url.path.endswith("/forensics/radar"):
            return httpx2.Response(
                200,
                json={
                    "summary": "Saved 1 setup(s) over 7d.",
                    "cases": [{"symbol": "BTCUSDT", "lean": "up", "similarity": "82"}],
                },
            )
        if request.url.path.endswith("/forensics/scan"):
            return httpx2.Response(
                200,
                json={
                    "summary": "1 of 4 scanned coin(s) currently look like a past pre-move setup.",
                    "hits": [
                        {
                            "symbol": "BTCUSDT",
                            "similarity": "82",
                            "lean": "up",
                            "afterStory": "When BTC looked like this before, price more often kept rising over the next 1h.",
                        }
                    ],
                },
            )
        if request.url.path.endswith("/forensics"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "summary": "Price rose 2% during. 2 similar cases went up after.",
                    "tape": {
                        "combined": {
                            "phases": [
                                {"phase": "before"},
                                {"phase": "during"},
                                {"phase": "after"},
                            ]
                        }
                    },
                    "similar": {"matches": [{"afterReturnPct": "+3.5"}]},
                },
            )
        if request.url.path.endswith("/around/similar"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "summary": "2 similar past setups",
                    "matches": [{"similarity": "82", "afterReturnPct": "+3.5"}],
                },
            )
        if request.url.path.endswith("/around/precursors"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "summary": "volume was elevated before most up-moves",
                    "patterns": [{"metric": "volume_elevated", "common": True, "side": "up"}],
                },
            )
        if request.url.path.endswith("/around/moves"):
            return httpx2.Response(
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
            return httpx2.Response(
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
            return httpx2.Response(
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
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "combined": {"vwap": "65000", "distancePct": "+1.2", "volume": "1000000"},
                },
            )
        if request.url.path.endswith("/volume-profile"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "exchange": request.url.params.get("exchange") or "all",
                    "window": request.url.params.get("window") or "24h",
                    "poc": {"price": "65200", "volume": "150000"},
                    "valueArea": {"low": "64800", "high": "66100"},
                },
            )
        if request.url.path.endswith("/derivatives-board"):
            return httpx2.Response(
                200,
                json={
                    "exchange": "all",
                    "universe": 1,
                    "definite": False,
                    "summary": "No market read. Missing: okx.",
                    "coins": [
                        {
                            "symbol": "BTCUSDT",
                            "base": "BTC",
                            "openInterestDefinite": False,
                            "fundingDefinite": False,
                        }
                    ],
                },
            )
        if request.url.path.endswith("/perp-depth/overview"):
            return httpx2.Response(
                200,
                json={
                    "exchange": request.url.params.get("exchange") or "all",
                    "rangePct": "1",
                    "known": 2,
                    "bid": "2400000000",
                    "ask": "1800000000",
                    "leaders": ["BTC", "ETH"],
                    "summary": "2 liquid perps: 2.4B bid vs 1.8B ask within 1% of mid (bid-heavy). BTC 48% and ETH 17% of that book.",
                    "coins": [
                        {"symbol": "BTCUSDT", "base": "BTC", "sharePct": "48"},
                        {"symbol": "ETHUSDT", "base": "ETH", "sharePct": "17"},
                    ],
                },
            )
        if request.url.path.endswith("/orderbook/spot-perp"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "rangePct": "1",
                    "thinnerBook": "spot",
                    "strongerBook": "perp",
                    "spot": {"total": "1200000", "thinnerSide": "ask"},
                    "perp": {"total": "55000000", "thinnerSide": "bid"},
                    "summary": "Within 1.0% of mid, the spot book is thinner.",
                },
            )
        if request.url.path.endswith("/perp-depth"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "exchange": request.url.params.get("exchange") or "all",
                    "rangePct": "1",
                    "combined": {
                        "headline": {
                            "below": {"notional": "40000000"},
                            "above": {"notional": "15000000"},
                            "lean": "bid",
                        }
                    },
                },
            )
        if request.url.path.endswith("/liquidity-flow"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "exchange": request.url.params.get("exchange") or "all",
                    "window": request.url.params.get("window") or "5m",
                    "bids": {"pulled": "3000000", "traded": "3000000", "added": "0"},
                    "asks": {"pulled": "0", "traded": "3000000", "added": "0"},
                },
            )
        if request.url.path.endswith("/flow-at-prices/candles"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "exchange": request.url.params.get("exchange") or "all",
                    "interval": request.url.params.get("interval") or "1h",
                    "definite": True,
                    "combined": {
                        "exchange": "all",
                        "candles": [
                            {
                                "time": "2026-10-04T18:00:00Z",
                                "rows": [
                                    {
                                        "price": "100",
                                        "high": "100.05",
                                        "buyVolume": "8",
                                        "sellVolume": "2",
                                        "isPoc": True,
                                        "imbalance": "buy",
                                    }
                                ],
                            }
                        ],
                    },
                },
            )
        if request.url.path.endswith("/flow-at-prices"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "exchange": request.url.params.get("exchange") or "all",
                    "window": request.url.params.get("window") or "15m",
                    "poc": {"price": "65520", "buyVolume": "8000000", "sellVolume": "2000000"},
                    "levels": [
                        {
                            "price": "65520",
                            "buyVolume": "8000000",
                            "sellVolume": "2000000",
                            "trades": 12,
                            "delta": "+6000000",
                        }
                    ],
                },
            )
        if request.url.path.endswith("/situation"):
            return httpx2.Response(
                200,
                json={
                    "symbol": request.url.params.get("symbol"),
                    "at": request.url.params.get("at") or "",
                    "priceCvd": {"kind": "agrees"},
                },
            )
        return httpx2.Response(404, json={"error": "not found"})


def test_market_tools_hit_api(monkeypatch):
    real_client = httpx2.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = _Transport()
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_Transport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
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

    assert "get_liquidation_overview" in by_name
    ov = json.loads(by_name["get_liquidation_overview"].invoke({"window": "1h"}))
    assert ov["coinWindow"] == "1h"
    assert ov["coins"][0]["symbol"] == "BTCUSDT"

    assert "get_liquidation_levels" in by_name
    lv = json.loads(by_name["get_liquidation_levels"].invoke({"symbol": "all"}))
    assert lv["kind"] == "totals"

    assert "get_liquidation_cascade" in by_name
    cas = json.loads(by_name["get_liquidation_cascade"].invoke({"symbol": "BTCUSDT"}))
    assert cas["symbol"] == "BTCUSDT"
    assert "scan_liquidation_cascades" in by_name
    scan = json.loads(by_name["scan_liquidation_cascades"].invoke({}))
    assert scan["hits"][0]["symbol"] == "SOLUSDT"

    assert "get_orderbook_heatmap" in by_name
    heat = json.loads(by_name["get_orderbook_heatmap"].invoke({"symbol": "BTCUSDT"}))
    assert heat["symbol"] == "BTCUSDT"
    assert heat["windowSeconds"] == 600

    assert "get_options_overview" in by_name
    opt = json.loads(by_name["get_options_overview"].invoke({"asset": "BTC"}))
    assert opt["asset"] == "BTC"
    assert opt["dvol"] == "49.5"

    assert "get_open_interest" in by_name
    oi = json.loads(by_name["get_open_interest"].invoke({"symbol": "BTCUSDT"}))
    assert oi["current"]["contracts"] == "100"

    assert "get_funding_rate" in by_name
    fr = json.loads(by_name["get_funding_rate"].invoke({"symbol": "BTCUSDT"}))
    assert fr["venues"][0]["current"]["payer"] == "long"

    assert "get_funding_arb" in by_name
    arb = json.loads(by_name["get_funding_arb"].invoke({"symbol": "BTCUSDT", "notional": 10000}))
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
    assert "create_funding_arb_watch" in by_name
    watch_raw = by_name["create_funding_arb_watch"].invoke(
        {"client_id": "c1", "symbol": "BTCUSDT", "min_profit": 10}
    )
    # Unbound workers fail closed on mutations (no inherited can_trade).
    if "403" in watch_raw:
        assert "read-only" in watch_raw
    else:
        watch = json.loads(watch_raw)
        assert watch["id"] == "w1"
        listed = json.loads(by_name["list_funding_arb_watches"].invoke({"client_id": "c1"}))
        assert "watches" in listed
        assert (
            json.loads(by_name["list_funding_arb_signals"].invoke({"client_id": "c1"}))["signals"]
            == []
        )

    assert "get_long_short_ratio" in by_name
    lsr = json.loads(by_name["get_long_short_ratio"].invoke({"symbol": "BTCUSDT"}))
    assert lsr["venues"][0]["current"]["bias"] == "long"

    assert "get_rsi_heatmap" in by_name
    heat = json.loads(by_name["get_rsi_heatmap"].invoke({"exchange": "binance"}))
    assert heat["items"][0]["symbol"] == "BTCUSDT"

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

    assert "get_market_forensics" in by_name
    forensic = json.loads(
        by_name["get_market_forensics"].invoke({"symbol": "BTCUSDT", "at": "2026-08-20T14:00:00Z"})
    )
    assert forensic["tape"]["combined"]["phases"][1]["phase"] == "during"
    assert forensic["similar"]["matches"][0]["afterReturnPct"] == "+3.5"

    assert "scan_market_forensics" in by_name
    scan = json.loads(by_name["scan_market_forensics"].invoke({}))
    assert scan["hits"][0]["symbol"] == "BTCUSDT"

    assert "review_setup_radar" in by_name
    hist = json.loads(by_name["review_setup_radar"].invoke({}))
    assert hist["cases"][0]["symbol"] == "BTCUSDT"

    assert "create_setup_radar_alert" in by_name
    radar_raw = by_name["create_setup_radar_alert"].invoke(
        {"client_id": "c1", "symbol": "BTCUSDT", "min_similarity": 70, "lean": "up"}
    )
    if "403" in radar_raw:
        assert "read-only" in radar_raw
    else:
        radar = json.loads(radar_raw)
        assert radar["kind"] == "setup_radar"
        assert radar["targetPrice"] == 70

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

    assert "compare_spot_perp_liquidity" in by_name
    compared = json.loads(
        by_name["compare_spot_perp_liquidity"].invoke({"symbol": "BTCUSDT", "range_pct": 1})
    )
    assert compared["thinnerBook"] == "spot"
    assert compared["spot"]["thinnerSide"] == "ask"

    assert "get_perp_depth" in by_name
    perp = json.loads(
        by_name["get_perp_depth"].invoke({"symbol": "BTCUSDT", "range_pct": 1})
    )
    assert perp["symbol"] == "BTCUSDT"
    assert perp["combined"]["headline"]["below"]["notional"] == "40000000"

    assert "get_derivatives_board" in by_name
    board = json.loads(by_name["get_derivatives_board"].invoke({"limit": 20}))
    assert board["coins"][0]["symbol"] == "BTCUSDT"
    assert board["definite"] is False

    assert "get_perp_depth_overview" in by_name
    ov = json.loads(by_name["get_perp_depth_overview"].invoke({"limit": 20}))
    assert ov["leaders"][0] == "BTC"
    assert ov["bid"] == "2400000000"

    assert "get_orderbook_liquidity_flow" in by_name
    liqflow = json.loads(
        by_name["get_orderbook_liquidity_flow"].invoke(
            {"symbol": "BTCUSDT", "exchange": "binance", "window": "5m"}
        )
    )
    assert liqflow["symbol"] == "BTCUSDT"
    assert liqflow["bids"]["pulled"] == "3000000"

    assert "get_flow_at_prices" in by_name
    flow = json.loads(
        by_name["get_flow_at_prices"].invoke(
            {"symbol": "BTCUSDT", "exchange": "all", "window": "15m", "min_price": 65400, "max_price": 65600}
        )
    )
    assert flow["symbol"] == "BTCUSDT"
    assert flow["poc"]["price"] == "65520"
    assert flow["levels"][0]["delta"] == "+6000000"

    assert "get_candle_footprint" in by_name
    footprint = json.loads(
        by_name["get_candle_footprint"].invoke({"symbol": "BTCUSDT", "interval": "1h"})
    )
    assert footprint["combined"]["candles"][0]["rows"][0]["isPoc"] is True

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

    assert "get_coin_situation" in by_name
    assert "Pass at as RFC3339" in by_name["get_coin_situation"].description
    assert "changes" in by_name["get_coin_situation"].description
    live = json.loads(by_name["get_coin_situation"].invoke({"symbol": "BTCUSDT"}))
    assert live["symbol"] == "BTCUSDT"
    assert live["at"] == ""
    assert live["priceCvd"]["kind"] == "agrees"
    past = json.loads(
        by_name["get_coin_situation"].invoke(
            {"symbol": "BTCUSDT", "at": "2026-06-01T16:00:00Z"}
        )
    )
    assert past["at"] == "2026-06-01T16:00:00Z"

    health = by_name["health"].invoke({})
    assert "ok" in health


def test_market_http_reuses_one_client(monkeypatch):
    created: list[httpx2.Client] = []
    real_client = httpx2.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = _Transport()
        kwargs.pop("timeout", None)
        client = real_client(*args, timeout=5.0, transport=_Transport())
        created.append(client)
        return client

    monkeypatch.setattr(httpx2, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {t.name: t for t in tools}
    by_name["get_ticker"].invoke({"symbol": "BTCUSDT", "exchange": "binance"})
    by_name["health"].invoke({})
    assert len(created) == 1


def test_read_only_scope_blocks_mutations(monkeypatch):
    posted: list[str] = []

    class _MutTransport(httpx2.BaseTransport):
        def handle_request(self, request: httpx2.Request) -> httpx2.Response:
            posted.append(f"{request.method} {request.url.path}")
            return httpx2.Response(200, json={"ok": True})

    real_client = httpx2.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = _MutTransport()
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_MutTransport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
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


def test_get_cvd_sends_click_anchor(monkeypatch):
    seen: list[httpx2.Request] = []
    real_client = httpx2.Client

    class _CvdTransport(httpx2.BaseTransport):
        def handle_request(self, request: httpx2.Request) -> httpx2.Response:
            seen.append(request)
            return httpx2.Response(200, json={"symbol": "BTCUSDT", "range": "7d"})

    def fake_client(*args, **kwargs):
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_CvdTransport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {t.name: t for t in tools}
    out = json.loads(
        by_name["get_cvd"].invoke(
            {
                "symbol": "BTCUSDT",
                "range": "7d",
                "anchor": "2026-10-07T08:00:00Z",
            }
        )
    )
    assert out["symbol"] == "BTCUSDT"
    assert seen[-1].url.path.endswith("/cvd")
    assert seen[-1].url.params["anchor"] == "2026-10-07T08:00:00Z"
    assert seen[-1].url.params["range"] == "7d"
    by_name["get_cvd"].invoke({"symbol": "BTCUSDT"})
    assert "anchor" not in seen[-1].url.params


def test_get_positioning_anchor_sends_candle(monkeypatch):
    seen: list[httpx2.Request] = []
    real_client = httpx2.Client

    class _AnchorTransport(httpx2.BaseTransport):
        def handle_request(self, request: httpx2.Request) -> httpx2.Response:
            seen.append(request)
            return httpx2.Response(200, json={"symbol": "BTCUSDT", "bucket": "5m"})

    def fake_client(*args, **kwargs):
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=_AnchorTransport())

    monkeypatch.setattr(httpx2, "Client", fake_client)
    tools = build_market_tools(Settings(api_base_url="http://test"))
    by_name = {t.name: t for t in tools}
    out = json.loads(
        by_name["get_positioning_anchor"].invoke(
            {
                "symbol": "BTCUSDT",
                "anchor": "2026-10-08T12:00:00Z",
                "bucket": "5m",
                "start": "2026-10-08T00:00:00Z",
            }
        )
    )
    assert out["symbol"] == "BTCUSDT"
    assert seen[-1].url.path.endswith("/positioning-anchor")
    assert seen[-1].url.params["anchor"] == "2026-10-08T12:00:00Z"
    assert seen[-1].url.params["bucket"] == "5m"
    assert seen[-1].url.params["from"] == "2026-10-08T00:00:00Z"


def test_memory_key_isolates_tenants():
    mem = SessionMemory()
    a = memory_key("alice", "shared")
    b = memory_key("bob", "shared")
    assert a != b
    mem.append(a, [HumanMessage(content="alice secret"), AIMessage(content="ok")])
    assert mem.get(b) == []
    assert len(mem.get(a)) == 2
