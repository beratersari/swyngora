"""Free web search tools (DuckDuckGo — no paid API tier)."""

from __future__ import annotations

from langchain_core.tools import StructuredTool
from pydantic import BaseModel, Field


class WebSearchInput(BaseModel):
    query: str = Field(description="Search query")
    max_results: int = Field(default=5, ge=1, le=10)


def _ddgs():
    try:
        from duckduckgo_search import DDGS

        return DDGS
    except ImportError:
        return None


def _search(query: str, max_results: int = 5) -> str:
    DDGS = _ddgs()
    if DDGS is None:
        return "ERROR: duckduckgo-search not installed"

    lines: list[str] = []
    last_err: str | None = None
    # Try backends — auto often rate-limits; html/lite can still work.
    for backend in ("auto", "html", "lite"):
        try:
            with DDGS() as ddgs:
                results = list(ddgs.text(query, max_results=max_results, backend=backend))
            if results:
                for i, r in enumerate(results, 1):
                    title = r.get("title") or ""
                    href = r.get("href") or r.get("link") or ""
                    body = r.get("body") or r.get("snippet") or ""
                    lines.append(f"{i}. {title}\n   URL: {href}\n   {body}")
                return "\n".join(lines)
            last_err = f"empty results (backend={backend})"
        except Exception as e:  # noqa: BLE001 — surface to agent
            last_err = f"{type(e).__name__}: {e}"
            # rate-limit → try next backend
            continue

    if last_err and "Ratelimit" in last_err:
        return f"ERROR web_search: rate-limited by DuckDuckGo ({last_err}). Retry shortly."
    if last_err:
        return f"ERROR web_search: {last_err}"
    return "No results."


def _hn_news(query: str, max_results: int = 5) -> str:
    """Free Hacker News Algolia fallback when DuckDuckGo news is blocked."""
    import json
    import urllib.parse
    import urllib.request

    q = urllib.parse.quote(query)
    url = (
        "https://hn.algolia.com/api/v1/search_by_date"
        f"?query={q}&tags=story&hitsPerPage={max_results}"
    )
    req = urllib.request.Request(
        url,
        headers={"User-Agent": "SwyngoraAI/0.1 (news fallback)", "Accept": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=12) as resp:
        data = json.loads(resp.read().decode("utf-8", errors="replace"))
    lines: list[str] = []
    for i, hit in enumerate(data.get("hits") or [], 1):
        title = hit.get("title") or hit.get("story_title") or ""
        if not title:
            continue
        link = hit.get("url") or f"https://news.ycombinator.com/item?id={hit.get('objectID')}"
        pts = hit.get("points")
        created = hit.get("created_at") or ""
        lines.append(f"{i}. [Hacker News] {title} ({created}, pts={pts})\n   {link}")
    if not lines:
        return ""
    return "NOTE: DuckDuckGo news unavailable; Hacker News fallback:\n" + "\n".join(lines)


def _news(query: str, max_results: int = 5) -> str:
    DDGS = _ddgs()
    last_err: str | None = None
    results: list = []
    if DDGS is not None:
        try:
            with DDGS() as ddgs:
                results = list(ddgs.news(query, max_results=max_results))
        except Exception as e:  # noqa: BLE001
            last_err = f"{type(e).__name__}: {e}"
            results = []

    if results:
        lines: list[str] = []
        for i, r in enumerate(results, 1):
            title = r.get("title") or ""
            url = r.get("url") or r.get("href") or ""
            source = r.get("source") or ""
            date = r.get("date") or ""
            body = r.get("body") or ""
            lines.append(f"{i}. [{source}] {title} ({date})\n   {url}\n   {body}")
        return "\n".join(lines)

    # Fallbacks: DDG text → Hacker News (free, no API key).
    text = _search(f"{query} news", max_results=max_results) if DDGS is not None else "No results."
    if not text.startswith("ERROR") and text != "No results.":
        note = "NOTE: news endpoint empty/failed; web text fallback:\n"
        if last_err:
            note = f"NOTE: news endpoint failed ({last_err}); web text fallback:\n"
        return note + text

    try:
        hn = _hn_news(query, max_results=max_results)
        if hn:
            return hn
    except Exception as e:  # noqa: BLE001
        last_err = f"{last_err}; HN: {e}" if last_err else f"HN: {e}"

    if last_err and "Ratelimit" in last_err:
        return f"ERROR web_news: rate-limited ({last_err}). Retry shortly."
    if last_err:
        return f"ERROR web_news: {last_err}"
    return "No news results."


def build_web_tools() -> list[StructuredTool]:
    return [
        StructuredTool.from_function(
            _search,
            name="web_search",
            description="Search the public web (news, docs, general info). Free DuckDuckGo.",
            args_schema=WebSearchInput,
        ),
        StructuredTool.from_function(
            _news,
            name="web_news",
            description="Search recent news articles for a topic.",
            args_schema=WebSearchInput,
        ),
    ]
