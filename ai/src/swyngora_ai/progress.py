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


def emit(event_type: str, text: str = "", **extra: Any) -> None:
    cb = _progress_cb.get()
    if cb is None:
        with _stack_lock:
            if _callback_stack:
                cb = _callback_stack[-1]
    if cb is None:
        return
    payload: dict[str, Any] = {"type": event_type, "text": text}
    payload.update(extra)
    try:
        cb(payload)
    except Exception:
        pass
