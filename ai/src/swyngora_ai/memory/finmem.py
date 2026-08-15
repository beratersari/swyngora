"""Thread-safe working memory plus optional SQLite FinMem layers."""

from __future__ import annotations

import sqlite3
import threading
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from langchain_core.messages import AIMessage, BaseMessage, HumanMessage

from swyngora_ai.constants import TAPE_TTL_SECONDS


def memory_key(client_id: str, session_id: str) -> str:
    """Namespace conversation memory by tenant so sessionId cannot cross clients."""
    cid = (client_id or "").strip() or "_"
    sid = (session_id or "default").strip() or "default"
    return f"{cid}\n{sid}"


@dataclass
class SessionMemory:
    """Thread-safe per-session message buffer. Keys should be memory_key(...)."""

    sessions: dict[str, list[BaseMessage]] = field(default_factory=dict)
    max_messages: int = 40
    _lock: threading.Lock = field(default_factory=threading.Lock, repr=False)

    def get(self, session_id: str) -> list[BaseMessage]:
        with self._lock:
            return list(self.sessions.get(session_id, []))

    def append(self, session_id: str, messages: list[BaseMessage]) -> None:
        with self._lock:
            buf = self.sessions.setdefault(session_id, [])
            buf.extend(messages)
            if len(buf) > self.max_messages:
                self.sessions[session_id] = buf[-self.max_messages :]

    def clear(self, session_id: str) -> None:
        with self._lock:
            self.sessions.pop(session_id, None)


class FinMem(SessionMemory):
    """Working (RAM) + daily notes + long-term facts. SQLite when path is set."""

    def __init__(
        self,
        path: str = "",
        max_messages: int = 40,
        tape_ttl_seconds: int = TAPE_TTL_SECONDS,
    ) -> None:
        super().__init__(max_messages=max_messages)
        self.tape_ttl_seconds = tape_ttl_seconds
        self.path = (path or "").strip()
        self._db: sqlite3.Connection | None = None
        self._db_lock = threading.Lock()
        if self.path:
            self._open()

    def _open(self) -> None:
        if self.path == ":memory:":
            self._db = sqlite3.connect(":memory:", check_same_thread=False)
        else:
            p = Path(self.path)
            p.parent.mkdir(parents=True, exist_ok=True)
            self._db = sqlite3.connect(str(p), check_same_thread=False)
        self._db.execute(
            """
            CREATE TABLE IF NOT EXISTS daily_notes (
                client_id TEXT NOT NULL,
                day TEXT NOT NULL,
                summary TEXT NOT NULL,
                PRIMARY KEY (client_id, day)
            )
            """
        )
        self._db.execute(
            """
            CREATE TABLE IF NOT EXISTS facts (
                client_id TEXT NOT NULL,
                key TEXT NOT NULL,
                value TEXT NOT NULL,
                updated_at TEXT NOT NULL,
                PRIMARY KEY (client_id, key)
            )
            """
        )
        self._db.commit()

    def set_fact(self, client_id: str, key: str, value: str) -> None:
        cid = (client_id or "").strip() or "_"
        if self._db is None:
            return
        now = datetime.now(UTC).isoformat()
        with self._db_lock:
            self._db.execute(
                """
                INSERT INTO facts(client_id, key, value, updated_at)
                VALUES (?, ?, ?, ?)
                ON CONFLICT(client_id, key) DO UPDATE SET
                    value=excluded.value, updated_at=excluded.updated_at
                """,
                (cid, key, value, now),
            )
            self._db.commit()

    def get_facts(self, client_id: str) -> dict[str, str]:
        cid = (client_id or "").strip() or "_"
        if self._db is None:
            return {}
        with self._db_lock:
            rows = self._db.execute(
                "SELECT key, value FROM facts WHERE client_id=?",
                (cid,),
            ).fetchall()
        return {str(k): str(v) for k, v in rows}

    def note_tape(self, client_id: str, symbol: str = "", exchange: str = "") -> None:
        now = datetime.now(UTC).isoformat()
        self.set_fact(client_id, "last_tape_at", now)
        if symbol:
            self.set_fact(client_id, "last_symbol", symbol)
        if exchange:
            self.set_fact(client_id, "last_exchange", exchange)

    def tape_is_stale(self, client_id: str) -> bool:
        facts = self.get_facts(client_id)
        raw = facts.get("last_tape_at") or ""
        if not raw:
            return True
        try:
            ts = datetime.fromisoformat(raw)
            if ts.tzinfo is None:
                ts = ts.replace(tzinfo=UTC)
        except ValueError:
            return True
        age = (datetime.now(UTC) - ts).total_seconds()
        return age > self.tape_ttl_seconds

    def append_daily(self, client_id: str, text: str) -> None:
        cid = (client_id or "").strip() or "_"
        if self._db is None or not text.strip():
            return
        day = datetime.now(UTC).date().isoformat()
        clip = text.strip()[:400]
        with self._db_lock:
            row = self._db.execute(
                "SELECT summary FROM daily_notes WHERE client_id=? AND day=?",
                (cid, day),
            ).fetchone()
            prev = (row[0] if row else "").strip()
            merged = f"{prev}\n• {clip}".strip() if prev else f"• {clip}"
            merged = merged[-2000:]
            self._db.execute(
                """
                INSERT INTO daily_notes(client_id, day, summary)
                VALUES (?, ?, ?)
                ON CONFLICT(client_id, day) DO UPDATE SET summary=excluded.summary
                """,
                (cid, day, merged),
            )
            self._db.commit()

    def recent_daily(self, client_id: str, days: int = 3) -> list[str]:
        cid = (client_id or "").strip() or "_"
        if self._db is None:
            return []
        with self._db_lock:
            rows = self._db.execute(
                """
                SELECT day, summary FROM daily_notes
                WHERE client_id=? ORDER BY day DESC LIMIT ?
                """,
                (cid, days),
            ).fetchall()
        return [f"{d}: {s}" for d, s in rows]

    def context_block(self, client_id: str) -> str:
        facts = self.get_facts(client_id)
        daily = self.recent_daily(client_id)
        parts: list[str] = []
        if facts:
            bits = [f"{k}={v}" for k, v in facts.items()]
            parts.append("Long-term facts: " + "; ".join(bits))
        if daily:
            parts.append("Recent daily notes:\n" + "\n".join(daily))
        return "\n".join(parts)

    def persist_turn(self, client_id: str, user: str, reply: str) -> None:
        self.append_daily(client_id, f"Q: {user[:160]} → {reply[:200]}")


def as_human_ai(user: str, reply: str) -> list[BaseMessage]:
    return [HumanMessage(content=user), AIMessage(content=reply)]


def finmem_from_settings(path: str = "", max_messages: int = 40) -> FinMem:
    return FinMem(path=path, max_messages=max_messages)


def message_text(msg: Any) -> str:
    content = getattr(msg, "content", "")
    if isinstance(content, str):
        return content
    return str(content)
