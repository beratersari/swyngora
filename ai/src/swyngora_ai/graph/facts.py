"""Structured MarketFacts extracted from tool JSON (numbers from code, not the LLM)."""

from __future__ import annotations

import json
import re
from dataclasses import asdict, dataclass, field
from typing import Any

_NUM_RE = re.compile(r"-?\d+(?:\.\d+)?")

_PRICE_KEYS = (
    "lastPrice",
    "last",
    "price",
    "close",
    "bestBid",
    "bestAsk",
    "averagePrice",
)
_CHANGE_KEYS = ("priceChangePercent", "changePercent", "changePct", "pctChange")
_RSI_KEYS = ("rsi", "RSI")
_VOL_KEYS = ("quoteVolume", "volume", "volume24h")
_HIGH_KEYS = ("highPrice", "high")
_LOW_KEYS = ("lowPrice", "low")


@dataclass
class MarketFacts:
    symbol: str = ""
    exchange: str = ""
    last_price: str = ""
    change_24h: str = ""
    rsi: str = ""
    high: str = ""
    low: str = ""
    volume: str = ""
    as_of: str = ""
    numbers: list[str] = field(default_factory=list)

    def as_dict(self) -> dict[str, Any]:
        return asdict(self)

    def as_prompt(self) -> str:
        rows = []
        for k, v in self.as_dict().items():
            if k == "numbers":
                continue
            if v:
                rows.append(f"- {k}: {v}")
        if not rows:
            return "(no structured market facts this turn)"
        return "MarketFacts (tool-extracted, do not invent others):\n" + "\n".join(rows)


def _walk_numbers(value: Any, out: list[str]) -> None:
    if isinstance(value, dict):
        for v in value.values():
            _walk_numbers(v, out)
    elif isinstance(value, list):
        for v in value:
            _walk_numbers(v, out)
    elif isinstance(value, bool):
        return
    elif isinstance(value, int | float):
        out.append(str(value))
    elif isinstance(value, str):
        s = value.strip()
        if _NUM_RE.fullmatch(s):
            out.append(s)


def _first(data: dict[str, Any], keys: tuple[str, ...]) -> str:
    for k in keys:
        if k in data and data[k] not in (None, ""):
            return str(data[k])
    return ""


def _parse_json_blobs(blobs: list[str]) -> list[dict[str, Any]]:
    found: list[dict[str, Any]] = []
    for blob in blobs:
        text = (blob or "").strip()
        if not text or text.startswith(("ERROR", "(")):
            continue
        try:
            data = json.loads(text)
        except json.JSONDecodeError:
            continue
        if isinstance(data, dict):
            found.append(data)
        elif isinstance(data, list):
            found.extend(x for x in data if isinstance(x, dict))
    return found


def extract_market_facts(*blobs: str) -> MarketFacts:
    """Pull known fields + every numeric token from tool JSON payloads."""
    facts = MarketFacts()
    nums: list[str] = []
    for data in _parse_json_blobs(list(blobs)):
        _walk_numbers(data, nums)
        if not facts.symbol:
            facts.symbol = str(data.get("symbol") or data.get("asset") or "")
        if not facts.exchange:
            facts.exchange = str(data.get("exchange") or "")
        if not facts.last_price:
            facts.last_price = _first(data, _PRICE_KEYS)
        if not facts.change_24h:
            facts.change_24h = _first(data, _CHANGE_KEYS)
        if not facts.rsi:
            rsi = data.get("rsi") or (data.get("indicators") or {}).get("rsi")
            if isinstance(rsi, dict):
                rsi = rsi.get("value") or rsi.get("rsi")
            if rsi not in (None, ""):
                facts.rsi = str(rsi)
        if not facts.high:
            facts.high = _first(data, _HIGH_KEYS)
        if not facts.low:
            facts.low = _first(data, _LOW_KEYS)
        if not facts.volume:
            facts.volume = _first(data, _VOL_KEYS)
        if not facts.as_of:
            facts.as_of = str(data.get("updatedAt") or data.get("closeTime") or "")
    # unique, preserve order
    seen: set[str] = set()
    for n in nums:
        if n not in seen:
            seen.add(n)
            facts.numbers.append(n)
    return facts
