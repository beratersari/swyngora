"""Resolve a user topic to a wiki title, gecko id, and venue hint."""

from __future__ import annotations

import re
from dataclasses import dataclass

# Tickers / pair suffixes → Wikipedia title (identity only, not prices).
_WIKI_ALIASES: dict[str, str] = {
    "BTC": "Bitcoin",
    "BTCUSDT": "Bitcoin",
    "BTCUSD": "Bitcoin",
    "BITCOIN": "Bitcoin",
    "ETH": "Ethereum",
    "ETHUSDT": "Ethereum",
    "ETHUSD": "Ethereum",
    "ETHEREUM": "Ethereum",
    "SOL": "Solana (blockchain platform)",
    "SOLUSDT": "Solana (blockchain platform)",
    "SOLANA": "Solana (blockchain platform)",
    "XRP": "XRP",
    "XRPUSDT": "XRP",
    "RIPPLE": "XRP",
    "DOGE": "Dogecoin",
    "DOGEUSDT": "Dogecoin",
    "DOGECOIN": "Dogecoin",
    "ADA": "Cardano (blockchain platform)",
    "ADAUSDT": "Cardano (blockchain platform)",
    "CARDANO": "Cardano (blockchain platform)",
    "BNB": "BNB",
    "BNBUSDT": "BNB",
    "AVAX": "Avalanche (blockchain platform)",
    "AVAXUSDT": "Avalanche (blockchain platform)",
    "DOT": "Polkadot (cryptocurrency)",
    "LINK": "Chainlink (blockchain)",
    "MATIC": "Polygon (blockchain)",
    "POL": "Polygon (blockchain)",
    "PEPE": "Pepe (cryptocurrency)",
    "SHIB": "Shiba Inu (cryptocurrency)",
    "TON": "Toncoin",
    "SUI": "Sui (blockchain)",
    "APT": "Aptos (blockchain)",
    "ARB": "Arbitrum",
    "OP": "Optimism (blockchain)",
    "JUV": "Juventus F.C. Fan Token",
    "JUVUSDT": "Juventus F.C. Fan Token",
    "JUVENTUS": "Juventus F.C. Fan Token",
    "AAPL": "Apple Inc.",
    "MSFT": "Microsoft",
    "GOOGL": "Alphabet Inc.",
    "GOOG": "Alphabet Inc.",
    "AMZN": "Amazon (company)",
    "NVDA": "Nvidia",
    "TSLA": "Tesla, Inc.",
    "META": "Meta Platforms",
    "NFLX": "Netflix",
    "AMD": "Advanced Micro Devices",
    "INTC": "Intel",
    "JPM": "JPMorgan Chase",
    "THYAO": "Turkish Airlines",
    "THY": "Turkish Airlines",
    "PGSUS": "Pegasus Airlines",
    "GARAN": "Garanti BBVA",
    "AKBNK": "Akbank",
    "EREGL": "Erdemir",
    "SISE": "Şişecam",
    "KCHOL": "Koç Holding",
    "SAHOL": "Sabancı Holding",
    "BIMAS": "BIM (supermarket)",
    "ASELS": "Aselsan",
    "TUPRS": "Tüpraş",
}


@dataclass(frozen=True)
class ResolvedTopic:
    raw: str
    wiki_title: str
    base: str
    venue_hint: str
    kind: str  # crypto | equity | unknown


@dataclass(frozen=True)
class Instrument:
    symbol: str
    exchange: str
    kind: str


_NASDAQ = frozenset(
    {
        "AAPL",
        "MSFT",
        "GOOGL",
        "GOOG",
        "AMZN",
        "NVDA",
        "TSLA",
        "META",
        "NFLX",
        "AMD",
        "INTC",
        "JPM",
    }
)
_BIST = frozenset(
    {
        "THYAO",
        "THY",
        "PGSUS",
        "GARAN",
        "AKBNK",
        "EREGL",
        "SISE",
        "KCHOL",
        "SAHOL",
        "BIMAS",
        "ASELS",
        "TUPRS",
    }
)
_BIST_REMAP = {"THY": "THYAO"}

# Folded phrases in the user text → alias key.
_PHRASES: tuple[tuple[str, str], ...] = (
    ("turkish airlines", "THYAO"),
    ("turk hava yollari", "THYAO"),
    ("turk hava", "THYAO"),
    ("thy ao", "THYAO"),
    ("pegasus", "PGSUS"),
)


def _norm_key(topic: str) -> str:
    return re.sub(r"[^A-Za-z0-9]", "", topic).upper()


def wiki_title(topic: str) -> str:
    t = (topic or "").strip()
    if not t:
        return t
    key = _norm_key(t)
    if key in _WIKI_ALIASES:
        return _WIKI_ALIASES[key]
    m = re.fullmatch(r"([A-Za-z]{1,10})(USDT|USD)", t, re.IGNORECASE)
    if m:
        inner = m.group(1).upper()
        if inner in _WIKI_ALIASES:
            return _WIKI_ALIASES[inner]
    return t


def _fold(text: str) -> str:
    table = str.maketrans(
        {
            "ç": "c",
            "ğ": "g",
            "ı": "i",
            "ö": "o",
            "ş": "s",
            "ü": "u",
            "Ç": "c",
            "Ğ": "g",
            "İ": "i",
            "Ö": "o",
            "Ş": "s",
            "Ü": "u",
        }
    )
    return (text or "").translate(table).lower()


def resolve_topic(topic: str) -> ResolvedTopic:
    raw = (topic or "").strip()
    key = _norm_key(raw)
    if key in _BIST_REMAP:
        key = _BIST_REMAP[key]
    title = wiki_title(key if key in _WIKI_ALIASES else raw)
    venue = ""
    kind = "unknown"
    base = raw
    m = re.fullmatch(r"([A-Za-z]{1,10})(USDT|USD)", raw, re.IGNORECASE)
    if m:
        base = m.group(1).upper()
        kind = "crypto"
        venue = "binance"
    elif key in _NASDAQ:
        base = key
        kind = "equity"
        venue = "nasdaq"
    elif key in _BIST:
        base = _BIST_REMAP.get(key, key)
        kind = "equity"
        venue = "bist"
    elif key in _WIKI_ALIASES and key not in {"APPLE", "MICROSOFT"}:
        base = key
        kind = "crypto"
        venue = "binance"
    return ResolvedTopic(raw=raw, wiki_title=title, base=base, venue_hint=venue, kind=kind)


def to_instrument(topic: ResolvedTopic) -> Instrument | None:
    if topic.kind == "unknown" or not topic.base:
        return None
    if topic.kind == "equity":
        return Instrument(symbol=topic.base, exchange=topic.venue_hint or "nasdaq", kind="equity")
    base = topic.base.upper()
    if base.endswith(("USDT", "USD")):
        symbol = base
    else:
        symbol = f"{base}USDT"
    return Instrument(symbol=symbol, exchange=topic.venue_hint or "binance", kind="crypto")


def find_instruments(text: str) -> list[Instrument]:
    """Pull tradable symbols from free text (tickers + common names)."""
    found: list[Instrument] = []
    seen: set[tuple[str, str]] = set()

    def add(inst: Instrument | None) -> None:
        if inst is None:
            return
        key = (inst.symbol, inst.exchange)
        if key in seen:
            return
        seen.add(key)
        found.append(inst)

    folded = _fold(text)
    for phrase, alias in _PHRASES:
        if phrase in folded:
            add(to_instrument(resolve_topic(alias)))

    for tok in re.findall(r"[A-Za-z]{2,12}", text or ""):
        add(to_instrument(resolve_topic(tok)))

    return found
