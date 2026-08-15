import json

from swyngora_ai.graph.facts import extract_market_facts
from swyngora_ai.grounding import apply_grounding, strip_unallowed_urls
from swyngora_ai.references import Reference
from swyngora_ai.sources.allowlist import classify_reliability, filter_references


def test_extract_market_facts_from_ticker_json():
    blob = json.dumps(
        {
            "symbol": "BTCUSDT",
            "exchange": "binance",
            "lastPrice": "67000.5",
            "priceChangePercent": "1.2",
            "highPrice": "68000",
            "lowPrice": "66000",
            "quoteVolume": "1000000",
        }
    )
    facts = extract_market_facts(blob)
    assert facts.symbol == "BTCUSDT"
    assert facts.last_price == "67000.5"
    assert "67000.5" in facts.numbers
    assert "MarketFacts" in facts.as_prompt()


def test_strip_unknown_urls_keep_allowlisted():
    text = (
        "See [CoinDesk](https://www.coindesk.com/markets/x) "
        "and [spam](https://evil-pumps.example/buy) "
        "https://sec.gov/cgi-bin/browse-edgar"
    )
    out = strip_unallowed_urls(text)
    assert "coindesk.com" in out
    assert "sec.gov" in out
    assert "evil-pumps" not in out
    assert "spam" in out  # title kept


def test_grounding_flags_unverified_price():
    facts = extract_market_facts(json.dumps({"lastPrice": "100.25"}))
    reply = "BTC last is 99999.12 on binance."
    out = apply_grounding(reply, facts, [json.dumps({"lastPrice": "100.25"})])
    assert "Unverified" in out
    assert "99999.12" in out


def test_grounding_accepts_tool_number():
    facts = extract_market_facts(json.dumps({"lastPrice": "100.25", "rsi": 42}))
    reply = "Last 100.25 with RSI 42. Informational only — not financial advice."
    out = apply_grounding(reply, facts, [])
    assert "Unverified" not in out


def test_reliability_classes():
    assert classify_reliability("https://www.sec.gov/ix?doc=/x") == "primary"
    assert classify_reliability("https://www.coindesk.com/x") == "newsroom"
    assert classify_reliability("https://stocktwits.com/symbol/BTC") == "weak"
    assert classify_reliability("https://random-blog.example/p") == "unknown"
    refs = filter_references(
        [
            Reference(title="a", url="https://www.coindesk.com/a", source="news"),
            Reference(title="b", url="https://random-blog.example/p", source="web"),
        ]
    )
    assert len(refs) == 1
    assert refs[0].reliability == "newsroom"
