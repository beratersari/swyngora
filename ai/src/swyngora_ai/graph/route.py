"""Deterministic desk router (no LLM). Parallel specialists + optional debate."""

from __future__ import annotations

import re
from dataclasses import asdict, dataclass

from swyngora_ai.sources.identity import find_instruments, resolve_topic

_TICKER_RE = re.compile(
    r"\b([A-Z]{2,6}(?:USDT|USD)?|bitcoin|ethereum|solana|\$[A-Za-z]{2,6})\b",
    re.IGNORECASE,
)


@dataclass
class Route:
    tape: bool = False
    book: bool = False
    paper: bool = False
    account: bool = False
    web: bool = False
    social: bool = False
    debate: bool = False
    desk_note: bool = False

    def as_dict(self) -> dict[str, bool]:
        return asdict(self)

    def any_market(self) -> bool:
        return self.tape or self.book or self.paper


def classify_route(message: str) -> Route:
    text = (message or "").strip()
    low = text.lower()
    route = Route()

    named = bool(_TICKER_RE.search(text)) or bool(find_instruments(text))
    # Broader identity: a single known ticker word.
    resolved = resolve_topic(text.split()[0] if text.split() else text)
    if resolved.kind != "unknown":
        named = True

    if any(
        k in low
        for k in (
            "price",
            "fiyat",
            "rsi",
            "ema",
            "candle",
            "ticker",
            "mcap",
            "supply",
            "fx",
            "last",
            "delist",
        )
    ):
        route.tape = True
    if any(
        k in low
        for k in (
            "order book",
            "orderbook",
            "wall",
            "liquidity",
            "liquidation",
            "slippage",
            "impact",
            "pump",
            "swing",
            "depth",
        )
    ):
        route.book = True
        route.tape = True
    if any(
        k in low
        for k in (
            "portfolio",
            "paper",
            "place order",
            "buy ",
            "sell ",
            "margin",
            "oco",
            "bracket",
            "amend",
            "cancel all",
            "recurring",
            "basket",
            "rebalance",
            "deposit",
            "withdraw",
            "lot",
        )
    ):
        route.paper = True
    if any(
        k in low
        for k in (
            "watchlist",
            "alert",
            "api key",
            "export",
            "import",
            "webhook",
            "scanner",
        )
    ):
        route.account = True
    if any(
        k in low
        for k in (
            "news",
            "headline",
            "catalyst",
            "what is",
            "nedir",
            "who is",
            "filing",
            "8-k",
            "10-q",
            "kap",
            "sec",
            "project",
            "regulation",
        )
    ):
        route.web = True
    if any(
        k in low
        for k in (
            "twitter",
            "x.com",
            "sentiment",
            "stocktwits",
            "chatter",
            "social",
            "reddit",
        )
    ):
        route.social = True
    if any(
        k in low
        for k in (
            "should i",
            "lean long",
            "lean short",
            "long or short",
            "buy or sell",
            "bull or bear",
            "go long",
            "go short",
        )
    ):
        route.debate = True
        route.tape = True
        route.web = True
        route.desk_note = True

    if any(
        k in low
        for k in (
            "analysis",
            "analiz",
            "outlook",
            "görünüm",
            "gorunum",
            "bias",
            "1-2 day",
            "1–2 day",
            "1-2 gün",
            "1–2 gün",
            "what do you think",
            "ne düşün",
            "ne dusun",
            "should i watch",
            "izleyeyim",
            "tactical",
        )
    ):
        route.desk_note = True
        route.tape = True
        route.web = True

    # Named name alone is not a request for tape or a 1–2 day brief.
    if named and not (route.tape or route.book or route.paper or route.account):
        route.web = True

    return route
