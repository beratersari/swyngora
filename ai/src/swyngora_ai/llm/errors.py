"""Remember the last error on each chat model so a dual failure can name both."""

from __future__ import annotations

from langchain_core.callbacks import BaseCallbackHandler


class NoteLLMError(BaseCallbackHandler):
    """Attach to ChatXAI / ChatOllama. Stores the last on_llm_error on this object."""

    def __init__(self, label: str) -> None:
        self.label = label
        self.last_error: BaseException | None = None

    def on_llm_error(self, error: BaseException, **kwargs: object) -> None:
        self.last_error = error

    def reset(self) -> None:
        self.last_error = None


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
