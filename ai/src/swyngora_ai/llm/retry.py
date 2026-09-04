"""Shared create_agent middleware: retry the primary, then the other provider."""

from __future__ import annotations

from typing import Any

from langchain.agents.middleware import (
    ModelCallLimitMiddleware,
    ModelFallbackMiddleware,
    ModelRetryMiddleware,
    ToolRetryMiddleware,
)

from swyngora_ai.config import Settings, get_settings
from swyngora_ai.llm.factory import build_fallback_chat_model
from swyngora_ai.tools.errors import ToolTransientError


def format_tool_retry_failure(exc: Exception) -> str:
    """Keep fail-soft ERROR prefix so progress and grounding treat this as a tool failure."""
    text = str(exc).strip() or type(exc).__name__
    if text.startswith("ERROR"):
        return text[:500]
    return f"ERROR {text}"[:500]


MODEL_RETRY = ModelRetryMiddleware(
    max_retries=3,
    backoff_factor=2.0,
    initial_delay=1.0,
)

TOOL_RETRY = ToolRetryMiddleware(
    max_retries=2,
    retry_on=(ToolTransientError,),
    on_failure=format_tool_retry_failure,
    backoff_factor=2.0,
    initial_delay=1.0,
)

# Per-invoke ReAct model steps (orchestrator and each specialist). Not AI_MAX_ITERATIONS.
MODEL_CALL_LIMIT_RUN = 6
MODEL_CALL_LIMIT_PREFIX = "Model call limits exceeded"

MODEL_CALL_LIMIT = ModelCallLimitMiddleware(
    run_limit=MODEL_CALL_LIMIT_RUN,
    exit_behavior="end",
)


def is_model_call_limit_text(text: str) -> bool:
    """True when ModelCallLimitMiddleware ended the run with its stock message."""
    return (text or "").lstrip().startswith(MODEL_CALL_LIMIT_PREFIX)


def default_agent_middleware() -> list[Any]:
    """Retry + tool retry + model-step cap (injected model / no fallback)."""
    return [MODEL_RETRY, TOOL_RETRY, MODEL_CALL_LIMIT]


def build_agent_middleware(
    settings: Settings | None = None,
) -> list[Any]:
    """Retry the primary model, then fall back to the other provider if it exists."""
    cfg = settings or get_settings()
    other = build_fallback_chat_model(cfg)
    if other is None:
        return default_agent_middleware()
    return [
        ModelFallbackMiddleware(other),
        ModelRetryMiddleware(
            max_retries=MODEL_RETRY.max_retries,
            backoff_factor=MODEL_RETRY.backoff_factor,
            initial_delay=MODEL_RETRY.initial_delay,
            on_failure="error",
        ),
        TOOL_RETRY,
        MODEL_CALL_LIMIT,
    ]
