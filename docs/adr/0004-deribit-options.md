# ADR 0004: Deribit public API for BTC/ETH options

## Status

Accepted — 2026-09-05

## Context

The desk needs CoinGlass-like options data (open interest, volume, put/call,
implied vol, expiry mix) for BTC and ETH. CoinGlass and similar vendors are
paid APIs. AGENTS.md forbids paid pricing-tier data sources without an ADR
and team approval.

Deribit is the primary listed-options venue for BTC/ETH and publishes
book summaries, DVOL, and the USD index on public REST with no API key.

## Decision

1. Use **Deribit public REST** for listed inverse options on BTC and ETH.
2. Do **not** integrate CoinGlass or other paid options vendors.
3. Do **not** treat this as futures open interest (Binance/Bybit remain
   `/market/open-interest`).
4. Cache responses (~30s). Isolate parsing in `adapter/deribit`.

## Consequences

- Coverage is Deribit-listed contracts only (the bulk of BTC/ETH options).
- Inverse-option OI is in the base coin, not USD contracts.
- Adding another venue later stays a separate port implementation.
