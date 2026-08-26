"""Prove concurrent chat binds cannot leak tenant identity on pool threads."""

from __future__ import annotations

import threading
import time
from concurrent.futures import ThreadPoolExecutor

import httpx

from swyngora_ai.config import Settings
from swyngora_ai.tools.market_http import (
    bind_client_id,
    bind_tool_scope,
    build_market_tools,
    reset_bound_client_id,
    reset_tool_scope,
    submit_with_bound_context,
)


class _RecordingTransport(httpx.BaseTransport):
    def __init__(self) -> None:
        self.requests: list[httpx.Request] = []
        self.lock = threading.Lock()

    def handle_request(self, request: httpx.Request) -> httpx.Response:
        with self.lock:
            self.requests.append(request)
        # Hold the request so two chat turns stay overlapped on _bind_stack.
        time.sleep(0.15)
        return httpx.Response(200, json={"ok": True, "path": str(request.url.path)})


def _install(monkeypatch, transport: _RecordingTransport):
    real = httpx.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = transport
        kwargs.pop("timeout", None)
        return real(*args, timeout=5.0, transport=transport)

    monkeypatch.setattr(httpx, "Client", fake_client)
    return {t.name: t for t in build_market_tools(Settings(api_base_url="http://backend.test"))}


def _client_id(req: httpx.Request) -> str:
    return req.headers.get("X-Client-Id") or req.url.params.get("clientId") or ""


def test_concurrent_chats_on_pool_threads_do_not_swap_tenant(monkeypatch):
    """Desk gather copies ContextVars onto workers; each turn keeps its tenant."""
    transport = _RecordingTransport()
    tools = _install(monkeypatch, transport)
    barrier = threading.Barrier(2)
    errors: list[str] = []

    def chat_turn(own_id: str) -> None:
        tok = bind_client_id(own_id)
        scope = bind_tool_scope(can_trade=True, can_manage_keys=False)
        try:
            barrier.wait(timeout=5)

            def on_pool() -> str:
                return tools["get_portfolio"].invoke({"client_id": "model-should-be-ignored"})

            with ThreadPoolExecutor(max_workers=1) as pool:
                out = submit_with_bound_context(pool, on_pool).result(timeout=5)
            if "ERROR" in out:
                errors.append(f"{own_id}: {out}")
        finally:
            reset_tool_scope(scope)
            reset_bound_client_id(tok)

    t1 = threading.Thread(target=chat_turn, args=("alice-tenant",))
    t2 = threading.Thread(target=chat_turn, args=("bob-tenant",))
    t1.start()
    t2.start()
    t1.join(timeout=8)
    t2.join(timeout=8)
    assert not t1.is_alive()
    assert not t2.is_alive()
    assert not errors, errors
    assert len(transport.requests) == 2, [r.url for r in transport.requests]
    clients = [_client_id(req) for req in transport.requests]
    assert set(clients) == {"alice-tenant", "bob-tenant"}, (
        f"concurrent AI chats mixed tenant headers via _bind_stack: {clients}"
    )


def test_concurrent_chats_do_not_grant_other_users_trade_scope(monkeypatch):
    transport = _RecordingTransport()
    tools = _install(monkeypatch, transport)
    barrier = threading.Barrier(2)
    results: dict[str, str] = {}

    def chat_turn(own_id: str, can_trade: bool) -> None:
        tok = bind_client_id(own_id)
        scope = bind_tool_scope(can_trade=can_trade, can_manage_keys=False)
        try:
            barrier.wait(timeout=5)

            def on_pool() -> str:
                return tools["place_portfolio_order"].invoke(
                    {
                        "client_id": "model-should-be-ignored",
                        "symbol": "BTCUSDT",
                        "side": "buy",
                        "quantity": 1,
                    }
                )

            with ThreadPoolExecutor(max_workers=1) as pool:
                results[own_id] = submit_with_bound_context(pool, on_pool).result(timeout=5)
        finally:
            reset_tool_scope(scope)
            reset_bound_client_id(tok)

    t1 = threading.Thread(target=chat_turn, args=("alice-readonly", False))
    t2 = threading.Thread(target=chat_turn, args=("bob-trader", True))
    t1.start()
    t2.start()
    t1.join(timeout=8)
    t2.join(timeout=8)
    assert not t1.is_alive()
    assert not t2.is_alive()

    alice_out = results.get("alice-readonly", "")
    alice_posted = [
        r
        for r in transport.requests
        if _client_id(r) == "alice-readonly"
        and r.method == "POST"
        and "/portfolio/orders" in r.url.path
    ]
    assert "403" in alice_out, alice_out
    assert "read-only" in alice_out, alice_out
    assert not alice_posted, (
        "read-only Alice posted a paper order during overlapping Bob trade-scope bind: "
        f"{[(r.method, r.url.path, _client_id(r)) for r in alice_posted]}"
    )
    bob_posted = [
        r
        for r in transport.requests
        if _client_id(r) == "bob-trader"
        and r.method == "POST"
        and "/portfolio/orders" in r.url.path
    ]
    assert bob_posted, results.get("bob-trader", "")


def test_pool_without_copied_context_does_not_guess_the_other_tenant(monkeypatch):
    """If a worker drops ContextVars while two chats overlap, fail closed."""
    transport = _RecordingTransport()
    tools = _install(monkeypatch, transport)
    started = threading.Barrier(2)
    both_on_pool = threading.Barrier(2)
    both_finished_invoke = threading.Barrier(2)
    results: dict[str, str] = {}

    def chat_turn(own_id: str) -> None:
        tok = bind_client_id(own_id)
        scope = bind_tool_scope(can_trade=True, can_manage_keys=False)
        try:
            started.wait(timeout=5)

            def on_pool() -> str:
                both_on_pool.wait(timeout=5)
                try:
                    return tools["get_portfolio"].invoke({"client_id": "model-should-be-ignored"})
                finally:
                    # Keep both binds live until each worker has chosen a tenant.
                    both_finished_invoke.wait(timeout=5)

            with ThreadPoolExecutor(max_workers=1) as pool:
                results[own_id] = pool.submit(on_pool).result(timeout=5)
        finally:
            reset_tool_scope(scope)
            reset_bound_client_id(tok)

    t1 = threading.Thread(target=chat_turn, args=("alice-tenant",))
    t2 = threading.Thread(target=chat_turn, args=("bob-tenant",))
    t1.start()
    t2.start()
    t1.join(timeout=8)
    t2.join(timeout=8)
    assert not t1.is_alive()
    assert not t2.is_alive()

    for own_id, out in results.items():
        assert "403" in out, (own_id, out)
        assert "ambiguous" in out, (own_id, out)
    leaked = [(_client_id(r), r.url.path) for r in transport.requests]
    assert not leaked, f"ambiguous bind still sent a tenant request: {leaked}"
