"""Read tools raise ToolTransientError; mutations and 4xx stay fail-soft strings."""

from __future__ import annotations

import httpx
import pytest

from swyngora_ai.tools.errors import ToolTransientError
from swyngora_ai.tools.market_http import _HTTP, bind_tool_scope, reset_tool_scope


def _client(transport: httpx.BaseTransport) -> _HTTP:
    http = _HTTP("http://test")
    http._client.close()
    http._client = httpx.Client(transport=transport, timeout=5.0)
    return http


class _TimeoutTransport(httpx.BaseTransport):
    def handle_request(self, request: httpx.Request) -> httpx.Response:
        raise httpx.TimeoutException("slow")


class _ConnectTransport(httpx.BaseTransport):
    def handle_request(self, request: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("refused")


class _StatusTransport(httpx.BaseTransport):
    def __init__(self, status: int) -> None:
        self.status = status

    def handle_request(self, request: httpx.Request) -> httpx.Response:
        return httpx.Response(self.status, text=f"body-{self.status}")


def test_get_timeout_raises_transient() -> None:
    http = _client(_TimeoutTransport())
    with pytest.raises(ToolTransientError, match="timeout"):
        http.get("/api/v1/market/ticker")


def test_get_connect_raises_transient() -> None:
    http = _client(_ConnectTransport())
    with pytest.raises(ToolTransientError, match="connect"):
        http.get("/api/v1/market/ticker")


def test_get_429_and_5xx_raise_transient() -> None:
    with pytest.raises(ToolTransientError, match="429"):
        _client(_StatusTransport(429)).get("/api/v1/market/ticker")
    with pytest.raises(ToolTransientError, match="502"):
        _client(_StatusTransport(502)).get("/api/v1/market/ticker")


def test_get_4xx_stays_error_string() -> None:
    out = _client(_StatusTransport(404)).get("/api/v1/market/ticker")
    assert out.startswith("ERROR 404")
    forbidden = _client(_StatusTransport(403)).get("/api/v1/market/ticker")
    assert forbidden.startswith("ERROR 403")


def test_post_timeout_stays_error_string() -> None:
    toks = bind_tool_scope(can_trade=True, can_manage_keys=True)
    try:
        out = _client(_TimeoutTransport()).post("/api/v1/portfolio/orders", {"symbol": "BTCUSDT"})
        assert out.startswith("ERROR timeout")
    finally:
        reset_tool_scope(toks)


def test_post_502_stays_error_string() -> None:
    toks = bind_tool_scope(can_trade=True, can_manage_keys=True)
    try:
        out = _client(_StatusTransport(502)).post("/api/v1/portfolio/orders", {"symbol": "BTCUSDT"})
        assert out.startswith("ERROR 502")
    finally:
        reset_tool_scope(toks)


def test_delete_timeout_stays_error_string() -> None:
    toks = bind_tool_scope(can_trade=True, can_manage_keys=True)
    try:
        out = _client(_TimeoutTransport()).delete("/api/v1/portfolio/orders/1", {})
        assert out.startswith("ERROR timeout")
    finally:
        reset_tool_scope(toks)
