"""Tool response redaction and tenant binding for AI market tools."""

from __future__ import annotations

import httpx

from swyngora_ai.config import Settings
from swyngora_ai.tools.market_http import (
    bind_client_id,
    build_market_tools,
    redact_secrets,
    reset_bound_client_id,
)


class _RecordingTransport(httpx.BaseTransport):
    def __init__(self) -> None:
        self.requests: list[httpx.Request] = []

    def handle_request(self, request: httpx.Request) -> httpx.Response:
        self.requests.append(request)
        if request.url.path == "/api/v1/account/api-keys" and request.method == "POST":
            return httpx.Response(
                201,
                json={
                    "id": "k1",
                    "name": "pwn",
                    "permission": "trade",
                    "secret": "swy_leaked_secret_should_be_redacted0123456789abcdef",
                    "prefix": "swy_leaked",
                },
            )
        return httpx.Response(404, json={"error": "not found"})


def test_redact_secrets_strips_secret_fields():
    out = redact_secrets(
        {
            "id": "k1",
            "secret": "swy_abc1234567890abcdef01234567890abcdef01",
            "nested": {"token": "x"},
        }
    )
    assert out["secret"] == "[redacted]"
    assert out["nested"]["token"] == "[redacted]"
    assert out["id"] == "k1"


def test_create_api_key_tool_redacts_secret(monkeypatch):
    transport = _RecordingTransport()
    real_client = httpx.Client

    def fake_client(*args, **kwargs):
        kwargs["transport"] = transport
        kwargs.pop("timeout", None)
        return real_client(*args, timeout=5.0, transport=transport)

    monkeypatch.setattr(httpx, "Client", fake_client)
    monkeypatch.setenv("SWYNGORA_API_URL", "http://backend.test")
    monkeypatch.setenv("SWYNGORA_API_TOKEN", "master-token")

    tools = build_market_tools(Settings())
    by_name = {t.name: t for t in tools}
    tok = bind_client_id("victim-client")
    try:
        out = by_name["create_api_key"].invoke(
            {"client_id": "ignored", "name": "pwn", "permission": "trade"}
        )
    finally:
        reset_bound_client_id(tok)

    assert transport.requests
    last = transport.requests[-1]
    assert last.headers.get("Authorization") == "Bearer master-token"
    assert last.headers.get("X-Client-Id") == "victim-client"
    assert "swy_leaked_secret" not in out
    assert "[redacted]" in out
