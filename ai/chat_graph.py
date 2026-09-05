"""Compiled orchestrator for `langgraph dev` / Agent Chat UI (local desk only)."""

from __future__ import annotations

from swyngora_ai.config import load_env_files
from swyngora_ai.graph.orchestrator import build_orchestrator

load_env_files()
graph = build_orchestrator().graph
