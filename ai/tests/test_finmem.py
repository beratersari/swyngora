from langchain_core.messages import AIMessage, HumanMessage

from swyngora_ai.config import Settings
from swyngora_ai.graph.orchestrator import Orchestrator, SessionMemory, memory_key
from swyngora_ai.memory.finmem import FinMem
from test_specialists_and_orchestrator import ScriptedModel


def test_reset_uses_tenant_memory_key():
    mem = SessionMemory()
    cid = "alice"
    sid = "desk"
    mem.append(memory_key(cid, sid), [HumanMessage(content="secret"), AIMessage(content="ok")])
    orch = Orchestrator(
        settings=Settings(_env_file=None, default_client_id=cid),
        memory=mem,
        model=ScriptedModel(responses=[AIMessage(content="hi. Not financial advice.")]),
    )
    # Old bug: reset(sid) left tenant-namespaced history in place.
    orch.reset(sid)
    assert mem.get(sid) == []
    orch.reset(sid, client_id=cid)
    assert mem.get(memory_key(cid, sid)) == []


def test_finmem_sqlite_facts_and_ttl(tmp_path):
    path = str(tmp_path / "mem.db")
    mem = FinMem(path=path, tape_ttl_seconds=60)
    mem.note_tape("c1", "BTCUSDT", "binance")
    assert not mem.tape_is_stale("c1")
    mem.set_fact("c1", "last_tape_at", "2000-01-01T00:00:00+00:00")
    assert mem.tape_is_stale("c1")
    mem.persist_turn("c1", "price of btc?", "last 100")
    daily = mem.recent_daily("c1")
    assert daily
    assert "btc" in daily[0].lower()
    block = mem.context_block("c1")
    assert "last_symbol=BTCUSDT" in block
    assert "stale" not in block.lower()


def test_session_memory_is_capped():
    mem = SessionMemory(max_messages=4)
    mem.append("s", [HumanMessage(content="a"), AIMessage(content="b")])
    mem.append("s", [HumanMessage(content="c"), AIMessage(content="d")])
    mem.append("s", [HumanMessage(content="e"), AIMessage(content="f")])
    assert len(mem.get("s")) == 4
