"""Tests for free social / x_search tool."""

from __future__ import annotations

from unittest.mock import patch

from swyngora_ai.tools.x_search import _guess_symbols, _x_search


def test_guess_symbols_aliases_and_pairs():
    assert "BTC" in _guess_symbols("btc ne olur twitter")
    assert "BTC" in _guess_symbols("What about BTCUSDT?")
    assert "ETH" in _guess_symbols("$ETH price")
    assert "JUV" in _guess_symbols("deep analysis for JUV")


def test_x_search_stocktwits_primary():
    fake_st = {
        "symbol": {"symbol": "BTC.X", "title": "Bitcoin"},
        "messages": [
            {
                "body": "$BTC looking strong today",
                "created_at": "2026-07-26T12:00:00Z",
                "user": {"username": "trader1"},
                "likes": {"total": 3},
            },
            {
                "body": "noise about alts",
                "created_at": "2026-07-26T11:00:00Z",
                "user": {"username": "noise"},
                "likes": {"total": 0},
            },
        ],
    }

    def fake_json(url: str, timeout: float = 12.0):
        if "stocktwits.com" in url:
            return fake_st
        if "algolia" in url:
            return {"hits": [{"title": "BTC thread", "points": 10, "url": "https://example.com"}]}
        raise AssertionError(url)

    with (
        patch("swyngora_ai.tools.x_search._http_json", side_effect=fake_json),
        patch("swyngora_ai.tools.x_search._fetch_ddg_social", return_value=[]),
    ):
        out = _x_search("BTC twitter sentiment", max_results=5)

    assert "StockTwits" in out
    assert "looking strong" in out
    assert "@trader1" in out
    assert "NOT the official X API" in out
    # Must not pretend site:x.com worked
    assert "No indexed X/Twitter results" not in out


def test_x_search_empty_symbols_still_tries_query():
    with (
        patch("swyngora_ai.tools.x_search._fetch_stocktwits", return_value=[]),
        patch("swyngora_ai.tools.x_search._fetch_hn", return_value=[]),
        patch("swyngora_ai.tools.x_search._fetch_ddg_social", return_value=[]),
    ):
        out = _x_search("totallyunknownassetxyz", max_results=3)
    assert "No social results" in out
