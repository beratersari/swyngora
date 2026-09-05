"""langgraph.json registers the orchestrator for local Agent Chat UI."""

from __future__ import annotations

import json
from pathlib import Path

from langchain_core.messages import AIMessage

from swyngora_ai.config import Settings
from swyngora_ai.graph.orchestrator import Orchestrator, SessionMemory
from test_specialists_and_orchestrator import ScriptedModel

_AI_ROOT = Path(__file__).resolve().parents[1]


def test_langgraph_json_registers_swyngora() -> None:
    spec = json.loads((_AI_ROOT / "langgraph.json").read_text(encoding="utf-8"))
    assert spec["graphs"]["swyngora"] == "./src/swyngora_ai/graph/chat_graph.py:graph"
    assert spec["env"] == ".env"
    assert spec["dependencies"] == ["."]


def test_orchestrator_exposes_compiled_graph() -> None:
    orch = Orchestrator(
        settings=Settings(_env_file=None),
        memory=SessionMemory(),
        model=ScriptedModel(responses=[AIMessage(content="ok. Not financial advice.")]),
    )
    assert orch.graph is orch._graph
    assert hasattr(orch.graph, "stream_events")
