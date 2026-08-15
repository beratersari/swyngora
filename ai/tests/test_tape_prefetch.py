import json
from typing import Any

from swyngora_ai.config import Settings
from swyngora_ai.graph.desk import _packet
from swyngora_ai.graph.facts import extract_market_facts
from swyngora_ai.graph.tape_fetch import prefetch_tape


def test_prefetch_calls_ticker(monkeypatch):
    from swyngora_ai.graph import tape_fetch as tf

    class _Tool:
        def __init__(self, name: str) -> None:
            self.name = name
            self.calls: list[dict[str, Any]] = []

        def invoke(self, args: dict[str, Any]) -> str:
            self.calls.append(args)
            if self.name == "get_ticker":
                return json.dumps(
                    {
                        "symbol": args["symbol"],
                        "exchange": args["exchange"],
                        "lastPrice": "312.4",
                    }
                )
            return json.dumps({"rsi": 51})

    ticker = _Tool("get_ticker")
    indicators = _Tool("get_indicators")

    def fake_build(settings, pack=None):
        return [ticker, indicators]

    monkeypatch.setattr(tf, "build_market_tools", fake_build)
    blobs = prefetch_tape(Settings(_env_file=None), "THY longest flight")
    assert ticker.calls
    assert ticker.calls[0]["symbol"] == "THYAO"
    assert ticker.calls[0]["exchange"] == "bist"
    facts = extract_market_facts(*blobs)
    assert facts.last_price == "312.4"
    assert facts.symbol == "THYAO"


def test_packet_never_mentions_stale_cache():
    text = _packet(
        {
            "message": "THY nedir?",
            "facts": {"last_price": "312.4", "symbol": "THYAO", "exchange": "bist"},
            "memory_context": "Tape cache stale (>300s) — re-fetch live numbers.",
        }
    )
    assert "Tape cache stale" not in text
    assert "re-fetch live numbers" not in text
    assert "312.4" in text
    assert "Bottom line" in text or "Sadece soruyu" in text
