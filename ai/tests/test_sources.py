from swyngora_ai.sources.feeds import parse_rss_items
from swyngora_ai.sources.identity import find_instruments, resolve_topic, wiki_title


def test_wiki_aliases_cover_fan_token_and_equities():
    assert wiki_title("JUVUSDT") == "Juventus F.C. Fan Token"
    assert wiki_title("AAPL") == "Apple Inc."
    assert wiki_title("THYAO") == "Turkish Airlines"
    assert wiki_title("BTCUSDT") == "Bitcoin"


def test_resolve_topic_venues():
    assert resolve_topic("AAPL").venue_hint == "nasdaq"
    assert resolve_topic("THYAO").venue_hint == "bist"
    assert resolve_topic("THY").base == "THYAO"
    assert resolve_topic("THY").venue_hint == "bist"
    assert resolve_topic("ETHUSDT").venue_hint == "binance"
    assert resolve_topic("ETHUSDT").kind == "crypto"


def test_find_instruments_from_prose():
    thy = find_instruments("THY longest flight and fuel costs")
    assert any(i.symbol == "THYAO" and i.exchange == "bist" for i in thy)
    named = find_instruments("türk hava yolları yakıt maliyeti")
    assert any(i.symbol == "THYAO" for i in named)
    btc = find_instruments("what is BTC doing")
    assert any(i.symbol == "BTCUSDT" and i.exchange == "binance" for i in btc)


def test_parse_rss_items():
    xml = b"""<?xml version="1.0"?>
    <rss><channel>
      <item>
        <title>BTC whales</title>
        <link>https://www.coindesk.com/markets/example</link>
        <pubDate>Fri, 07 Aug 2026 12:00:00 GMT</pubDate>
      </item>
    </channel></rss>"""
    lines = parse_rss_items(xml, "CoinDesk", 3)
    assert "CoinDesk" in lines[0]
    assert "https://www.coindesk.com/markets/example" in lines[0]
