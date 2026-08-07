"""Social / X-adjacent free signal search.

Official X API is paid — we never use it (AGENTS.md §6.5).
DuckDuckGo ``site:x.com`` / ``site:twitter.com`` returns empty in practice (X is not
publicly indexed). This tool therefore combines:

1. **StockTwits** public symbol streams (free JSON) — primary live social chatter
2. **Hacker News** Algolia (free) — secondary discussion
3. **DuckDuckGo** plain-text search (no site: filter) — weak web/social roundups

Results are always weak signals, not market truth.
"""

from __future__ import annotations

import json
import re
import urllib.error
import urllib.parse
import urllib.request
from typing import Any

from langchain_core.tools import StructuredTool
from pydantic import BaseModel, Field

_UA = "SwyngoraAI/0.1 (+https://github.com/beratersari/swyngora; research bot)"

# Common crypto aliases → StockTwits-style base ticker.
_ALIASES: dict[str, str] = {
    "BITCOIN": "BTC",
    "BTCUSDT": "BTC",
    "BTCUSD": "BTC",
    "ETHEREUM": "ETH",
    "ETHUSDT": "ETH",
    "ETHUSD": "ETH",
    "SOLANA": "SOL",
    "SOLUSDT": "SOL",
    "RIPPLE": "XRP",
    "XRPUSDT": "XRP",
    "DOGECOIN": "DOGE",
    "DOGEUSDT": "DOGE",
    "CARDANO": "ADA",
    "ADAUSDT": "ADA",
    "JUVENTUS": "JUV",
    "JUVUSDT": "JUV",
}


class XSearchInput(BaseModel):
    query: str = Field(description="Topic, ticker, or cashtag e.g. BTC OR bitcoin")
    max_results: int = Field(default=8, ge=1, le=15)


def _http_json(url: str, timeout: float = 12.0) -> dict[str, Any]:
    req = urllib.request.Request(url, headers={"User-Agent": _UA, "Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode("utf-8", errors="replace"))


def _guess_symbols(query: str) -> list[str]:
    """Extract likely tickers from free text (BTC, $ETH, JUVUSDT, bitcoin, …)."""
    found: list[str] = []
    upper = query.upper()

    for m in re.finditer(r"\$([A-Z]{2,10})", upper):
        found.append(m.group(1))

    for raw in re.findall(r"[A-Za-z]{2,12}", query):
        w = raw.upper()
        if w in _ALIASES:
            found.append(_ALIASES[w])
        elif re.fullmatch(r"[A-Z]{2,6}(USDT|USD)", w):
            found.append(re.sub(r"(USDT|USD)$", "", w))
        elif re.fullmatch(r"[A-Z]{2,6}", w) and w not in {
            "OR",
            "AND",
            "THE",
            "FOR",
            "WHAT",
            "WILL",
            "ABOUT",
            "TWITTER",
            "CRYPTO",
            "PRICE",
            "FROM",
            "WITH",
            "THIS",
            "THAT",
            "HAVE",
            "BEEN",
        }:
            # short all-caps-looking tokens already uppercased from free text words
            if raw.isupper() or raw.islower() and len(raw) <= 5:
                # only keep well-known-ish 2–5 letter tickers from mixed query words
                if len(w) <= 5 and w not in {"OLUR", "NE", "DIYOLAR", "HAKKINDA"}:
                    # Turkish/common stopwords skipped above; still filter length
                    if w in _ALIASES.values() or w in {
                        "BTC",
                        "ETH",
                        "SOL",
                        "XRP",
                        "DOGE",
                        "ADA",
                        "JUV",
                        "BNB",
                        "AVAX",
                        "DOT",
                        "LINK",
                        "MATIC",
                        "PEPE",
                        "SHIB",
                        "TON",
                        "SUI",
                        "APT",
                        "ARB",
                        "OP",
                    }:
                        found.append(w)

    # preserve order, unique
    out: list[str] = []
    seen: set[str] = set()
    for s in found:
        if s not in seen:
            seen.add(s)
            out.append(s)
    return out[:5]


def _stocktwits_symbol_candidates(base: str) -> list[str]:
    """StockTwits uses BTC.X for crypto; also try bare equity-style codes."""
    base = base.upper().strip(".")
    if base.endswith(".X"):
        return [base, base[:-2]]
    return [f"{base}.X", base]


def _fetch_stocktwits(symbol: str, limit: int) -> list[str]:
    url = f"https://api.stocktwits.com/api/2/streams/symbol/{urllib.parse.quote(symbol)}.json"
    try:
        data = _http_json(url)
    except urllib.error.HTTPError as e:
        if e.code == 404:
            return []
        return [f"(StockTwits {symbol}: HTTP {e.code})"]
    except Exception as e:  # noqa: BLE001
        return [f"(StockTwits {symbol}: {e})"]

    lines: list[str] = []
    sym_meta = data.get("symbol") or {}
    title = sym_meta.get("title") or sym_meta.get("symbol") or symbol
    for msg in (data.get("messages") or [])[:limit]:
        body = (msg.get("body") or "").replace("\n", " ").strip()
        if not body:
            continue
        user = ((msg.get("user") or {}).get("username")) or "?"
        created = msg.get("created_at") or ""
        likes = ((msg.get("likes") or {}).get("total")) or 0
        if len(body) > 220:
            body = body[:217] + "…"
        lines.append(
            f"@{user} ({created}, likes={likes}) [{title}]: {body}\n"
            f"   URL: https://stocktwits.com/symbol/{urllib.parse.quote(symbol)}"
        )
    return lines


def _fetch_hn(query: str, limit: int) -> list[str]:
    q = urllib.parse.quote(query)
    url = (
        "https://hn.algolia.com/api/v1/search_by_date"
        f"?query={q}&tags=story&hitsPerPage={limit}"
    )
    try:
        data = _http_json(url)
    except Exception as e:  # noqa: BLE001
        return [f"(Hacker News: {e})"]

    lines: list[str] = []
    for hit in data.get("hits") or []:
        title = hit.get("title") or hit.get("story_title") or ""
        if not title:
            continue
        url_h = hit.get("url") or (
            f"https://news.ycombinator.com/item?id={hit.get('objectID')}"
        )
        pts = hit.get("points")
        lines.append(f"HN: {title} (pts={pts}) {url_h}")
    return lines


def _fetch_ddg_social(query: str, limit: int) -> list[str]:
    """Plain DDG (no site:x.com — that filter always returns empty)."""
    try:
        from duckduckgo_search import DDGS
    except ImportError:
        return ["(DuckDuckGo package not installed)"]

    # Prefer social-ish wording without site: filters that zero-out results.
    variants = [
        f"{query} twitter OR x.com OR stocktwits sentiment",
        f"{query} crypto social discussion",
        query,
    ]
    lines: list[str] = []
    errors: list[str] = []
    for q in variants:
        if len(lines) >= limit:
            break
        try:
            with DDGS() as ddgs:
                results = list(ddgs.text(q, max_results=limit, backend="auto"))
        except Exception as e:  # noqa: BLE001
            errors.append(f"{type(e).__name__}: {e}")
            continue
        for r in results:
            title = r.get("title") or ""
            href = r.get("href") or r.get("link") or ""
            body = r.get("body") or r.get("snippet") or ""
            if not title and not body:
                continue
            if len(body) > 160:
                body = body[:157] + "…"
            lines.append(f"{title}\n   {href}\n   {body}")
            if len(lines) >= limit:
                break
    if not lines and errors:
        return [f"(DuckDuckGo failed: {errors[0]})"]
    return lines


def _x_search(query: str, max_results: int = 8) -> str:
    query = (query or "").strip()
    if not query:
        return "ERROR x_search: empty query"

    header = [
        "NOTE: Free social signals only — NOT the official X API.",
        "DuckDuckGo site:x.com indexing is empty in practice; primary source is StockTwits (+ HN/web).",
        "Treat as weak, incomplete chatter — never sole evidence for trades.",
        "",
    ]

    symbols = _guess_symbols(query)
    if not symbols:
        # default popular crypto if query is vague social question
        if re.search(r"\b(btc|bitcoin)\b", query, re.I):
            symbols = ["BTC"]
        elif re.search(r"\b(eth|ethereum)\b", query, re.I):
            symbols = ["ETH"]

    sections: list[str] = []
    st_lines: list[str] = []
    per_sym = max(3, max_results // max(1, len(symbols) or 1))

    for base in symbols or []:
        got = False
        for cand in _stocktwits_symbol_candidates(base):
            rows = _fetch_stocktwits(cand, per_sym)
            # skip soft errors that are single parenthetical
            real = [r for r in rows if not r.startswith("(StockTwits")]
            errs = [r for r in rows if r.startswith("(StockTwits")]
            if real:
                st_lines.append(f"— StockTwits {cand} —")
                st_lines.extend(real)
                got = True
                break
            if errs and not got:
                st_lines.extend(errs)
        if got and len(st_lines) >= max_results + 5:
            break

    if st_lines:
        sections.append("### StockTwits (live social)\n" + "\n".join(st_lines[: max_results + 8]))

    hn = _fetch_hn(query, min(5, max_results))
    hn_real = [h for h in hn if h.startswith("HN:")]
    if hn_real:
        sections.append("### Hacker News\n" + "\n".join(hn_real))

    ddg = _fetch_ddg_social(query, min(5, max_results))
    ddg_real = [d for d in ddg if not d.startswith("(DuckDuckGo")]
    if ddg_real:
        sections.append("### Web social roundups (DDG)\n" + "\n".join(ddg_real[:5]))
    elif ddg:
        sections.append("### Web social roundups (DDG)\n" + ddg[0])

    if not sections:
        return "\n".join(
            header
            + [
                "No social results returned.",
                f"Tried symbols={symbols or '[]'} via StockTwits, HN, DuckDuckGo.",
                "Retry with an explicit ticker (e.g. BTC, ETH, JUV).",
            ]
        )

    return "\n".join(header + sections)


def build_x_tools() -> list[StructuredTool]:
    return [
        StructuredTool.from_function(
            _x_search,
            name="x_search",
            description=(
                "Search free social chatter for crypto (StockTwits live streams, HN, web). "
                "Not official X API; weak signal only. Prefer explicit tickers (BTC, ETH, JUV)."
            ),
            args_schema=XSearchInput,
        ),
    ]
