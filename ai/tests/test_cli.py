"""CLI tests (Typer; no live LLM)."""

from __future__ import annotations

import os

from typer.testing import CliRunner

from swyngora_ai.cli import _status_text, app
from swyngora_ai.graph.orchestrator import ChatResult


class _FakeOrch:
    def __init__(self) -> None:
        self.chats: list[tuple[str, str]] = []
        self.resets: list[str] = []

    def chat(self, message: str, session_id: str = "default", on_event=None) -> ChatResult:
        self.chats.append((message, session_id))
        return ChatResult(
            reply=f"echo:{message}",
            session_id=session_id,
            tools=["market_agent(BTC)"],
            thinking=["checking RSI"],
            references=[{"title": "Binance", "url": "https://example.com/btc", "source": "web"}],
        )

    def reset(self, session_id: str, client_id: str = "") -> None:
        self.resets.append(session_id)


class _BoomOrch(_FakeOrch):
    def chat(self, message: str, session_id: str = "default", on_event=None) -> ChatResult:
        raise RuntimeError("backend down")


runner = CliRunner()


def test_status_text_filters_noise_and_truncates() -> None:
    assert _status_text({"type": "final", "text": "done"}) is None
    assert _status_text({"type": "ping", "text": "x"}) is None
    assert _status_text({"type": "status", "text": "Planning…"}) == "Planning…"
    long = "x" * 120
    out = _status_text({"type": "tool", "text": long})
    assert out is not None
    assert out.endswith("…")
    assert len(out) == 88


def test_help() -> None:
    result = runner.invoke(app, ["--help"])
    assert result.exit_code == 0
    assert "One-shot question" in result.stdout
    assert "--session" in result.stdout
    assert "--provider" in result.stdout


def test_one_shot_prints_reply_not_dataclass(monkeypatch) -> None:
    fake = _FakeOrch()
    monkeypatch.setattr("swyngora_ai.cli._build_orchestrator", lambda: fake)
    result = runner.invoke(app, ["What is BTC RSI?", "--session", "desk"])
    assert result.exit_code == 0
    assert "echo:What is BTC RSI?" in result.stdout
    assert "ChatResult" not in result.stdout
    assert "details" in result.stdout
    assert "checking RSI" in result.stdout
    assert "market_agent(BTC)" in result.stdout
    assert "Binance" in result.stdout
    assert "https://example.com/btc" in result.stdout
    assert "desk" in result.stdout
    assert fake.chats == [("What is BTC RSI?", "desk")]


def test_provider_sets_env(monkeypatch) -> None:
    fake = _FakeOrch()
    monkeypatch.setattr("swyngora_ai.cli._build_orchestrator", lambda: fake)
    monkeypatch.delenv("AI_LLM_PROVIDER", raising=False)
    result = runner.invoke(app, ["--provider", "grok", "hi"])
    assert result.exit_code == 0
    assert os.environ["AI_LLM_PROVIDER"] == "grok"
    assert fake.chats == [("hi", "cli")]


def test_one_shot_error_is_caught(monkeypatch) -> None:
    monkeypatch.setattr("swyngora_ai.cli._build_orchestrator", _BoomOrch)
    result = runner.invoke(app, ["ping"])
    assert result.exit_code == 1
    assert "error: backend down" in result.stderr
    assert "Traceback" not in result.stderr


def test_repl_reset_and_exit(monkeypatch) -> None:
    fake = _FakeOrch()
    monkeypatch.setattr("swyngora_ai.cli._build_orchestrator", lambda: fake)
    result = runner.invoke(app, ["--session", "s1"], input="reset\nhello\nexit\n")
    assert result.exit_code == 0
    assert fake.resets == ["s1"]
    assert fake.chats == [("hello", "s1")]
    assert "echo:hello" in result.stdout
    assert "session cleared" in result.stdout
    assert "not financial advice" in result.stdout.lower()
    assert "leave the session" in result.stdout
    assert "clear conversation memory" in result.stdout
