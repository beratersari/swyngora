"""Remember the last error on each chat model so a dual failure can name both."""

from __future__ import annotations

import threading
from typing import Any
from weakref import WeakSet

from langchain_core.callbacks import BaseCallbackHandler

from swyngora_ai.progress import emit
from swyngora_ai.tools.errors import ToolTransientError


def usage_from_llm_result(response: Any) -> tuple[int, int] | None:
    """Provider-reported (input, output) tokens, or None if the result has no usage."""
    found = False
    in_tok = 0
    out_tok = 0
    for group in getattr(response, "generations", None) or []:
        for gen in group:
            msg = getattr(gen, "message", None)
            meta = getattr(msg, "usage_metadata", None) if msg is not None else None
            if not isinstance(meta, dict):
                continue
            if "input_tokens" not in meta and "output_tokens" not in meta:
                continue
            in_tok += int(meta.get("input_tokens") or 0)
            out_tok += int(meta.get("output_tokens") or 0)
            found = True
    if found:
        return in_tok, out_tok
    extra = getattr(response, "llm_output", None) or {}
    if not isinstance(extra, dict):
        return None
    raw = extra.get("token_usage") or extra.get("usage") or {}
    if not isinstance(raw, dict):
        return None
    if (
        raw.get("prompt_tokens") is None
        and raw.get("input_tokens") is None
        and raw.get("completion_tokens") is None
        and raw.get("output_tokens") is None
    ):
        return None
    return (
        int(raw.get("prompt_tokens") or raw.get("input_tokens") or 0),
        int(raw.get("completion_tokens") or raw.get("output_tokens") or 0),
    )


def format_usage_emit(label: str, input_tokens: int, output_tokens: int) -> str:
    return f"{label}|in={input_tokens}|out={output_tokens}"


def parse_usage_emit(text: str) -> tuple[str, int, int] | None:
    raw = (text or "").strip()
    if "|in=" not in raw or "|out=" not in raw:
        return None
    try:
        head, out_s = raw.rsplit("|out=", 1)
        label, in_s = head.rsplit("|in=", 1)
    except ValueError:
        return None
    try:
        return (label.strip() or "unknown", int(in_s), int(out_s))
    except ValueError:
        return None


# Same budget as ModelRetryMiddleware: 1 try + max_retries=3.
_TRY_BUDGET = 4
# Same budget as ToolRetryMiddleware: 1 try + max_retries=2.
_TOOL_TRY_BUDGET = 3


class NoteLLMError(BaseCallbackHandler):
    """Attach to ChatXAI / ChatOllama. Stores the last on_llm_error on this object."""

    _live: WeakSet[NoteLLMError] = WeakSet()

    def __init__(self, label: str, reasoning_effort: str = "") -> None:
        self.label = label
        self.reasoning_effort = reasoning_effort
        self.last_error: BaseException | None = None
        self.attempts = 0
        NoteLLMError._live.add(self)

    def llm_activity_label(self) -> str:
        if self.reasoning_effort:
            return f"{self.label} ({self.reasoning_effort})"
        return self.label

    def on_chat_model_start(
        self, serialized: dict[str, Any], messages: list[Any], **kwargs: object
    ) -> None:
        # Real generation start (retries, fallback, specialist, analyze). Not status text.
        del serialized, messages, kwargs
        emit("llm", self.llm_activity_label())

    def on_llm_end(self, response: Any, **kwargs: object) -> None:
        # LangChain dispatches on_llm_end (there is no on_chat_model_end hook).
        del kwargs
        usage = usage_from_llm_result(response)
        if usage is None:
            return
        inn, out = usage
        emit("usage", format_usage_emit(self.llm_activity_label(), inn, out))

    def on_llm_error(self, error: BaseException, **_kwargs: object) -> None:
        self.last_error = error
        self.attempts += 1
        self._emit_status()

    def reset(self) -> None:
        self.last_error = None
        self.attempts = 0

    def _peer_already_failed(self) -> bool:
        return any(other is not self and other.attempts > 0 for other in NoteLLMError._live)

    def _emit_status(self) -> None:
        if self.attempts == 1 and self._peer_already_failed():
            emit("status", f"Falling back to {self.label}…")
        if self.attempts < _TRY_BUDGET:
            emit("status", f"Retrying {self.label} ({self.attempts + 1}/{_TRY_BUDGET})…")
            return
        emit("status", f"{self.label} failed after {_TRY_BUDGET} tries")


class NoteToolError(BaseCallbackHandler):
    """Log Retrying {tool} (n/3)… on ToolTransientError (same idea as NoteLLMError)."""

    run_inline = True

    def __init__(self) -> None:
        self.label = "tool"
        self.last_error: BaseException | None = None
        self.attempts = 0
        self._lock = threading.Lock()
        self._names: dict[str, str] = {}
        self._attempts: dict[str, int] = {}

    def on_tool_start(self, serialized: dict[str, Any], input_str: str, **kwargs: object) -> None:  # noqa: ARG002
        name = ""
        if isinstance(serialized, dict):
            name = str(serialized.get("name") or "")
        if not name:
            name = str(kwargs.get("name") or "tool")
        key = str(kwargs.get("tool_call_id") or name)
        with self._lock:
            self._names[key] = name
            self._attempts.setdefault(key, 0)
            self.label = name

    def on_tool_error(self, error: BaseException, **kwargs: object) -> None:
        if not isinstance(error, ToolTransientError):
            return
        key = str(kwargs.get("tool_call_id") or self.label)
        with self._lock:
            name = self._names.get(key, self.label)
            n = self._attempts.get(key, 0) + 1
            self._attempts[key] = n
            self.label = name
            self.attempts = n
            self.last_error = error
        self._emit_status(name, n)

    def reset(self) -> None:
        with self._lock:
            self.last_error = None
            self.attempts = 0
            self.label = "tool"
            self._names.clear()
            self._attempts.clear()

    def _emit_status(self, name: str, attempts: int) -> None:
        if attempts < _TOOL_TRY_BUDGET:
            # Same as llm retries: each new invoke is a start Activity should count.
            emit("tool_retry", name)
            emit("status", f"Retrying {name} ({attempts + 1}/{_TOOL_TRY_BUDGET})…")
            return
        emit("status", f"{name} failed after {_TOOL_TRY_BUDGET} tries")


def note_handlers_on(*models: object) -> list[NoteLLMError]:
    found: list[NoteLLMError] = []
    for model in models:
        cbs = getattr(model, "callbacks", None) or []
        found.extend(cb for cb in cbs if isinstance(cb, NoteLLMError))
    return found


def format_llm_failures(handlers: list[NoteLLMError]) -> str | None:
    items = [(h.label, h.last_error) for h in handlers if h.last_error is not None]
    if len(items) < 2:
        return None
    (p_label, p_exc), (f_label, f_exc) = items[0], items[1]
    return (
        f"primary {p_label}: {type(p_exc).__name__}: {p_exc}\n"
        f"fallback {f_label}: {type(f_exc).__name__}: {f_exc}"
    )
