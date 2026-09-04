"""Configuration for the AI package (env-driven; no secrets in code)."""

from __future__ import annotations

from functools import lru_cache
from pathlib import Path
from typing import Literal

from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict

from swyngora_ai.constants import (
    DEFAULT_GROK_MODEL,
    DEFAULT_GROK_REASONING_EFFORT,
    DEFAULT_OLLAMA_MODEL,
)

GrokReasoningEffort = Literal["none", "low", "medium", "high"]

# Same files Settings reads. Order: first file that sets a key wins for os.environ
# (load_env_files uses override=False). Shell exports always win.
ENV_FILES = (".env", "ai/.env", "../.env", "../backend/.env", "backend/.env")


def _default_env_paths() -> tuple[Path, ...]:
    """Prefer ai/.env next to this package so cwd does not matter."""
    ai_root = Path(__file__).resolve().parents[2]
    repo = ai_root.parent
    cwd = Path.cwd()
    return (
        ai_root / ".env",
        cwd / ".env",
        repo / ".env",
        repo / "backend" / ".env",
        cwd / "ai" / ".env",
        cwd / "backend" / ".env",
    )


def load_env_files(*paths: str | Path) -> None:
    """Copy env files into os.environ without overriding variables already set.

    With no args, loads ai/.env (package dir) first, then cwd / repo / backend.
    Pass explicit paths in tests.
    """
    from dotenv import load_dotenv

    seen: set[Path] = set()
    for raw in paths or _default_env_paths():
        path = Path(raw)
        resolved = path.resolve()
        if resolved in seen or not path.is_file():
            continue
        seen.add(resolved)
        load_dotenv(path, override=False)


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=ENV_FILES,
        env_file_encoding="utf-8",
        extra="ignore",
    )

    # LLM: ollama (local default) or grok (xAI). AGENTS.md §6.5 — only these two.
    llm_provider: Literal["ollama", "grok"] = Field(default="ollama", alias="AI_LLM_PROVIDER")
    ollama_base_url: str = Field(default="http://127.0.0.1:11434", alias="OLLAMA_BASE_URL")
    ollama_model: str = Field(default=DEFAULT_OLLAMA_MODEL, alias="OLLAMA_MODEL")
    xai_api_key: str = Field(default="", alias="XAI_API_KEY")
    grok_model: str = Field(default=DEFAULT_GROK_MODEL, alias="GROK_MODEL")
    grok_reasoning_effort: GrokReasoningEffort = Field(
        default=DEFAULT_GROK_REASONING_EFFORT,
        alias="GROK_REASONING_EFFORT",
    )
    # Shared secret with the Go AI proxy. Empty = open localhost (dev).
    service_token: str = Field(default="", alias="AI_SERVICE_TOKEN")
    # FinMem SQLite path. Empty = RAM only. Use data/ai-memory.db or :memory:.
    memory_path: str = Field(default="", alias="AI_MEMORY_PATH")

    # Backend + MCP
    api_base_url: str = Field(default="http://localhost:8080", alias="SWYNGORA_API_URL")
    api_auth_token: str = Field(default="", alias="SWYNGORA_API_TOKEN")
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

    @field_validator("grok_reasoning_effort", mode="before")
    @classmethod
    def _normalize_grok_reasoning_effort(cls, value: object) -> object:
        if isinstance(value, str):
            return value.strip().lower()
        return value


@lru_cache
def get_settings() -> Settings:
    return Settings()
