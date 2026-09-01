"""HTTP chat server for Swyngora AI (used by Telegram + REST proxy).

Supports:
  POST /v1/chat          → JSON {reply, tools, thinking}
  POST /v1/chat/stream   → NDJSON lines of live events + final

  python -m swyngora_ai.serve --host 127.0.0.1 --port 8090
"""

from __future__ import annotations

import json
import queue
import sys
import threading
import traceback
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Annotated, Any
from urllib.parse import urlparse

import typer

from swyngora_ai.config import get_settings
from swyngora_ai.graph.orchestrator import Orchestrator, build_orchestrator

_orch: Orchestrator | None = None


def is_service_authorized(headers: Any, token: str) -> bool:
    """True when AI_SERVICE_TOKEN is empty or the request carries it."""
    expected = (token or "").strip()
    if not expected:
        return True
    auth = (headers.get("Authorization") or "").strip()
    if auth == f"Bearer {expected}":
        return True
    return (headers.get("X-AI-Token") or "").strip() == expected


def get_orch() -> Orchestrator:
    global _orch  # noqa: PLW0603 — process-wide HTTP singleton
    if _orch is None:
        _orch = build_orchestrator()
    return _orch


class Handler(BaseHTTPRequestHandler):
    server_version = "SwyngoraAI/0.1"

    def log_message(self, format: str, *args: Any) -> None:
        sys.stderr.write(f"{self.address_string()} - {format % args}\n")

    def _read_json(self) -> dict[str, Any] | None:
        length = int(self.headers.get("Content-Length") or "0")
        if length <= 0 or length > 1 << 20:
            return None
        try:
            return json.loads(self.rfile.read(length).decode("utf-8"))
        except (json.JSONDecodeError, UnicodeDecodeError, ValueError):
            return None

    def _json(self, code: int, body: dict[str, Any]) -> None:
        data = json.dumps(body).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def _authorized(self) -> bool:
        return is_service_authorized(self.headers, get_settings().service_token)

    def do_GET(self) -> None:
        path = urlparse(self.path).path
        if path in ("/health", "/v1/health"):
            self._json(200, {"status": "ok", "service": "swyngora-ai"})
            return
        self._json(404, {"error": "not found"})

    def do_POST(self) -> None:
        if not self._authorized():
            self._json(
                401, {"error": "unauthorized", "message": "invalid or missing AI_SERVICE_TOKEN"}
            )
            return
        path = urlparse(self.path).path
        payload = self._read_json()
        if payload is None:
            self._json(400, {"error": "invalid body"})
            return
        message = str(payload.get("message") or payload.get("text") or "").strip()
        if not message:
            self._json(400, {"error": "message is required"})
            return
        session_id = str(payload.get("sessionId") or payload.get("session_id") or "default")
        client_id = str(payload.get("clientId") or payload.get("client_id") or "").strip()
        # Missing flags deny mutations (Go proxy always sends both).
        can_trade = (
            payload["canTrade"] if "canTrade" in payload else payload.get("can_trade", False)
        )
        can_manage_keys = (
            payload["canManageKeys"]
            if "canManageKeys" in payload
            else payload.get("can_manage_keys", False)
        )
        can_trade = bool(can_trade)
        can_manage_keys = bool(can_manage_keys)

        if path in ("/v1/chat/stream", "/chat/stream"):
            self._stream_chat(message, session_id, client_id, can_trade, can_manage_keys)
            return
        if path not in ("/v1/chat", "/chat"):
            self._json(404, {"error": "not found"})
            return
        try:
            result = get_orch().chat(
                message,
                session_id=session_id,
                client_id=client_id,
                can_trade=can_trade,
                can_manage_keys=can_manage_keys,
            )
            self._json(
                200,
                {
                    "reply": result.reply,
                    "sessionId": result.session_id,
                    "tools": result.tools,
                    "thinking": result.thinking,
                    "references": result.references,
                },
            )
        except Exception as e:  # noqa: BLE001
            traceback.print_exc()
            self._json(502, {"error": "ai_failed", "message": str(e)[:500]})

    def _stream_chat(
        self,
        message: str,
        session_id: str,
        client_id: str = "",
        can_trade: bool = False,
        can_manage_keys: bool = False,
    ) -> None:
        """NDJSON stream: one JSON object per line (status/tool/thinking/final/error/done)."""
        q: queue.Queue[dict[str, Any] | None] = queue.Queue()

        def on_event(ev: dict[str, Any]) -> None:
            q.put(ev)

        def worker() -> None:
            try:
                result = get_orch().chat(
                    message,
                    session_id=session_id,
                    on_event=on_event,
                    client_id=client_id,
                    can_trade=can_trade,
                    can_manage_keys=can_manage_keys,
                )
                q.put(
                    {
                        "type": "final",
                        "reply": result.reply,
                        "tools": result.tools,
                        "thinking": result.thinking,
                        "references": result.references,
                        "sessionId": result.session_id,
                    }
                )
            except Exception as e:  # noqa: BLE001
                traceback.print_exc()
                q.put({"type": "error", "message": str(e)[:500]})
            finally:
                q.put(None)

        threading.Thread(target=worker, daemon=True).start()

        self.send_response(200)
        self.send_header("Content-Type", "application/x-ndjson; charset=utf-8")
        self.send_header("Cache-Control", "no-cache, no-store")
        self.send_header("X-Accel-Buffering", "no")
        self.send_header("Connection", "close")
        self.end_headers()
        # Immediate first line so clients/Telegram can open the progress card before tools run.
        try:
            self.wfile.write(b'{"type":"status","text":"Planning\\u2026"}\n')
            self.wfile.flush()
            while True:
                try:
                    ev = q.get(timeout=2.0)
                except queue.Empty:
                    # Soft keep-alive (type=ping) so proxies do not buffer the whole response.
                    # Clients should ignore ping for UI status text.
                    self.wfile.write(b'{"type":"ping"}\n')
                    self.wfile.flush()
                    continue
                if ev is None:
                    self.wfile.write(b'{"type":"done"}\n')
                    self.wfile.flush()
                    break
                line = (json.dumps(ev, ensure_ascii=False) + "\n").encode("utf-8")
                self.wfile.write(line)
                self.wfile.flush()
        except BrokenPipeError:
            pass


app = typer.Typer(
    add_completion=False,
    pretty_exceptions_enable=False,
    context_settings={"help_option_names": ["-h", "--help"]},
)


def run_server(host: str, port: int) -> int:
    try:
        get_orch()
    except Exception as e:  # noqa: BLE001
        print(f"failed to start orchestrator: {e}", file=sys.stderr)
        return 1

    cfg = get_settings()
    model = cfg.grok_model if cfg.llm_provider == "grok" else cfg.ollama_model
    httpd = ThreadingHTTPServer((host, port), Handler)
    print(
        f"swyngora-ai listening on http://{host}:{port} provider={cfg.llm_provider} model={model}",
        flush=True,
    )
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("shutting down", flush=True)
    return 0


@app.command()
def run(
    host: Annotated[str, typer.Option("--host", help="Bind address")] = "127.0.0.1",
    port: Annotated[int, typer.Option("--port", help="Bind port")] = 8090,
) -> None:
    """Swyngora AI HTTP server."""
    raise typer.Exit(run_server(host, port))


def main(argv: list[str] | None = None) -> None:
    app(args=argv, prog_name="swyngora-ai-serve")


if __name__ == "__main__":
    main()
