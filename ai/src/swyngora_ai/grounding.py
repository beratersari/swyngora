"""Citation filter + cheap number groundedness against MarketFacts."""

from __future__ import annotations

import re
from urllib.parse import urlparse

from swyngora_ai.graph.facts import MarketFacts
from swyngora_ai.language import detect_reply_lang, low_confidence_note, unverified_note
from swyngora_ai.references import Reference, extract_references
from swyngora_ai.sources.allowlist import filter_references, host_allowed

_MD_LINK = re.compile(r"\[([^\]]+)\]\((https?://[^)\s]+)\)")
_BARE_URL = re.compile(r"https?://[^\s\)\]\>\"']+", re.IGNORECASE)
# Price-like or indicator-like: decimals, or integers with 3+ digits.
_CLAIM_NUM = re.compile(r"(?<![\w./-])(\d{1,3}(?:,\d{3})+(?:\.\d+)?|\d+\.\d+|\d{3,})(?![\w./])")

_SKIP_NEAR = re.compile(
    r"\b(1-2|1–2|24-48|24–48|2-4|2–4|day|days|hours|hour)\b",
    re.IGNORECASE,
)


def strip_unallowed_urls(reply: str, extra_hosts: set[str] | None = None) -> str:
    """Remove markdown/bare URLs whose host is not allowlisted or in extra_hosts."""

    def keep(url: str) -> bool:
        return host_allowed(url, extra_hosts=extra_hosts)

    def md_sub(m: re.Match[str]) -> str:
        return m.group(0) if keep(m.group(2)) else m.group(1)

    out = _MD_LINK.sub(md_sub, reply)

    def bare_sub(m: re.Match[str]) -> str:
        return m.group(0) if keep(m.group(0)) else ""

    out = _BARE_URL.sub(bare_sub, out)
    return re.sub(r"\n{3,}", "\n\n", out).strip()


def extra_hosts_from_blobs(*blobs: str) -> set[str]:
    hosts: set[str] = set()
    for blob in blobs:
        for m in _BARE_URL.finditer(blob or ""):
            try:
                h = (urlparse(m.group(0)).hostname or "").lower().removeprefix("www.")
            except ValueError:
                continue
            if h:
                hosts.add(h)
    return hosts


def grounded_references(*blobs: str, reply: str = "") -> list[Reference]:
    extra = extra_hosts_from_blobs(*blobs, reply)
    refs = extract_references(*blobs, reply)
    return filter_references(refs, extra_hosts=extra)


def apply_grounding(
    reply: str,
    facts: MarketFacts,
    tool_blobs: list[str],
    user_message: str = "",
) -> str:
    """Keep the reply; flag unverified price-like numbers when tools ran."""
    lang = detect_reply_lang(user_message or reply)
    extra = extra_hosts_from_blobs(*tool_blobs)
    cleaned = strip_unallowed_urls(reply, extra_hosts=extra)
    allowed = set(facts.numbers)
    if facts.last_price:
        allowed.add(facts.last_price)
    if facts.rsi:
        allowed.add(facts.rsi)
    if facts.change_24h:
        allowed.add(str(facts.change_24h).rstrip("%"))

    if not allowed:
        if _looks_like_prints(cleaned):
            return f"{cleaned.rstrip()}\n\n{low_confidence_note(lang)}"
        return cleaned

    unverified: list[str] = []
    for m in _CLAIM_NUM.finditer(cleaned):
        raw = m.group(1).replace(",", "")
        span = cleaned[max(0, m.start() - 12) : m.end() + 12]
        if _SKIP_NEAR.search(span):
            continue
        if raw in allowed or _close_enough(raw, allowed):
            continue
        # tiny integers (1–31) are usually days / ranks
        try:
            if "." not in raw and abs(int(raw)) <= 31:
                continue
        except ValueError:
            pass
        if raw not in unverified:
            unverified.append(raw)
    if unverified:
        shown = ", ".join(unverified[:6])
        return f"{cleaned.rstrip()}\n\n{unverified_note(lang, shown)}"
    return cleaned


def _close_enough(raw: str, allowed: set[str]) -> bool:
    try:
        val = float(raw)
    except ValueError:
        return False
    for a in allowed:
        try:
            if abs(float(a) - val) < 1e-9:
                return True
        except ValueError:
            continue
    return False


def _looks_like_prints(text: str) -> bool:
    low = text.lower()
    if any(k in low for k in ("last", "price", "rsi", "volume", "24h")):
        return bool(_CLAIM_NUM.search(text))
    return False
