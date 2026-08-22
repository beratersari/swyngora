"""AI tool tenant/scope must survive LangChain worker threads."""

from __future__ import annotations

import threading

import httpx

from swyngora_ai.config import Settings
from swyngora_ai.tools.market_http import (
    bind_client_id,
    bind_tool_scope,
    build_market_tools,
    reset_bound_client_id,
    reset_tool_scope,
)


class _RecordingTransport(httpx.BaseTransport):
    def __init__(self) -> None:
        self.requests: list[httpx.Request] = []

    def handle_request(self, request: httpx.Request) -> httpx.Response:
        self.requests.append(request)
        return httpx.Response(200, json={"ok": True, "clientId": "should-not-matter"})


def _install_transport(monkeypatch, transport: _RecordingTransport) -> None:
    real = httpx.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = transport
        kwargs.pop("timeout", None)
        return real(*args, timeout=5.0, transport=transport)

    monkeypatch.setattr(httpx, "Client", fake_client)


def test_worker_thread_keeps_read_only_scope(monkeypatch):
    transport = _RecordingTransport()
    _install_transport(monkeypatch, transport)
    tools = {t.name: t for t in build_market_tools(Settings(api_base_url="http://backend.test"))}
    bind_tok = bind_client_id("tg-mapped-alice")
    scope = bind_tool_scope(can_trade=False, can_manage_keys=False)
    box: dict[str, str] = {}

    def worker() -> None:
        box["out"] = tools["place_portfolio_order"].invoke(
            {
                "client_id": "victim-other-tenant",
                "symbol": "BTCUSDT",
                "side": "buy",
                "quantity": 1,
            }
        )

    try:
        th = threading.Thread(target=worker)
        th.start()
        th.join(timeout=5)
    finally:
        reset_tool_scope(scope)
        reset_bound_client_id(bind_tok)

    assert th.is_alive() is False
    out = box.get("out", "")
    assert not transport.requests, (
        "worker thread lost can_trade=False and posted "
        f"{transport.requests[-1].method} {transport.requests[-1].url}"
    )
    assert "403" in out
    assert "read-only" in out


def test_worker_thread_keeps_bound_client_id(monkeypatch):
    transport = _RecordingTransport()
    _install_transport(monkeypatch, transport)
    tools = {t.name: t for t in build_market_tools(Settings(api_base_url="http://backend.test"))}
    bind_tok = bind_client_id("chat-user-alice")
    scope = bind_tool_scope(can_trade=True, can_manage_keys=False)
    box: dict[str, str] = {}

    def worker() -> None:
        box["out"] = tools["get_portfolio"].invoke({"client_id": "victim-bob"})

    try:
        th = threading.Thread(target=worker)
        th.start()
        th.join(timeout=5)
    finally:
        reset_tool_scope(scope)
        reset_bound_client_id(bind_tok)

    assert th.is_alive() is False
    assert transport.requests, f"expected GET, tool_out={box.get('out')!r}"
    last = transport.requests[-1]
    sent = last.url.params.get("clientId") or last.headers.get("X-Client-Id")
    assert sent == "chat-user-alice", (
        f"worker thread used model clientId {sent!r} instead of bound chat-user-alice (url={last.url})"
    )
