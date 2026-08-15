"""Publisher allowlist and citation reliability."""

from __future__ import annotations

from urllib.parse import urlparse

from swyngora_ai.references import Reference

PRIMARY_HOSTS: frozenset[str] = frozenset(
    {
        "sec.gov",
        "data.sec.gov",
        "efts.sec.gov",
        "kap.org.tr",
        "www.kap.org.tr",
        "binance.com",
        "www.binance.com",
        "coinbase.com",
        "www.coinbase.com",
        "bybit.com",
        "www.bybit.com",
        "federalreserve.gov",
        "www.federalreserve.gov",
        "fred.stlouisfed.org",
        "ecb.europa.eu",
        "www.ecb.europa.eu",
        "tcmb.gov.tr",
        "www.tcmb.gov.tr",
    }
)

NEWSROOM_HOSTS: frozenset[str] = frozenset(
    {
        "coindesk.com",
        "www.coindesk.com",
        "theblock.co",
        "www.theblock.co",
        "decrypt.co",
        "www.decrypt.co",
        "reuters.com",
        "www.reuters.com",
        "bloomberg.com",
        "www.bloomberg.com",
        "ft.com",
        "www.ft.com",
        "wsj.com",
        "www.wsj.com",
        "cnbc.com",
        "www.cnbc.com",
        "coingecko.com",
        "www.coingecko.com",
        "coinmarketcap.com",
        "www.coinmarketcap.com",
        "wikipedia.org",
        "en.wikipedia.org",
        "defillama.com",
        "api.llama.fi",
        "news.google.com",
    }
)

WEAK_HOSTS: frozenset[str] = frozenset(
    {
        "stocktwits.com",
        "www.stocktwits.com",
        "reddit.com",
        "www.reddit.com",
        "old.reddit.com",
        "news.ycombinator.com",
        "x.com",
        "twitter.com",
        "www.twitter.com",
    }
)


def _host(url: str) -> str:
    try:
        return (urlparse(url).hostname or "").lower().removeprefix("www.")
    except ValueError:
        return ""


def classify_reliability(url: str) -> str:
    """Return primary | newsroom | weak | unknown."""
    h = _host(url)
    full = (urlparse(url).hostname or "").lower()
    if h in {x.removeprefix("www.") for x in PRIMARY_HOSTS} or full in PRIMARY_HOSTS:
        return "primary"
    if h in {x.removeprefix("www.") for x in NEWSROOM_HOSTS} or full in NEWSROOM_HOSTS:
        return "newsroom"
    if h in {x.removeprefix("www.") for x in WEAK_HOSTS} or full in WEAK_HOSTS:
        return "weak"
    return "unknown"


def host_allowed(url: str, *, extra_hosts: set[str] | None = None) -> bool:
    """True if the URL may be shown as a citation."""
    rel = classify_reliability(url)
    if rel in {"primary", "newsroom", "weak"}:
        return True
    if extra_hosts:
        h = _host(url)
        if h in extra_hosts or any(h.endswith(f".{e}") for e in extra_hosts):
            return True
    return False


def filter_references(
    refs: list[Reference],
    *,
    extra_hosts: set[str] | None = None,
) -> list[Reference]:
    """Drop unknown hosts; stamp reliability on keepers."""
    out: list[Reference] = []
    for r in refs:
        if not host_allowed(r.url, extra_hosts=extra_hosts):
            continue
        rel = classify_reliability(r.url)
        out.append(
            Reference(
                title=r.title,
                url=r.url,
                source=r.source,
                snippet=r.snippet,
                reliability=rel,
            )
        )
    return out
