# AI research sources

Allowlisted public research for the web specialist. Market **prices** still come only from Swyngora HTTP tools.

## Layout

| File | Role |
|------|------|
| `allowlist.py` | Host reliability (`primary` / `newsroom` / `weak`) and citation filter |
| `identity.py` | Ticker → Wikipedia title + venue hint (Nasdaq / BIST / crypto) |
| `feeds.py` | Publisher RSS (CoinDesk, The Block, Decrypt, Binance) |
| `filings.py` | SEC EDGAR (US) and KAP (BIST) — fail soft |

## Config

No API keys. SEC requires a descriptive User-Agent (set in code).

## Tests

`cd ai && pytest tests/test_sources.py tests/test_web_search.py tests/test_grounding.py`
