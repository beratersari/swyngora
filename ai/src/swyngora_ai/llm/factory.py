"""LLM provider factory — Ollama (local) and Grok (xAI) only."""

from __future__ import annotations

from langchain_core.language_models.chat_models import BaseChatModel

from swyngora_ai.config import Settings, get_settings


def _build_grok(cfg: Settings) -> BaseChatModel:
    from langchain_xai import ChatXAI

    return ChatXAI(
        model=cfg.grok_model,
        api_key=cfg.xai_api_key,
        temperature=cfg.temperature,
        extra_body={"reasoning_effort": cfg.grok_reasoning_effort},
        max_retries=0,
    )


def _build_ollama(cfg: Settings) -> BaseChatModel:
    from langchain_ollama import ChatOllama

    return ChatOllama(
        model=cfg.ollama_model,
        base_url=cfg.ollama_base_url,
        temperature=cfg.temperature,
    )


def build_chat_model(settings: Settings | None = None) -> BaseChatModel:
    """Return a chat model based on AI_LLM_PROVIDER."""
    cfg = settings or get_settings()
    if cfg.llm_provider == "grok":
        if not cfg.xai_api_key:
            raise ValueError(
                "AI_LLM_PROVIDER=grok requires XAI_API_KEY. "
                "Or set AI_LLM_PROVIDER=ollama for local inference."
            )
        return _build_grok(cfg)
    return _build_ollama(cfg)


def build_fallback_chat_model(settings: Settings | None = None) -> BaseChatModel | None:
    """The other allowed provider, or None when it cannot be built."""
    cfg = settings or get_settings()
    if cfg.llm_provider == "grok":
        return _build_ollama(cfg)
    if not cfg.xai_api_key:
        return None
    return _build_grok(cfg)
