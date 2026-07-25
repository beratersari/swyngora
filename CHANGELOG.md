# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
