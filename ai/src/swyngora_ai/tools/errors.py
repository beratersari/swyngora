"""Tool-layer exceptions shared by HTTP helpers and retry middleware."""

from __future__ import annotations


class ToolTransientError(Exception):
    """Retryable market-tool HTTP failure (timeout, connect, 429, 5xx)."""
