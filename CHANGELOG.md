# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Technical indicators: **RSI** (Wilder) and **EMA** via `GET /api/v1/market/indicators`
- OpenAPI for **`POST /api/v1/market/indicators/batch`**; document `exchange` on ticker/intervals/tags
- **Watchlist** API (`/api/v1/watchlist`) + dashboard stars / filter (client id + localStorage)
- Multi-exchange spot data: **Coinbase** + **Bybit** via `?exchange=`; `GET /api/v1/market/exchanges`
- Binance product-catalog **tags** on spot markets: `tags` field, `tag`/`tags` filter (OR), `sort=tags`, and `GET /api/v1/market/tags`
- Simple frontend tag filter dropdown and Tags column

### Fixed
- Market-cap ranking: nulls last, no infinite max without price, collapse multi-quote pairs, refuse empty-supply mcap sorts
- Supply snapshot: atomic replace, USDT-pair preference, strict bapi success checks, last-good retained on failure; **retry with backoff** on failed refresh; default **48h safety TTL**
- Non-crypto filter: fail-closed on empty/soft catalog **and** when catalog is down with no last-good snapshot (spot list errors instead of listing equities)
- Candle/ticker thundering herd (singleflight); unbounded candle cache keys (range queries uncached + max entries)
- Ingress per-IP rate limit with **max bucket map**; watchlist **Add max items** + **max clients**; indicator batch **process-wide** upstream semaphore
- Sanitized public API errors; zero-duration config footguns
- Simple frontend: detail-page load race (stale symbol paint); watchlist **merge** sync (no wipe after offline adds); multi-exchange star paint vs click; tiny prices no longer format as `0`; XSS formatters; supply `asOf` cues


### Changed

- Supply (circulating / total / max) comes **only from Binance** marketing symbol list; CoinGecko adapter removed
- Supply snapshot still daily @ 03:00 UTC (+ startup); request path remains cache-only
- Max supply is null when Binance does not publish a hard cap
- Spot list and supply exclude non-crypto products (`bStocks` e.g. NVDAB/TSLAB, `tCommodities` e.g. PAXG)

### Added

- Daily supply/mcap snapshot refresh (default 03:00 UTC); user requests are cache-only
- Binance spot market list with search, metric sort, and pagination (`GET /api/v1/market/spot`)

## [0.1.0] - 2026-07-25

### Added

- Go backend (N-layered) with OpenAPI contract for market data
- Binance candlesticks (`/api/v1/market/candles`) with multi-interval support
- Binance 24h ticker including base and quote volume (`/api/v1/market/ticker/24h`)
- Asset circulating / total / max supply via free CoinGecko (`/api/v1/market/supply`)
- In-memory TTL caches with background cleanup for candles, ticker, and supply
- `simple-frontend/` static test harness (product UI reserved under `frontend/`)
- Feature docs and ADR for data-source choice

[0.1.0]: https://nova.teachx.ai/trace-analysis/swyngora/-/tags/v0.1.0
