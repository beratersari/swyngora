"""Remember the last error on each chat model so a dual failure can name both."""

from __future__ import annotations

import threading
from dataclasses import dataclass, fields
from typing import Any
from weakref import WeakSet

from langchain_core.callbacks import BaseCallbackHandler

from swyngora_ai.progress import emit
from swyngora_ai.tools.errors import ToolTransientError


@dataclass
class TokenUsage:
    """Provider-reported tokens. Extra fields stay 0 when the API omits them."""

    input_tokens: int = 0
    output_tokens: int = 0
    cache_read: int = 0
    cache_creation: int = 0
    reasoning: int = 0
    audio_in: int = 0
    audio_out: int = 0

    def add(self, other: TokenUsage) -> None:
        for f in fields(self):
            setattr(self, f.name, int(getattr(self, f.name)) + int(getattr(other, f.name)))

    @property
    def total(self) -> int:
        return self.input_tokens + self.output_tokens


def _as_int(value: object) -> int:
    if isinstance(value, bool):
        return int(value)
    if isinstance(value, int):
        return value
    if isinstance(value, str) and value.strip():
        try:
            return int(value)
        except ValueError:
            return 0
    return 0


def _details_map(raw: object) -> dict[str, Any]:
    if raw is None:
        return {}
    if isinstance(raw, dict):
        return raw
    dump = getattr(raw, "model_dump", None)
    if callable(dump):
        try:
            out = dump()
        except Exception:  # noqa: BLE001
            return {}
        return out if isinstance(out, dict) else {}
    return {}


def token_usage_from_metadata(meta: dict[str, Any]) -> TokenUsage | None:
    if "input_tokens" not in meta and "output_tokens" not in meta:
        return None
    inn = _details_map(meta.get("input_token_details"))
    out = _details_map(meta.get("output_token_details"))
    # LangChain UsageMetadata / InputTokenDetails / OutputTokenDetails only:
    # audio, cache_creation, cache_read, reasoning.
    return TokenUsage(
        input_tokens=_as_int(meta.get("input_tokens")),
        output_tokens=_as_int(meta.get("output_tokens")),
        cache_read=_as_int(inn.get("cache_read")),
        cache_creation=_as_int(inn.get("cache_creation")),
        reasoning=_as_int(out.get("reasoning")),
        audio_in=_as_int(inn.get("audio")),
        audio_out=_as_int(out.get("audio")),
    )


def token_usage_from_openai_usage(raw: dict[str, Any]) -> TokenUsage | None:
    if (
        raw.get("prompt_tokens") is None
        and raw.get("input_tokens") is None
        and raw.get("completion_tokens") is None
        and raw.get("output_tokens") is None
    ):
        return None
    prompt = _details_map(raw.get("prompt_tokens_details"))
    completion = _details_map(raw.get("completion_tokens_details"))
    return TokenUsage(
        input_tokens=_as_int(raw.get("prompt_tokens") or raw.get("input_tokens")),
        output_tokens=_as_int(raw.get("completion_tokens") or raw.get("output_tokens")),
        cache_read=_as_int(prompt.get("cached_tokens") or prompt.get("cache_read")),
        cache_creation=_as_int(prompt.get("cache_creation") or prompt.get("cache_write")),
        reasoning=_as_int(completion.get("reasoning_tokens") or completion.get("reasoning")),
        audio_in=_as_int(prompt.get("audio_tokens") or prompt.get("audio")),
        audio_out=_as_int(completion.get("audio_tokens") or completion.get("audio")),
    )


def usage_from_llm_result(response: Any) -> TokenUsage | None:
    """Provider-reported tokens, or None if the result has no usage."""
    acc: TokenUsage | None = None
    for group in getattr(response, "generations", None) or []:
        for gen in group:
            msg = getattr(gen, "message", None)
            meta = getattr(msg, "usage_metadata", None) if msg is not None else None
            if not isinstance(meta, dict):
                continue
            piece = token_usage_from_metadata(meta)
            if piece is None:
                continue
            if acc is None:
                acc = TokenUsage()
            acc.add(piece)
    if acc is not None:
        return acc
    extra = getattr(response, "llm_output", None) or {}
    if not isinstance(extra, dict):
        return None
    raw = extra.get("token_usage") or extra.get("usage") or {}
    if not isinstance(raw, dict):
        return None
    return token_usage_from_openai_usage(raw)


def format_usage_emit(label: str, usage: TokenUsage) -> str:
    parts = [label, f"in={usage.input_tokens}", f"out={usage.output_tokens}"]
    extras = (
        ("cache_read", usage.cache_read),
        ("cache_write", usage.cache_creation),
        ("reasoning", usage.reasoning),
        ("audio_in", usage.audio_in),
        ("audio_out", usage.audio_out),
    )
    parts.extend(f"{key}={n}" for key, n in extras if n)
    return "|".join(parts)


def parse_usage_emit(text: str) -> tuple[str, TokenUsage] | None:
    raw = (text or "").strip()
    if "|in=" not in raw:
        return None
    label, _, rest = raw.partition("|")
    usage = TokenUsage()
    alias = {
        "in": "input_tokens",
        "out": "output_tokens",
        "cache_write": "cache_creation",
    }
    for chunk in rest.split("|"):
        if "=" not in chunk:
            return None
        key, _, val = chunk.partition("=")
        field = alias.get(key, key)
        if not hasattr(usage, field):
            continue
        try:
            setattr(usage, field, int(val))
        except ValueError:
            return None
    return (label.strip() or "unknown", usage)


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
        emit("usage", format_usage_emit(self.llm_activity_label(), usage))

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
