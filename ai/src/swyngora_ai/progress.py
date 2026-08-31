"""Progress callbacks for live tool/thinking updates (Telegram, SSE).

Uses ContextVar plus a thread-safe stack so emits still work when LangChain
runs work on threads that do not inherit the request context.
"""

from __future__ import annotations

import threading
from collections.abc import Callable
from contextvars import ContextVar
from typing import Any

ProgressCallback = Callable[[dict[str, Any]], None]

_progress_cb: ContextVar[ProgressCallback | None] = ContextVar("swyngora_progress", default=None)
_stack_lock = threading.Lock()
_callback_stack: list[ProgressCallback] = []


def set_progress(cb: ProgressCallback | None):
    """Register a progress callback for the current chat turn. Returns a reset token."""
    if cb is not None:
        with _stack_lock:
            _callback_stack.append(cb)
    return _progress_cb.set(cb)


def reset_progress(token) -> None:
    """Pop the stack entry that matches the current ContextVar callback, then reset."""
    current = _progress_cb.get()
    if current is not None:
        with _stack_lock:
            # Remove latest matching callback (LIFO — concurrent chats supported).
            for i in range(len(_callback_stack) - 1, -1, -1):
                if _callback_stack[i] is current:
                    _callback_stack.pop(i)
                    break
    _progress_cb.reset(token)


def is_tool_error_text(text: str) -> bool:
    """True when a tool returned our fail-soft ERROR string (not a raised exception)."""
    return (text or "").lstrip().startswith("ERROR")


def emit_tool_outcome(name: str, preview: str) -> None:
    """Emit tool_result (✓) or tool_error when the body is an ERROR … string."""
    if is_tool_error_text(preview):
        emit("tool_error", f"{name} failed: {preview}" if preview else f"{name} failed")
        return
    emit("tool_result", f"{name} ✓ {preview}" if preview else f"{name} ✓")


def emit(event_type: str, text: str = "", **extra: Any) -> None:
    cb = _progress_cb.get()
    if cb is None:
        with _stack_lock:
            # Recover only when exactly one chat is in flight. Two or more
            # overlapping chats must not inherit the other user's callback.
            if len(_callback_stack) == 1:
                cb = _callback_stack[0]
    if cb is None:
        return
    payload: dict[str, Any] = {"type": event_type, "text": text}
    payload.update(extra)
    try:
        cb(payload)
    except Exception:  # noqa: BLE001
        pass
