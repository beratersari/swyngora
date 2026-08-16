"""Shared AI constants (venues, packs, TTLs)."""

from __future__ import annotations

EXCHANGE_VENUES = "binance|coinbase|bybit|nasdaq|bist"
EXCHANGE_VENUES_OR_ALL = f"{EXCHANGE_VENUES}|all"

# Default Ollama model that can actually call tools (llama3.2 often skips them).
DEFAULT_OLLAMA_MODEL = "qwen2.5"

# Cheapest current general-purpose xAI chat model (docs.x.ai/developers/pricing).
# grok-3-mini is retired. grok-build-0.1 is cheaper but coding-only.
DEFAULT_GROK_MODEL = "grok-4.3"

TAPE_TTL_SECONDS = 300

SPECIALIST_TAPE = "market_tape_agent"
SPECIALIST_BOOK = "market_book_agent"
SPECIALIST_PAPER = "paper_desk_agent"
SPECIALIST_ACCOUNT = "account_agent"
SPECIALIST_WEB = "web_agent"
SPECIALIST_X = "x_agent"
SPECIALIST_ANALYST = "analyst_agent"
SPECIALIST_BULL = "bull_agent"
SPECIALIST_BEAR = "bear_agent"
