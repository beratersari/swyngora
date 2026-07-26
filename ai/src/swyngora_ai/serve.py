"""HTTP chat server for Swyngora AI (used by Telegram + REST proxy).

Supports:
  POST /v1/chat          → JSON {reply, tools, thinking}
  POST /v1/chat/stream   → NDJSON lines of live events + final

  python -m swyngora_ai.serve --host 127.0.0.1 --port 8090
"""

from __future__ import annotations

import argparse
import json
import queue
import sys
import threading
import traceback
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any
from urllib.parse import urlparse

from swyngora_ai.graph.orchestrator import Orchestrator, build_orchestrator

_orch: Orchestrator | None = None


def get_orch() -> Orchestrator:
    global _orch
    if _orch is None:
        _orch = build_orchestrator()
    return _orch


class Handler(BaseHTTPRequestHandler):
    server_version = "SwyngoraAI/0.1"

    def log_message(self, fmt: str, *args: Any) -> None:
        sys.stderr.write("%s - %s\n" % (self.address_string(), fmt % args))

    def _read_json(self) -> dict[str, Any] | None:
        length = int(self.headers.get("Content-Length") or "0")
        if length <= 0 or length > 1 << 20:
            return None
        try:
            return json.loads(self.rfile.read(length).decode("utf-8"))
        except Exception:
            return None

    def _json(self, code: int, body: dict[str, Any]) -> None:
        data = json.dumps(body).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self) -> None:  # noqa: N802
        path = urlparse(self.path).path
        if path in ("/health", "/v1/health"):
            self._json(200, {"status": "ok", "service": "swyngora-ai"})
            return
        self._json(404, {"error": "not found"})

    def do_POST(self) -> None:  # noqa: N802
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

        if path in ("/v1/chat/stream", "/chat/stream"):
            self._stream_chat(message, session_id)
            return
        if path not in ("/v1/chat", "/chat"):
            self._json(404, {"error": "not found"})
            return
        try:
            result = get_orch().chat(message, session_id=session_id)
            self._json(
                200,
                {
                    "reply": result.reply,
                    "sessionId": result.session_id,
                    "tools": result.tools,
                    "thinking": result.thinking,
                },
            )
        except Exception as e:  # noqa: BLE001
            traceback.print_exc()
            self._json(502, {"error": "ai_failed", "message": str(e)[:500]})

    def _stream_chat(self, message: str, session_id: str) -> None:
        """NDJSON stream: one JSON object per line (status/tool/thinking/final/error/done)."""
        q: queue.Queue[dict[str, Any] | None] = queue.Queue()

        def on_event(ev: dict[str, Any]) -> None:
            q.put(ev)

        def worker() -> None:
            try:
                result = get_orch().chat(message, session_id=session_id, on_event=on_event)
                q.put(
                    {
                        "type": "final",
                        "reply": result.reply,
                        "tools": result.tools,
                        "thinking": result.thinking,
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


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="Swyngora AI HTTP server")
    p.add_argument("--host", default="127.0.0.1")
    p.add_argument("--port", type=int, default=8090)
    args = p.parse_args(argv)
    try:
        get_orch()
    except Exception as e:  # noqa: BLE001
        print(f"failed to start orchestrator: {e}", file=sys.stderr)
        return 1
    httpd = ThreadingHTTPServer((args.host, args.port), Handler)
    print(f"swyngora-ai listening on http://{args.host}:{args.port}", flush=True)
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("shutting down", flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
