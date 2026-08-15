"""Allowlisted publisher RSS (direct URLs, not Google wrappers)."""

from __future__ import annotations

import xml.etree.ElementTree as ET
from typing import Any

_FEEDS: tuple[tuple[str, str], ...] = (
    ("CoinDesk", "https://www.coindesk.com/arc/outboundfeeds/rss/"),
    ("The Block", "https://www.theblock.co/rss.xml"),
    ("Decrypt", "https://decrypt.co/feed"),
    ("Binance announcements", "https://www.binance.com/en/support/announcement/rss"),
)


def parse_rss_items(raw: bytes, source: str, max_results: int) -> list[str]:
    try:
        root = ET.fromstring(raw)
    except ET.ParseError as e:
        return [f"({source} RSS parse: {e})"]
    items = root.findall("./channel/item")
    if not items:
        items = root.findall(".//{http://www.w3.org/2005/Atom}entry")
    lines: list[str] = []
    for item in items[:max_results]:
        title = (
            item.findtext("title") or item.findtext("{http://www.w3.org/2005/Atom}title") or ""
        ).strip()
        link = (item.findtext("link") or "").strip()
        if not link:
            atom_link = item.find("{http://www.w3.org/2005/Atom}link")
            if atom_link is not None:
                link = (atom_link.get("href") or "").strip()
        pub = (
            item.findtext("pubDate") or item.findtext("{http://www.w3.org/2005/Atom}updated") or ""
        ).strip()
        if not title or not link:
            continue
        lines.append(f"{len(lines) + 1}. [{source}] {title} ({pub})\n   URL: {link}")
    return lines


def feed_catalog() -> tuple[tuple[str, str], ...]:
    return _FEEDS


def match_feeds(topic: str) -> list[tuple[str, str]]:
    """Return feeds likely relevant to the topic (all crypto feeds + venue notes)."""
    _ = topic
    return list(_FEEDS)


def format_feed_error(source: str, err: Any) -> str:
    return f"({source} RSS: {err})"
