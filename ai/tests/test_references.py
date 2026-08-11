from swyngora_ai.references import classify_source, extract_references


def test_extract_labeled_and_hn_urls():
    blob = """
1. Bitcoin price today
   URL: https://coinmarketcap.com/currencies/bitcoin/
   Live tape.
HN: Senate Clarity Act delay (pts=12) https://news.ycombinator.com/item?id=1
[CoinDesk whales](https://www.coindesk.com/markets/2026/08/07/bitcoin-whales)
"""
    refs = extract_references(blob)
    urls = [r.url for r in refs]
    assert "https://coinmarketcap.com/currencies/bitcoin/" in urls
    assert any("coindesk.com" in u for u in urls)
    assert any(r.source == "hn" for r in refs)


def test_dedupe_and_classify_x():
    blob = """
StockTwits BTC.X
URL: https://stocktwits.com/symbol/BTC
URL: https://stocktwits.com/symbol/BTC
https://x.com/DaanCrypto/status/1
"""
    refs = extract_references(blob)
    assert len(refs) == 2
    assert all(r.source == "x" for r in refs)


def test_skip_errors():
    assert extract_references("ERROR web_search: rate-limited") == []


def test_classify_source():
    assert classify_source("https://twitter.com/a/status/1") == "x"
    assert classify_source("https://www.coingecko.com/en/coins/bitcoin") == "web"
