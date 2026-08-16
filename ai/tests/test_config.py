import pytest
from pydantic import ValidationError

from swyngora_ai.config import Settings
from swyngora_ai.constants import DEFAULT_GROK_MODEL, DEFAULT_GROK_REASONING_EFFORT


def test_settings_defaults(monkeypatch):
    monkeypatch.delenv("AI_LLM_PROVIDER", raising=False)
    monkeypatch.delenv("XAI_API_KEY", raising=False)
    monkeypatch.delenv("GROK_MODEL", raising=False)
    monkeypatch.delenv("GROK_REASONING_EFFORT", raising=False)
    s = Settings(_env_file=None)
    assert s.llm_provider == "ollama"
    assert "8080" in s.api_base_url
    assert s.default_client_id
    assert s.ollama_model == "qwen2.5"
    assert s.grok_model == DEFAULT_GROK_MODEL
    assert s.grok_model == "grok-4.3"
    assert s.grok_reasoning_effort == DEFAULT_GROK_REASONING_EFFORT
    assert s.grok_reasoning_effort == "low"
    assert s.service_token == ""
    assert s.memory_path == ""


def test_grok_reasoning_effort_override() -> None:
    s = Settings(_env_file=None, grok_reasoning_effort="medium")
    assert s.grok_reasoning_effort == "medium"


def test_grok_reasoning_effort_normalizes_case() -> None:
    s = Settings(_env_file=None, grok_reasoning_effort=" HIGH ")
    assert s.grok_reasoning_effort == "high"


def test_grok_reasoning_effort_rejects_unknown() -> None:
    with pytest.raises(ValidationError):
        Settings(_env_file=None, grok_reasoning_effort="turbo")
