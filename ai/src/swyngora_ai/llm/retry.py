"""Shared ModelRetryMiddleware for create_agent (Grok and Ollama)."""

from langchain.agents.middleware import ModelRetryMiddleware

MODEL_RETRY = ModelRetryMiddleware(
    max_retries=3,
    backoff_factor=2.0,
    initial_delay=1.0,
)
