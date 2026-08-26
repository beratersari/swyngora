"""Shared create_agent middleware: retry the primary, then the other provider."""

from __future__ import annotations

from langchain.agents.middleware import ModelFallbackMiddleware, ModelRetryMiddleware

from swyngora_ai.config import Settings, get_settings
from swyngora_ai.llm.factory import build_fallback_chat_model

MODEL_RETRY = ModelRetryMiddleware(
    max_retries=3,
    backoff_factor=2.0,
    initial_delay=1.0,
)


def build_agent_middleware(
    settings: Settings | None = None,
) -> list[ModelRetryMiddleware | ModelFallbackMiddleware]:
    """Retry the primary model, then fall back to the other provider if it exists."""
    cfg = settings or get_settings()
    other = build_fallback_chat_model(cfg)
    if other is None:
        return [MODEL_RETRY]
    return [
        ModelFallbackMiddleware(other),
        ModelRetryMiddleware(
            max_retries=MODEL_RETRY.max_retries,
            backoff_factor=MODEL_RETRY.backoff_factor,
            initial_delay=MODEL_RETRY.initial_delay,
            on_failure="error",
        ),
    ]
