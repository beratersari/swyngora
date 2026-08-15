from swyngora_ai.config import Settings


def test_settings_defaults(monkeypatch):
    monkeypatch.delenv("AI_LLM_PROVIDER", raising=False)
    monkeypatch.delenv("XAI_API_KEY", raising=False)
    s = Settings(_env_file=None)
    assert s.llm_provider == "ollama"
    assert "8080" in s.api_base_url
    assert s.default_client_id
    assert s.ollama_model == "qwen2.5"
    assert s.service_token == ""
    assert s.memory_path == ""
