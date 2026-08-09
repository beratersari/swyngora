"""Extract clickable source references from web/X tool text."""

from __future__ import annotations

import re
from dataclasses import asdict, dataclass
from urllib.parse import urlparse

_URL_RE = re.compile(r"https?://[^\s\)\]\>\"']+", re.I)
_LABELED_URL_RE = re.compile(r"(?im)^\s*(?:URL|Link)\s*:\s*(https?://\S+)")
_MD_LINK_RE = re.compile(r"\[([^\]]+)\]\((https?://[^)\s]+)\)")
_HN_RE = re.compile(r"(?im)^\s*(?:HN:|\d+\.\s*\[Hacker News\])\s*(.+?)\s+(https?://\S+)")


@dataclass(frozen=True)
class Reference:
    title: str
    url: str
    source: str
    snippet: str = ""

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


def _norm_url(raw: str) -> str:
    u = raw.strip().rstrip(".,;)]}")
    if len(u) > 500:
        u = u[:500]
    return u


def _host(url: str) -> str:
    try:
        return (urlparse(url).hostname or "").lower().removeprefix("www.")
    except Exception:
        return ""


def classify_source(url: str, hint: str = "") -> str:
    h = _host(url)
    blob = f"{hint} {h}".lower()
    if any(x in h for x in ("x.com", "twitter.com", "nitter.", "stocktwits.com")):
        return "x"
    if "news.ycombinator.com" in h or "hacker news" in blob or blob.strip().startswith("hn"):
        return "hn"
    if any(x in blob for x in ("web_news", "news")) and "hacker" not in blob:
        if any(
            n in h
            for n in ("coindesk", "reuters", "bloomberg", "wsj", "ft.com", "theblock", "decrypt")
        ):
            return "news"
    if hint == "news" or "web_news" in blob:
        return "news"
    return "web"


def _title_from_context(text: str, url: str) -> str:
    idx = text.find(url)
    window = text[max(0, idx - 180) : idx] if idx >= 0 else text[:180]
    lines = [ln.strip(" -\t") for ln in window.splitlines() if ln.strip()]
    for ln in reversed(lines):
        cleaned = re.sub(r"^\d+\.\s*", "", ln)
        cleaned = re.sub(r"^\[.*?\]\s*", "", cleaned)
        cleaned = re.sub(r"^URL\s*:\s*", "", cleaned, flags=re.I)
        if cleaned and "http" not in cleaned.lower() and len(cleaned) > 3:
            return cleaned[:160]
    host = _host(url)
    return host or url[:80]


def extract_references(*blobs: str, limit: int = 12) -> list[Reference]:
    """Parse URLs from specialist/tool dumps. First occurrence wins."""
    seen: set[str] = set()
    out: list[Reference] = []

    def add(url: str, title: str, source: str, snippet: str = "") -> None:
        url = _norm_url(url)
        if not url.startswith("http") or url in seen:
            return
        seen.add(url)
        title = (title or "").strip() or _host(url) or url
        out.append(
            Reference(title=title[:160], url=url, source=source, snippet=(snippet or "")[:240])
        )

    for blob in blobs:
        if not blob or blob.startswith("ERROR"):
            continue
        hint = ""
        low = blob[:80].lower()
        if "hacker news" in low or "### hacker news" in blob.lower():
            hint = "hn"
        elif "stocktwits" in low or "### stocktwits" in blob.lower():
            hint = "x"
        elif "web_news" in low or "news endpoint" in low:
            hint = "news"

        for m in _MD_LINK_RE.finditer(blob):
            add(m.group(2), m.group(1), classify_source(m.group(2), hint))

        for m in _HN_RE.finditer(blob):
            add(m.group(2), m.group(1), "hn")

        for m in _LABELED_URL_RE.finditer(blob):
            url = m.group(1)
            add(url, _title_from_context(blob, url), classify_source(url, hint))

        for m in _URL_RE.finditer(blob):
            url = m.group(0)
            add(url, _title_from_context(blob, url), classify_source(url, hint))

        if len(out) >= limit:
            break

    return out[:limit]
