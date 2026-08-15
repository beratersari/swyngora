"""Web research uses Wikipedia / RSS / CoinGecko — not live DDG."""

from __future__ import annotations

from unittest.mock import patch

from swyngora_ai.tools import web_search as ws


def test_wiki_title_aliases():
    assert ws._wiki_title("BTCUSDT") == "Bitcoin"
    assert ws._wiki_title("sol") == "Solana (blockchain platform)"
    assert ws._wiki_title("Ethereum") == "Ethereum"


def test_wikipedia_summary_formats_url():
    fake = {
        "title": "Bitcoin",
        "extract": "Bitcoin is a cryptocurrency.",
        "content_urls": {"desktop": {"page": "https://en.wikipedia.org/wiki/Bitcoin"}},
        "type": "standard",
    }
    with patch.object(ws, "_http_json", return_value=fake):
        out = ws._wikipedia("Bitcoin")
    assert "URL: https://en.wikipedia.org/wiki/Bitcoin" in out
    assert "Bitcoin is a cryptocurrency" in out


def test_google_news_rss_parses_items():
    xml = b"""<?xml version="1.0"?>
    <rss><channel>
      <item>
        <title>BTC whales buy</title>
        <link>https://www.coindesk.com/markets/example</link>
        <pubDate>Fri, 07 Aug 2026 12:00:00 GMT</pubDate>
        <source>CoinDesk</source>
      </item>
    </channel></rss>"""
    with patch.object(ws, "_http_bytes", return_value=xml):
        out = ws._google_news_rss("bitcoin", 3)
    assert "CoinDesk" in out
    assert "URL: https://www.coindesk.com/markets/example" in out


def test_coingecko_search_links():
    fake = {"coins": [{"id": "bitcoin", "name": "Bitcoin", "symbol": "btc", "market_cap_rank": 1}]}
    with patch.object(ws, "_http_json", return_value=fake):
        out = ws._coingecko("BTC")
    assert "https://www.coingecko.com/en/coins/bitcoin" in out
    assert "coinmarketcap.com/currencies/bitcoin" in out


def test_research_skips_ddg_when_rss_and_wiki_work():
    with (
        patch.object(
            ws,
            "_wikipedia",
            return_value="1. [Wikipedia] Bitcoin\n   URL: https://en.wikipedia.org/wiki/Bitcoin",
        ),
        patch.object(
            ws,
            "_coingecko",
            return_value="1. [CoinGecko] Bitcoin\n   URL: https://www.coingecko.com/en/coins/bitcoin",
        ),
        patch.object(
            ws,
            "_google_news_rss",
            return_value="1. [CoinDesk] Hello\n   URL: https://www.coindesk.com/x",
        ),
        patch.object(
            ws,
            "_hn_news",
            return_value="1. [Hacker News] Thread\n   URL: https://news.ycombinator.com/item?id=1",
        ),
        patch.object(ws, "_allowlisted_rss", return_value=""),
        patch.object(ws, "_search", return_value="ERROR web_search: timeout") as ddg,
    ):
        out = ws._research("BTC", 6)
    ddg.assert_called()
    assert "wikipedia.org/wiki/Bitcoin" in out
    assert "coindesk.com" in out
    assert "No public web/news hits" not in out
