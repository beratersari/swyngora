"""Serve CLI tests (Typer; no live bind or LLM)."""

from __future__ import annotations

from typer.testing import CliRunner

from swyngora_ai.serve import app


class _FakeServer:
    def __init__(self, addr: tuple[str, int], _handler: object) -> None:
        self.addr = addr

    def serve_forever(self) -> None:
        return None


runner = CliRunner()


def test_help() -> None:
    result = runner.invoke(app, ["--help"])
    assert result.exit_code == 0
    assert "--host" in result.stdout
    assert "--port" in result.stdout


def test_invalid_port_is_nonzero() -> None:
    result = runner.invoke(app, ["--port", "nope"])
    assert result.exit_code != 0


def test_host_port_passed_to_server(monkeypatch) -> None:
    seen: list[tuple[str, int]] = []

    def fake_server(addr: tuple[str, int], handler: object) -> _FakeServer:
        seen.append(addr)
        return _FakeServer(addr, handler)

    monkeypatch.setattr("swyngora_ai.serve.get_orch", object)
    monkeypatch.setattr("swyngora_ai.serve.ThreadingHTTPServer", fake_server)
    result = runner.invoke(app, ["--host", "127.0.0.1", "--port", "8091"])
    assert result.exit_code == 0
    assert seen == [("127.0.0.1", 8091)]


def test_orchestrator_start_failure_is_exit_1(monkeypatch) -> None:
    def boom() -> object:
        raise RuntimeError("no model")

    monkeypatch.setattr("swyngora_ai.serve.get_orch", boom)
    result = runner.invoke(app, ["--port", "8091"])
    assert result.exit_code == 1
    assert "failed to start orchestrator" in result.stderr
