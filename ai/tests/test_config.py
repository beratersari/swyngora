import os

import pytest
from pydantic import ValidationError

from swyngora_ai.config import Settings, load_env_files
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


def test_load_env_files_sets_unset_keys(tmp_path, monkeypatch) -> None:
    monkeypatch.delenv("LANGSMITH_API_KEY", raising=False)
    monkeypatch.delenv("LANGSMITH_PROJECT", raising=False)
    monkeypatch.delenv("LANGSMITH_TRACING", raising=False)
    monkeypatch.delenv("LANGSMITH_ENDPOINT", raising=False)
    env = tmp_path / ".env"
    env.write_text(
        "LANGSMITH_API_KEY=lsv2_test\n"
        "LANGSMITH_PROJECT=Swyngora\n"
        "LANGSMITH_TRACING=true\n"
        "LANGSMITH_ENDPOINT=https://apac.api.smith.langchain.com\n"
    )
    load_env_files(env)
    assert os.environ["LANGSMITH_API_KEY"] == "lsv2_test"
    assert os.environ["LANGSMITH_PROJECT"] == "Swyngora"
    assert os.environ["LANGSMITH_TRACING"] == "true"
    assert os.environ["LANGSMITH_ENDPOINT"] == "https://apac.api.smith.langchain.com"


def test_load_env_files_does_not_override_shell(tmp_path, monkeypatch) -> None:
    monkeypatch.setenv("LANGSMITH_API_KEY", "from-shell")
    env = tmp_path / ".env"
    env.write_text("LANGSMITH_API_KEY=from-file\n")
    load_env_files(env)
    assert os.environ["LANGSMITH_API_KEY"] == "from-shell"
