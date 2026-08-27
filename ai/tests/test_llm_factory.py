"""LLM factory wiring (no live Ollama/Grok calls)."""

from __future__ import annotations

from swyngora_ai.config import Settings
from swyngora_ai.llm.errors import NoteLLMError
from swyngora_ai.llm.factory import build_chat_model, build_fallback_chat_model


class _Capture:
    def __init__(self, **kwargs: object) -> None:
        self.kwargs = kwargs


def test_grok_factory_sends_default_reasoning_effort(monkeypatch) -> None:
    captured: dict[str, object] = {}

    def fake_chat_xai(**kwargs: object) -> _Capture:
        captured.update(kwargs)
        return _Capture(**kwargs)

    monkeypatch.setattr("langchain_xai.ChatXAI", fake_chat_xai)
    monkeypatch.setenv("AI_LLM_PROVIDER", "grok")
    monkeypatch.setenv("XAI_API_KEY", "test-key")
    monkeypatch.setenv("GROK_MODEL", "grok-4.3")
    monkeypatch.delenv("GROK_REASONING_EFFORT", raising=False)
    model = build_chat_model(Settings(_env_file=None))
    assert isinstance(model, _Capture)
    assert captured["model"] == "grok-4.3"
    assert captured["extra_body"] == {"reasoning_effort": "low"}
    assert captured["max_retries"] == 0
    cbs = captured["callbacks"]
    assert isinstance(cbs, list)
    assert len(cbs) == 1
    assert isinstance(cbs[0], NoteLLMError)
    assert cbs[0].label == "grok-4.3"


def test_grok_factory_honors_reasoning_override(monkeypatch) -> None:
    captured: dict[str, object] = {}

    def fake_chat_xai(**kwargs: object) -> _Capture:
        captured.update(kwargs)
        return _Capture(**kwargs)

    monkeypatch.setattr("langchain_xai.ChatXAI", fake_chat_xai)
    monkeypatch.setenv("AI_LLM_PROVIDER", "grok")
    monkeypatch.setenv("XAI_API_KEY", "test-key")
    monkeypatch.setenv("GROK_REASONING_EFFORT", "none")
    build_chat_model(Settings(_env_file=None))
    assert captured["extra_body"] == {"reasoning_effort": "none"}


def test_ollama_factory_does_not_set_reasoning(monkeypatch) -> None:
    captured: dict[str, object] = {}

    def fake_chat_ollama(**kwargs: object) -> _Capture:
        captured.update(kwargs)
        return _Capture(**kwargs)

    monkeypatch.setattr("langchain_ollama.ChatOllama", fake_chat_ollama)
    monkeypatch.setenv("AI_LLM_PROVIDER", "ollama")
    monkeypatch.delenv("GROK_REASONING_EFFORT", raising=False)
    build_chat_model(Settings(_env_file=None))
    assert "extra_body" not in captured
    assert "reasoning_effort" not in captured


def test_fallback_none_when_ollama_has_no_key(monkeypatch) -> None:
    monkeypatch.setenv("AI_LLM_PROVIDER", "ollama")
    monkeypatch.setenv("XAI_API_KEY", "")
    assert build_fallback_chat_model(Settings(_env_file=None)) is None


def test_fallback_is_grok_when_primary_is_ollama(monkeypatch) -> None:
    captured: dict[str, object] = {}

    def fake_chat_xai(**kwargs: object) -> _Capture:
        captured.update(kwargs)
        return _Capture(**kwargs)

    monkeypatch.setattr("langchain_xai.ChatXAI", fake_chat_xai)
    monkeypatch.setenv("AI_LLM_PROVIDER", "ollama")
    monkeypatch.setenv("XAI_API_KEY", "test-key")
    model = build_fallback_chat_model(Settings(_env_file=None))
    assert isinstance(model, _Capture)
    assert captured["max_retries"] == 0


def test_fallback_is_ollama_when_primary_is_grok(monkeypatch) -> None:
    captured: dict[str, object] = {}

    def fake_chat_ollama(**kwargs: object) -> _Capture:
        captured.update(kwargs)
        return _Capture(**kwargs)

    monkeypatch.setattr("langchain_ollama.ChatOllama", fake_chat_ollama)
    monkeypatch.setenv("AI_LLM_PROVIDER", "grok")
    monkeypatch.setenv("XAI_API_KEY", "test-key")
    model = build_fallback_chat_model(Settings(_env_file=None))
    assert isinstance(model, _Capture)
    assert "extra_body" not in captured
