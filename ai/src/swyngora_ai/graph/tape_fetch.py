"""Deterministic tape pull — numbers from tools, not from the LLM."""

from __future__ import annotations

from swyngora_ai.config import Settings
from swyngora_ai.progress import emit
from swyngora_ai.sources.identity import Instrument, find_instruments
from swyngora_ai.tools.market_http import build_market_tools


def prefetch_tape(
    settings: Settings,
    message: str,
    *,
    fallback_symbol: str = "",
    fallback_exchange: str = "",
) -> list[str]:
    """Call get_ticker (+ 1h indicators) for every recognized instrument."""
    insts = find_instruments(message)
    if not insts and fallback_symbol:
        insts = [
            Instrument(
                symbol=fallback_symbol,
                exchange=fallback_exchange or "binance",
                kind="unknown",
            )
        ]
    if not insts:
        return []

    by_name = {t.name: t for t in build_market_tools(settings, pack="tape")}
    ticker = by_name.get("get_ticker")
    indicators = by_name.get("get_indicators")
    blobs: list[str] = []
    for inst in insts[:3]:
        emit("status", f"Fetching tape {inst.symbol} {inst.exchange}…")
        emit("tool", f"prefetch → get_ticker(symbol={inst.symbol}, exchange={inst.exchange})")
        if ticker is None:
            continue
        raw = ticker.invoke({"symbol": inst.symbol, "exchange": inst.exchange})
        blobs.append(raw if isinstance(raw, str) else str(raw))
        if indicators is None:
            continue
        try:
            ind = indicators.invoke(
                {
                    "symbol": inst.symbol,
                    "exchange": inst.exchange,
                    "interval": "1h",
                }
            )
            blobs.append(ind if isinstance(ind, str) else str(ind))
        except Exception as e:  # noqa: BLE001
            emit("tool_error", f"get_indicators failed: {e}")
    return blobs
