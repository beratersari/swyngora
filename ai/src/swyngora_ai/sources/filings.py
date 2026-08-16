"""Free filings: SEC EDGAR (US) and KAP (BIST). Fail soft."""

from __future__ import annotations

import urllib.parse
from datetime import UTC, datetime, timedelta
from typing import Any

import httpx

_UA = "SwyngoraAI/0.1 (research; +https://github.com/beratersari/swyngora)"


def _http_json(url: str, timeout: float = 12.0) -> Any:
    with httpx.Client(timeout=timeout, headers={"User-Agent": _UA}) as client:
        r = client.get(url, headers={"Accept": "application/json"})
        r.raise_for_status()
        return r.json()


def _http_bytes(url: str, timeout: float = 12.0) -> bytes:
    with httpx.Client(timeout=timeout, headers={"User-Agent": _UA}) as client:
        r = client.get(url, headers={"Accept": "*/*"})
        r.raise_for_status()
        return r.content


def edgar_recent(ticker: str, limit: int = 5) -> str:
    """Recent 8-K / 10-Q / 10-K for a US ticker via data.sec.gov."""
    ticker = (ticker or "").strip().upper()
    if not ticker:
        return ""
    try:
        mapping = _http_json("https://www.sec.gov/files/company_tickers.json", timeout=12)
    except Exception as e:  # noqa: BLE001
        return f"(EDGAR tickers: {e})"
    cik = ""
    name = ticker
    if isinstance(mapping, dict):
        for row in mapping.values():
            if not isinstance(row, dict):
                continue
            if str(row.get("ticker") or "").upper() == ticker:
                cik = str(row.get("cik_str") or "").zfill(10)
                name = str(row.get("title") or ticker)
                break
    if not cik:
        return f"(EDGAR: no CIK for {ticker})"
    try:
        data = _http_json(f"https://data.sec.gov/submissions/CIK{cik}.json", timeout=12)
    except Exception as e:  # noqa: BLE001
        return f"(EDGAR submissions: {e})"
    recent = (data.get("filings") or {}).get("recent") or {}
    forms = recent.get("form") or []
    dates = recent.get("filingDate") or []
    acc = recent.get("accessionNumber") or []
    primary = recent.get("primaryDocument") or []
    lines: list[str] = []
    wanted = {"8-K", "10-Q", "10-K", "6-K", "20-F"}
    for i, form in enumerate(forms):
        if form not in wanted:
            continue
        filed = dates[i] if i < len(dates) else ""
        accession = (acc[i] if i < len(acc) else "").replace("-", "")
        doc = primary[i] if i < len(primary) else ""
        href = ""
        if accession and doc:
            href = f"https://www.sec.gov/Archives/edgar/data/{int(cik)}/{accession}/{doc}"
        lines.append(f"{len(lines) + 1}. [SEC {form}] {name} ({filed})\n   URL: {href}")
        if len(lines) >= limit:
            break
    if not lines:
        return f"(EDGAR: no recent 8-K/10-Q/10-K for {ticker})"
    return "\n".join(lines)


def kap_recent(query: str, limit: int = 5) -> str:
    """Best-effort KAP disclosure search. Fail soft if the public API shape changes."""
    q = (query or "").strip()
    if not q:
        return ""
    # Public KAP disclosure list (HTML/JSON varies). Try the English disclosure index RSS-like page.
    since = (datetime.now(UTC) - timedelta(days=30)).strftime("%Y-%m-%d")
    encoded = urllib.parse.quote(q)
    url = f"https://www.kap.org.tr/en/api/disclosureList?fromDate={since}&search={encoded}"
    try:
        data = _http_json(url, timeout=12)
    except httpx.HTTPStatusError as e:
        return f"(KAP: HTTP {e.response.status_code})"
    except Exception as e:  # noqa: BLE001
        # Fallback: KAP search landing (still a primary host for the desk to open).
        return (
            f"1. [KAP] Search disclosures for {q} (since {since})\n"
            f"   URL: https://www.kap.org.tr/en/bildirim-sorgu\n"
            f"   (direct API unavailable: {e})"
        )
    rows: list[Any]
    if isinstance(data, list):
        rows = data
    elif isinstance(data, dict):
        rows = list(data.get("data") or data.get("items") or data.get("disclosures") or [])
    else:
        rows = []
    lines: list[str] = []
    for row in rows[:limit]:
        if not isinstance(row, dict):
            continue
        title = str(row.get("title") or row.get("header") or row.get("summary") or q)
        hid = row.get("disclosureIndex") or row.get("id") or ""
        href = (
            f"https://www.kap.org.tr/en/Bildirim/{hid}"
            if hid
            else "https://www.kap.org.tr/en/bildirim-sorgu"
        )
        pub = str(row.get("publishDate") or row.get("date") or "")
        lines.append(f"{len(lines) + 1}. [KAP] {title} ({pub})\n   URL: {href}")
    if not lines:
        return (
            f"1. [KAP] Search disclosures for {q}\n   URL: https://www.kap.org.tr/en/bildirim-sorgu"
        )
    return "\n".join(lines)
