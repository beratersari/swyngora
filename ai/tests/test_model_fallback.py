"""Primary retries first, then the other of ChatXAI ↔ ChatOllama."""

from __future__ import annotations

from typing import Any

import pytest
from langchain.agents import create_agent
from langchain.agents.middleware import ModelFallbackMiddleware, ModelRetryMiddleware
from langchain_core.language_models.chat_models import BaseChatModel
from langchain_core.messages import AIMessage, HumanMessage
from langchain_core.outputs import ChatGeneration, ChatResult

from swyngora_ai.config import Settings
from swyngora_ai.graph.orchestrator import Orchestrator, SessionMemory
from swyngora_ai.llm.retry import MODEL_RETRY, build_agent_middleware
from test_specialists_and_orchestrator import ScriptedModel


class _CountingModel(BaseChatModel):
    calls: int = 0
    error: Exception | None = TimeoutError("down")
    reply: str = "ok"

    @property
    def _llm_type(self) -> str:
        return "counting"

    def _generate(
        self,
        messages: list[Any],
        stop: list[str] | None = None,
        run_manager: Any = None,
        **kwargs: Any,
    ) -> ChatResult:
        self.calls += 1
        if self.error is not None:
            raise self.error
        return ChatResult(generations=[ChatGeneration(message=AIMessage(content=self.reply))])

    def bind_tools(self, tools: Any, **kwargs: Any) -> _CountingModel:
        return self


class _FlakyThenOk(_CountingModel):
    fails_left: int = 0

    def _generate(
        self,
        messages: list[Any],
        stop: list[str] | None = None,
        run_manager: Any = None,
        **kwargs: Any,
    ) -> ChatResult:
        self.calls += 1
        if self.fails_left > 0:
            self.fails_left -= 1
            raise self.error or TimeoutError("blip")
        return ChatResult(generations=[ChatGeneration(message=AIMessage(content=self.reply))])


def _retry_then_fallback(other: BaseChatModel) -> list[Any]:
    return [
        ModelFallbackMiddleware(other),
        ModelRetryMiddleware(
            max_retries=3,
            backoff_factor=2.0,
            initial_delay=0.0,
            on_failure="error",
        ),
    ]


def test_no_fallback_without_grok_key(monkeypatch) -> None:
    monkeypatch.setenv("AI_LLM_PROVIDER", "ollama")
    monkeypatch.setenv("XAI_API_KEY", "")
    mw = build_agent_middleware(Settings(_env_file=None))
    assert mw == [MODEL_RETRY]


def test_ollama_primary_adds_grok_fallback(monkeypatch) -> None:
    captured: dict[str, object] = {}

    def fake_chat_xai(**kwargs: object) -> object:
        captured.update(kwargs)
        return object()

    monkeypatch.setattr("langchain_xai.ChatXAI", fake_chat_xai)
    monkeypatch.setenv("AI_LLM_PROVIDER", "ollama")
    monkeypatch.setenv("XAI_API_KEY", "test-key")
    mw = build_agent_middleware(Settings(_env_file=None))
    assert len(mw) == 2
    assert isinstance(mw[0], ModelFallbackMiddleware)
    assert isinstance(mw[1], ModelRetryMiddleware)
    assert mw[1].on_failure == "error"
    assert mw[1].max_retries == MODEL_RETRY.max_retries
    assert captured["max_retries"] == 0


def test_grok_primary_adds_ollama_fallback(monkeypatch) -> None:
    ollama_kwargs: dict[str, object] = {}

    def fake_chat_ollama(**kwargs: object) -> object:
        ollama_kwargs.update(kwargs)
        return object()

    monkeypatch.setattr("langchain_ollama.ChatOllama", fake_chat_ollama)
    monkeypatch.setenv("AI_LLM_PROVIDER", "grok")
    monkeypatch.setenv("XAI_API_KEY", "test-key")
    mw = build_agent_middleware(Settings(_env_file=None))
    assert isinstance(mw[0], ModelFallbackMiddleware)
    assert isinstance(mw[1], ModelRetryMiddleware)
    assert mw[1].on_failure == "error"
    assert "model" in ollama_kwargs


def test_primary_retries_then_fallback_answers() -> None:
    primary = _CountingModel(error=TimeoutError("primary-down"))
    other = _CountingModel(error=None, reply="from-fallback")
    graph = create_agent(primary, [], system_prompt="t", middleware=_retry_then_fallback(other))
    result = graph.invoke({"messages": [HumanMessage(content="q")]}, {"recursion_limit": 4})
    texts = [str(getattr(m, "content", "")) for m in list((result or {}).get("messages") or [])]
    assert any("from-fallback" in t for t in texts)
    assert primary.calls == 4
    assert other.calls == 1


def test_primary_recovers_before_fallback() -> None:
    primary = _FlakyThenOk(fails_left=2, error=TimeoutError("blip"), reply="from-primary")
    other = _CountingModel(error=None, reply="from-fallback")
    graph = create_agent(primary, [], system_prompt="t", middleware=_retry_then_fallback(other))
    result = graph.invoke({"messages": [HumanMessage(content="q")]}, {"recursion_limit": 4})
    texts = [str(getattr(m, "content", "")) for m in list((result or {}).get("messages") or [])]
    assert any("from-primary" in t for t in texts)
    assert other.calls == 0


def test_both_providers_exhausted_raises() -> None:
    primary = _CountingModel(error=TimeoutError("primary-down"))
    other = _CountingModel(error=TimeoutError("fallback-down"))
    graph = create_agent(primary, [], system_prompt="t", middleware=_retry_then_fallback(other))
    with pytest.raises(TimeoutError, match="fallback-down"):
        graph.invoke({"messages": [HumanMessage(content="q")]}, {"recursion_limit": 4})
    assert primary.calls == 4
    assert other.calls == 4


def test_injected_orchestrator_model_skips_fallback(monkeypatch) -> None:
    constructed: list[object] = []

    def fake_chat_xai(**kwargs: object) -> object:
        constructed.append(kwargs)
        return object()

    monkeypatch.setattr("langchain_xai.ChatXAI", fake_chat_xai)
    monkeypatch.setenv("AI_LLM_PROVIDER", "ollama")
    monkeypatch.setenv("XAI_API_KEY", "test-key")
    orch = Orchestrator(
        settings=Settings(_env_file=None),
        memory=SessionMemory(),
        model=ScriptedModel(responses=[AIMessage(content="scripted. Not financial advice.")]),
    )
    assert constructed == []
    assert orch._middleware == [MODEL_RETRY]
