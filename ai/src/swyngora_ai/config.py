"""Configuration for the AI package (env-driven; no secrets in code)."""

from __future__ import annotations

from functools import lru_cache
from typing import Literal

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=(".env", "ai/.env", "../.env", "../backend/.env", "backend/.env"),
        env_file_encoding="utf-8",
        extra="ignore",
    )

    # LLM: ollama (local default) or grok (xAI). AGENTS.md §6.5 — only these two.
    llm_provider: Literal["ollama", "grok"] = Field(default="ollama", alias="AI_LLM_PROVIDER")
    ollama_base_url: str = Field(default="http://127.0.0.1:11434", alias="OLLAMA_BASE_URL")
    ollama_model: str = Field(default="llama3.2", alias="OLLAMA_MODEL")
    xai_api_key: str = Field(default="", alias="XAI_API_KEY")
    grok_model: str = Field(default="grok-3-mini", alias="GROK_MODEL")

    # Backend + MCP
    api_base_url: str = Field(default="http://localhost:8080", alias="SWYNGORA_API_URL")
    mcp_command: str = Field(
        default="",
        alias="SWYNGORA_MCP_COMMAND",
        description="Optional: e.g. 'go run ./cmd/mcp' from backend/. Empty = HTTP tools only.",
    )
    mcp_cwd: str = Field(default="", alias="SWYNGORA_MCP_CWD")

    # Agent behaviour
    max_agent_iterations: int = Field(default=8, alias="AI_MAX_ITERATIONS")
    temperature: float = Field(default=0.2, alias="AI_TEMPERATURE")
    default_client_id: str = Field(default="ai-assistant", alias="AI_DEFAULT_CLIENT_ID")
    disclaimer: str = (
        "Informational analysis only — not financial advice. "
        "Crypto markets are volatile; verify critical numbers via tools."
    )


@lru_cache
def get_settings() -> Settings:
    return Settings()
