"""Market tools via Swyngora HTTP API (mirror of Go MCP tools)."""

from __future__ import annotations

import json
from typing import Any, Optional

import httpx
from langchain_core.tools import StructuredTool
from pydantic import BaseModel, Field

from swyngora_ai.config import Settings, get_settings


class _HTTP:
    def __init__(self, base_url: str, timeout: float = 25.0) -> None:
        self.base = base_url.rstrip("/")
        self.timeout = timeout

    def get(self, path: str, params: dict[str, Any] | None = None) -> str:
        with httpx.Client(timeout=self.timeout) as client:
            r = client.get(f"{self.base}{path}", params=params or {})
            if r.status_code >= 400:
                return f"ERROR {r.status_code}: {r.text[:500]}"
            try:
                return json.dumps(r.json(), indent=2)
            except Exception:
                return r.text

    def post(self, path: str, body: dict[str, Any]) -> str:
        with httpx.Client(timeout=self.timeout) as client:
            r = client.post(f"{self.base}{path}", json=body)
            if r.status_code >= 400:
                return f"ERROR {r.status_code}: {r.text[:500]}"
            try:
                return json.dumps(r.json(), indent=2)
            except Exception:
                return r.text

    def delete(self, path: str, params: dict[str, Any]) -> str:
        with httpx.Client(timeout=self.timeout) as client:
            r = client.delete(f"{self.base}{path}", params=params)
            if r.status_code >= 400:
                return f"ERROR {r.status_code}: {r.text[:500]}"
            try:
                return json.dumps(r.json(), indent=2)
            except Exception:
                return r.text


class TickerInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or BTC-USD")
    exchange: str = Field(default="binance", description="binance|coinbase|bybit")


class CandlesInput(BaseModel):
    symbol: str
    exchange: str = "binance"
    interval: str = "1h"
    limit: int = Field(default=50, ge=1, le=1000)


class SupplyInput(BaseModel):
    asset: str = Field(description="Base asset e.g. BTC")


class SpotInput(BaseModel):
    exchange: str = "binance"
    q: str = ""
    quote: str = ""
    sort: str = "quoteVolume"
    order: str = "desc"
    tag: str = ""
    limit: int = Field(default=15, ge=1, le=100)
    offset: int = 0


class IndicatorsInput(BaseModel):
    symbol: str
    exchange: str = "binance"
    interval: str = "1h"
    limit: int = 30
    # Snake_case only — LangChain tool kwargs use schema field names.
    rsi_period: int = Field(default=14, description="RSI period (default 14)")
    ema_periods: str = Field(default="12,26", description="Comma-separated EMA periods")


class WatchGetInput(BaseModel):
    client_id: str = Field(description="Opaque client id (required)")


class WatchAddInput(BaseModel):
    client_id: str
    symbol: str
    exchange: str = "binance"
    note: str = ""


class WatchRemoveInput(BaseModel):
    client_id: str
    symbol: str
    exchange: str = "binance"


class PumpDetectInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or JUVUSDT")
    exchange: str = Field(default="binance", description="binance|coinbase|bybit")
    interval: str = Field(default="1h", description="Candle interval: 1m,5m,15m,1h,4h,1d…")
    lookback_hours: float = Field(default=48, description="Hours of history to scan")
    min_return_pct: float = Field(default=5, description="Threshold percent e.g. 5 = +5%")
    window_bars: int = Field(default=1, ge=1, le=100, description="Bars for close_return window")
    mode: str = Field(
        default="close_return",
        description="close_return | candle_body | high_from_low",
    )
    direction: str = Field(default="up", description="up | down | both")
    min_volume_ratio: float = Field(default=0, description="Volume ≥ N× median (0=off)")
    max_events: int = Field(default=20, ge=1, le=100)


class PumpScanInput(BaseModel):
    exchange: str = "binance"
    quote: str = Field(default="USDT", description="Quote filter USDT/USD")
    interval: str = Field(default="15m", description="Candle interval for scan")
    lookback_hours: float = Field(default=24, description="Hours to scan")
    min_return_pct: float = Field(default=8, description="Threshold percent")
    window_bars: int = Field(default=1, ge=1, le=100)
    mode: str = "close_return"
    direction: str = "up"
    min_volume_ratio: float = 0
    symbol_limit: int = Field(default=15, ge=1, le=40, description="Top volume symbols to scan")
    max_total_events: int = Field(
        default=30,
        ge=1,
        le=200,
        description="Cap on total events across all symbols (not hit count)",
    )


def build_market_tools(settings: Settings | None = None) -> list[StructuredTool]:
    cfg = settings or get_settings()
    http = _HTTP(cfg.api_base_url)

    def health() -> str:
        return http.get("/health")

    def list_exchanges() -> str:
        return http.get("/api/v1/market/exchanges")

    def get_ticker(symbol: str, exchange: str = "binance") -> str:
        return http.get(
            "/api/v1/market/ticker/24h",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_candles(
        symbol: str,
        exchange: str = "binance",
        interval: str = "1h",
        limit: int = 50,
    ) -> str:
        return http.get(
            "/api/v1/market/candles",
            {
                "symbol": symbol,
                "exchange": exchange,
                "interval": interval,
                "limit": limit,
            },
        )

    def get_supply(asset: str) -> str:
        return http.get("/api/v1/market/supply", {"asset": asset})

    def list_spot_markets(
        exchange: str = "binance",
        q: str = "",
        quote: str = "",
        sort: str = "quoteVolume",
        order: str = "desc",
        tag: str = "",
        limit: int = 15,
        offset: int = 0,
    ) -> str:
        params: dict[str, Any] = {
            "exchange": exchange,
            "sort": sort,
            "order": order,
            "limit": limit,
            "offset": offset,
        }
        if q:
            params["q"] = q
        if quote:
            params["quote"] = quote
        if tag:
            params["tag"] = tag
        return http.get("/api/v1/market/spot", params)

    def get_indicators(
        symbol: str,
        exchange: str = "binance",
        interval: str = "1h",
        limit: int = 30,
        rsi_period: int = 14,
        ema_periods: str = "12,26",
        # Accept camelCase from older prompts if the model still sends them.
        rsiPeriod: int | None = None,
        emaPeriods: str | None = None,
    ) -> str:
        rp = rsiPeriod if rsiPeriod is not None else rsi_period
        ep = emaPeriods if emaPeriods is not None else ema_periods
        return http.get(
            "/api/v1/market/indicators",
            {
                "symbol": symbol,
                "exchange": exchange,
                "interval": interval,
                "limit": limit,
                "rsiPeriod": rp,
                "emaPeriods": ep,
            },
        )

    def get_watchlist(client_id: str) -> str:
        return http.get("/api/v1/watchlist", {"clientId": client_id})

    def add_watchlist_item(
        client_id: str,
        symbol: str,
        exchange: str = "binance",
        note: str = "",
    ) -> str:
        return http.post(
            "/api/v1/watchlist/items",
            {
                "clientId": client_id,
                "symbol": symbol,
                "exchange": exchange,
                "note": note,
            },
        )

    def remove_watchlist_item(
        client_id: str,
        symbol: str,
        exchange: str = "binance",
    ) -> str:
        return http.delete(
            "/api/v1/watchlist/items",
            {
                "clientId": client_id,
                "symbol": symbol,
                "exchange": exchange,
            },
        )

    def detect_pump_events(
        symbol: str,
        exchange: str = "binance",
        interval: str = "1h",
        lookback_hours: float = 48,
        min_return_pct: float = 5,
        window_bars: int = 1,
        mode: str = "close_return",
        direction: str = "up",
        min_volume_ratio: float = 0,
        max_events: int = 20,
    ) -> str:
        params: dict[str, Any] = {
            "symbol": symbol,
            "exchange": exchange,
            "interval": interval,
            "lookbackHours": lookback_hours,
            "minReturnPct": min_return_pct,
            "windowBars": window_bars,
            "mode": mode,
            "direction": direction,
            "maxEvents": max_events,
        }
        if min_volume_ratio and min_volume_ratio > 0:
            params["minVolumeRatio"] = min_volume_ratio
        return http.get("/api/v1/market/pumps", params)

    def scan_pump_events(
        exchange: str = "binance",
        quote: str = "USDT",
        interval: str = "15m",
        lookback_hours: float = 24,
        min_return_pct: float = 8,
        window_bars: int = 1,
        mode: str = "close_return",
        direction: str = "up",
        min_volume_ratio: float = 0,
        symbol_limit: int = 15,
        max_total_events: int = 30,
    ) -> str:
        params: dict[str, Any] = {
            "exchange": exchange,
            "quote": quote,
            "interval": interval,
            "lookbackHours": lookback_hours,
            "minReturnPct": min_return_pct,
            "windowBars": window_bars,
            "mode": mode,
            "direction": direction,
            "symbolLimit": symbol_limit,
            "maxTotalEvents": max_total_events,
        }
        if min_volume_ratio and min_volume_ratio > 0:
            params["minVolumeRatio"] = min_volume_ratio
        return http.get("/api/v1/market/pumps/scan", params)

    return [
        StructuredTool.from_function(
            health,
            name="health",
            description="Check Swyngora API health.",
        ),
        StructuredTool.from_function(
            list_exchanges,
            name="list_exchanges",
            description="List configured market venues.",
        ),
        StructuredTool.from_function(
            get_ticker,
            name="get_ticker",
            description="24h ticker (price, volume, change) for a pair.",
            args_schema=TickerInput,
        ),
        StructuredTool.from_function(
            get_candles,
            name="get_candles",
            description="OHLCV candles for a pair.",
            args_schema=CandlesInput,
        ),
        StructuredTool.from_function(
            get_supply,
            name="get_supply",
            description="Circulating/total/max supply for a base asset.",
            args_schema=SupplyInput,
        ),
        StructuredTool.from_function(
            list_spot_markets,
            name="list_spot_markets",
            description="Search/sort spot markets (volume, mcap, tags).",
            args_schema=SpotInput,
        ),
        StructuredTool.from_function(
            get_indicators,
            name="get_indicators",
            description="RSI + EMA indicators. Informational only.",
            args_schema=IndicatorsInput,
        ),
        StructuredTool.from_function(
            get_watchlist,
            name="get_watchlist",
            description="Get watchlist for a clientId.",
            args_schema=WatchGetInput,
        ),
        StructuredTool.from_function(
            add_watchlist_item,
            name="add_watchlist_item",
            description="Add symbol to watchlist.",
            args_schema=WatchAddInput,
        ),
        StructuredTool.from_function(
            remove_watchlist_item,
            name="remove_watchlist_item",
            description="Remove symbol from watchlist.",
            args_schema=WatchRemoveInput,
        ),
        StructuredTool.from_function(
            detect_pump_events,
            name="detect_pump_events",
            description=(
                "Detect pump/dump events on ONE symbol from candles. "
                "Set min_return_pct, lookback_hours, interval (1m/5m/15m/1h…), "
                "window_bars, mode (close_return|candle_body|high_from_low), direction. "
                "Mechanical filter only — not a trade signal."
            ),
            args_schema=PumpDetectInput,
        ),
        StructuredTool.from_function(
            scan_pump_events,
            name="scan_pump_events",
            description=(
                "Scan top-volume symbols for recent pumps/dumps with thresholds "
                "(min_return_pct, lookback_hours, interval, mode, direction, max_total_events). "
                "max_total_events caps aggregate events across symbols. "
                "Response metadata echoes resolved defaults. "
                "Use for 'what pumped in the last N hours'."
            ),
            args_schema=PumpScanInput,
        ),
    ]
