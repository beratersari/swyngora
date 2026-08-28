"""Remember the last error on each chat model so a dual failure can name both."""

from __future__ import annotations

from weakref import WeakSet

from langchain_core.callbacks import BaseCallbackHandler

from swyngora_ai.progress import emit

# Same budget as ModelRetryMiddleware: 1 try + max_retries=3.
_TRY_BUDGET = 4


class NoteLLMError(BaseCallbackHandler):
    """Attach to ChatXAI / ChatOllama. Stores the last on_llm_error on this object."""

    _live: WeakSet[NoteLLMError] = WeakSet()

    def __init__(self, label: str) -> None:
        self.label = label
        self.last_error: BaseException | None = None
        self.attempts = 0
        NoteLLMError._live.add(self)

    def on_llm_error(self, error: BaseException, **kwargs: object) -> None:
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
