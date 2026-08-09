"""Free public research tools — no paid search API.

DuckDuckGo is optional and often rate-limits/hangs. Primary sources:
Wikipedia REST, Google News RSS, CoinGecko search, Hacker News Algolia.
"""

from __future__ import annotations

import json
import re
import urllib.error
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET
from concurrent.futures import ThreadPoolExecutor
from concurrent.futures import TimeoutError as FutTimeout
from typing import Any

from langchain_core.tools import StructuredTool
from pydantic import BaseModel, Field

_UA = "SwyngoraAI/0.1 (+https://github.com/beratersari/swyngora; research bot)"
_DDG_TIMEOUT_SEC = 8.0


class WebSearchInput(BaseModel):
    query: str = Field(description="Search query")
    max_results: int = Field(default=5, ge=1, le=10)


class WebResearchInput(BaseModel):
    topic: str = Field(
        description="Coin, ticker, or project to research (e.g. BTC, Solana, JUVUSDT)"
    )
    max_results: int = Field(default=8, ge=3, le=12)


def _ddgs():
    try:
        from duckduckgo_search import DDGS

        return DDGS
    except ImportError:
        return None


def _http_bytes(url: str, timeout: float = 12.0) -> bytes:
    req = urllib.request.Request(url, headers={"User-Agent": _UA, "Accept": "*/*"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.read()


def _http_json(url: str, timeout: float = 12.0) -> Any:
    req = urllib.request.Request(
        url,
        headers={"User-Agent": _UA, "Accept": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode("utf-8", errors="replace"))


def _ddgs_text_once(query: str, max_results: int, backend: str) -> list[dict]:
    DDGS = _ddgs()
    if DDGS is None:
        raise RuntimeError("duckduckgo-search not installed")
    with DDGS() as ddgs:
        return list(ddgs.text(query, max_results=max_results, backend=backend))


def _search(query: str, max_results: int = 5) -> str:
    """DuckDuckGo text search with a hard timeout (often blocked/slow)."""
    DDGS = _ddgs()
    if DDGS is None:
        return "ERROR: duckduckgo-search not installed"

    last_err: str | None = None
    for backend in ("lite", "html", "auto"):
        try:
            with ThreadPoolExecutor(max_workers=1) as pool:
                fut = pool.submit(_ddgs_text_once, query, max_results, backend)
                results = fut.result(timeout=_DDG_TIMEOUT_SEC)
        except FutTimeout:
            last_err = f"timeout after {_DDG_TIMEOUT_SEC}s (backend={backend})"
            continue
        except Exception as e:  # noqa: BLE001
            last_err = f"{type(e).__name__}: {e}"
            continue
        if results:
            lines: list[str] = []
            for i, r in enumerate(results, 1):
                title = r.get("title") or ""
                href = r.get("href") or r.get("link") or ""
                body = r.get("body") or r.get("snippet") or ""
                lines.append(f"{i}. {title}\n   URL: {href}\n   {body}")
            return "\n".join(lines)
        last_err = f"empty results (backend={backend})"

    if last_err and "Ratelimit" in last_err:
        return f"ERROR web_search: rate-limited by DuckDuckGo ({last_err})."
    if last_err:
        return f"ERROR web_search: {last_err}"
    return "No results."


def _hn_news(query: str, max_results: int = 5) -> str:
    """Hacker News Algolia — free, no key."""
    q = urllib.parse.quote(query)
    url = f"https://hn.algolia.com/api/v1/search?query={q}&tags=story&hitsPerPage={max_results}"
    data = _http_json(url, timeout=12)
    lines: list[str] = []
    for i, hit in enumerate(data.get("hits") or [], 1):
        title = hit.get("title") or hit.get("story_title") or ""
        if not title:
            continue
        link = hit.get("url") or f"https://news.ycombinator.com/item?id={hit.get('objectID')}"
        pts = hit.get("points")
        created = hit.get("created_at") or ""
        lines.append(f"{i}. [Hacker News] {title} ({created}, pts={pts})\n   URL: {link}")
    if not lines:
        return ""
    return "\n".join(lines)


def _wikipedia(topic: str) -> str:
    slug = urllib.parse.quote(topic.replace(" ", "_"))
    try:
        data = _http_json(f"https://en.wikipedia.org/api/rest_v1/page/summary/{slug}", timeout=10)
    except urllib.error.HTTPError as e:
        if e.code != 404:
            return f"(Wikipedia: HTTP {e.code})"
        try:
            opensearch = _http_json(
                "https://en.wikipedia.org/w/api.php?action=opensearch"
                f"&search={urllib.parse.quote(topic)}&limit=1&namespace=0&format=json",
                timeout=10,
            )
            if isinstance(opensearch, list) and len(opensearch) >= 4 and opensearch[1]:
                title = opensearch[1][0]
                page = opensearch[3][0] if opensearch[3] else ""
                return f"1. [Wikipedia] {title}\n   URL: {page}"
        except Exception as e2:  # noqa: BLE001
            return f"(Wikipedia: {e}; retry {e2})"
        return ""
    except Exception as e:  # noqa: BLE001
        return f"(Wikipedia: {e})"

    title = data.get("title") or topic
    extract = (data.get("extract") or "")[:280]
    desktop = (data.get("content_urls") or {}).get("desktop") or {}
    url = (
        desktop.get("page")
        or f"https://en.wikipedia.org/wiki/{urllib.parse.quote(title.replace(' ', '_'))}"
    )
    if data.get("type") == "disambiguation":
        return f"1. [Wikipedia] {title} (disambiguation)\n   URL: {url}\n   {extract}"
    return f"1. [Wikipedia] {title}\n   URL: {url}\n   {extract}"


def _google_news_rss(query: str, max_results: int = 6) -> str:
    q = urllib.parse.quote(query)
    url = f"https://news.google.com/rss/search?q={q}&hl=en-US&gl=US&ceid=US:en"
    try:
        raw = _http_bytes(url, timeout=12)
    except Exception as e:  # noqa: BLE001
        return f"(Google News RSS: {e})"
    try:
        root = ET.fromstring(raw)
    except ET.ParseError as e:
        return f"(Google News RSS parse: {e})"
    items = root.findall("./channel/item")
    lines: list[str] = []
    for i, item in enumerate(items[:max_results], 1):
        title = (item.findtext("title") or "").strip()
        link = (item.findtext("link") or "").strip()
        pub = (item.findtext("pubDate") or "").strip()
        source_el = item.find("source")
        source = (source_el.text or "").strip() if source_el is not None else ""
        if not title or not link:
            continue
        lines.append(f"{i}. [{source or 'News'}] {title} ({pub})\n   URL: {link}")
    return "\n".join(lines)


def _coingecko(topic: str) -> str:
    q = urllib.parse.quote(topic)
    try:
        data = _http_json(f"https://api.coingecko.com/api/v3/search?query={q}", timeout=10)
    except Exception as e:  # noqa: BLE001
        return f"(CoinGecko: {e})"
    coins = (data.get("coins") or [])[:2]
    if not coins:
        return ""
    lines: list[str] = []
    for i, c in enumerate(coins, 1):
        cid = c.get("id") or ""
        name = c.get("name") or cid
        sym = c.get("symbol") or ""
        href = f"https://www.coingecko.com/en/coins/{cid}" if cid else ""
        mcap = c.get("market_cap_rank")
        rank = f" rank #{mcap}" if mcap else ""
        lines.append(f"{i}. [CoinGecko] {name} ({sym}){rank}\n   URL: {href}")
    top = coins[0].get("id") or ""
    if top:
        slug = re.sub(r"[^a-z0-9-]", "", (coins[0].get("name") or top).lower().replace(" ", "-"))
        if slug:
            lines.append(
                f"{len(lines) + 1}. [CoinMarketCap] {coins[0].get('name')}\n"
                f"   URL: https://coinmarketcap.com/currencies/{slug}/"
            )
    return "\n".join(lines)


def _news(query: str, max_results: int = 5) -> str:
    """Prefer Google News RSS + HN; DuckDuckGo last (often blocked)."""
    rss = _google_news_rss(query, max_results=max_results)
    if rss and not rss.startswith("("):
        return rss
    try:
        hn = _hn_news(query, max_results=max_results)
        if hn:
            note = "NOTE: Google News empty/failed; Hacker News fallback:\n"
            if rss.startswith("("):
                note = f"NOTE: Google News failed ({rss}); Hacker News fallback:\n"
            return note + hn
    except Exception as e:  # noqa: BLE001
        rss = f"{rss}; HN: {e}" if rss else f"HN: {e}"

    ddg = _search(f"{query} news", max_results=max_results)
    if ddg and not ddg.startswith("ERROR") and ddg != "No results.":
        return "NOTE: RSS/HN thin; DuckDuckGo text fallback:\n" + ddg
    if rss and rss.startswith("("):
        return f"ERROR web_news: {rss}"
    return "No news results."


def _research(topic: str, max_results: int = 8) -> str:
    """Fan-out reliable free sources first; one timed DDG pass last."""
    topic = (topic or "").strip()
    if not topic:
        return "ERROR web_research: empty topic"

    chunks: list[str] = [
        f"NOTE: Multi-source desk research for {topic!r} "
        "(Wikipedia, Google News RSS, CoinGecko, Hacker News). Not live prices."
    ]

    wiki = _wikipedia(_wiki_title(topic))
    if wiki and not wiki.startswith("("):
        chunks.append(f"### wikipedia\n{wiki}")

    gecko = _coingecko(topic)
    if gecko and not gecko.startswith("("):
        chunks.append(f"### market pages\n{gecko}")

    news = _google_news_rss(topic, max_results=min(6, max_results))
    if news and not news.startswith("("):
        chunks.append(f"### news\n{news}")

    try:
        hn = _hn_news(topic, max_results=min(5, max_results))
        if hn:
            chunks.append(f"### hacker news\n{hn}")
    except Exception as e:  # noqa: BLE001
        chunks.append(f"(Hacker News skipped: {e})")

    ddg = _search(f"{topic} news", max_results=3)
    if ddg and not ddg.startswith("ERROR") and ddg != "No results.":
        chunks.append(f"### web\n{ddg}")

    if len(chunks) == 1:
        return f"No public web/news hits for {topic!r}."
    return "\n\n".join(chunks)


def _wiki_title(topic: str) -> str:
    t = topic.strip()
    upper = t.upper().replace("-", "")
    aliases = {
        "BTC": "Bitcoin",
        "BTCUSDT": "Bitcoin",
        "BTCUSD": "Bitcoin",
        "ETH": "Ethereum",
        "ETHUSDT": "Ethereum",
        "SOL": "Solana (blockchain platform)",
        "SOLUSDT": "Solana (blockchain platform)",
        "XRP": "XRP",
        "DOGE": "Dogecoin",
        "ADA": "Cardano (blockchain platform)",
        "BNB": "BNB",
    }
    if upper in aliases:
        return aliases[upper]
    m = re.fullmatch(r"([A-Za-z]{2,10})(USDT|USD)", t, re.I)
    if m and m.group(1).upper() in aliases:
        return aliases[m.group(1).upper()]
    return t


def build_web_tools() -> list[StructuredTool]:
    return [
        StructuredTool.from_function(
            _search,
            name="web_search",
            description="Search the public web (DuckDuckGo; may time out). Prefer web_research for coins.",
            args_schema=WebSearchInput,
        ),
        StructuredTool.from_function(
            _news,
            name="web_news",
            description="Recent headlines via Google News RSS + Hacker News (free, no key).",
            args_schema=WebSearchInput,
        ),
        StructuredTool.from_function(
            _research,
            name="web_research",
            description=(
                "Extensive research for a coin/project: Wikipedia, Google News, CoinGecko/CMC pages, "
                "Hacker News. Use this first when the user asks about a coin."
            ),
            args_schema=WebResearchInput,
        ),
    ]
