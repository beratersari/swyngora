"""Deterministic coin-situation pull — the same read as the coin card, not a paraphrase."""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from typing import Any

from swyngora_ai.config import Settings
from swyngora_ai.progress import emit
from swyngora_ai.sources.identity import find_instruments
from swyngora_ai.tools.market_http import build_market_tools

_MAX_INSTRUMENTS = 3
_MAX_BOOKS = 8

# Clause order matches frontend situationLines. A missing list names venues.
# A missing venue is never written as a side, a flow, or a number.
_CLAUSES: tuple[tuple[str, str, str, str, tuple[str, ...]], ...] = (
    ("price vs futures CVD", "priceCvd", "kind", "priceCvdMissing", ()),
    ("1h open interest", "openInterest", "flow", "openInterestMissing", ()),
    ("estimated liquidation pocket", "liquidation", "side", "liquidationMissing", ("distancePct",)),
    (
        "measured 24h prints",
        "prints",
        "side",
        "printsMissing",
        ("longNotional", "shortNotional"),
    ),
    ("funding payer", "funding", "payer", "fundingMissing", ("ratePct",)),
    ("crowd gap", "crowd", "lean", "crowdMissing", ("gapPct",)),
    ("whale vs retail", "size", "kind", "sizeMissing", ()),
)

_SITUATION_KEYS = (
    {spec[1] for spec in _CLAUSES} | {spec[3] for spec in _CLAUSES} | {"symbol", "asOf"}
)

_HEADER = (
    "Coin situation (Swyngora's read). Lead the answer with these clauses. "
    "A blank clause stays blank: name the missing venues and do not invent a side, a flow, or a number."
)


@dataclass
class SituationEntry:
    """One open paper spot buy that stored a read for this symbol."""

    exchange: str
    situation: dict[str, Any]
    entry_at: str = ""
    book: str = ""

    def as_dict(self) -> dict[str, Any]:
        out: dict[str, Any] = {
            "exchange": self.exchange,
            "entrySituation": self.situation,
        }
        if self.entry_at:
            out["entryAt"] = self.entry_at
        if self.book:
            out["book"] = self.book
        return out


@dataclass
class SituationRead:
    """Live coin read plus any open paper buys for the same symbol."""

    symbol: str
    situation: dict[str, Any]
    entries: list[SituationEntry] = field(default_factory=list)

    def as_dict(self) -> dict[str, Any]:
        out: dict[str, Any] = {"symbol": self.symbol, "situation": self.situation}
        if self.entries:
            out["entries"] = [entry.as_dict() for entry in self.entries]
        return out


def clause_lines(report: dict[str, Any] | None) -> list[str]:
    """One fact line per published clause. A missing venue stays on the missing line."""
    if not isinstance(report, dict):
        return []
    lines: list[str] = []
    for label, obj_key, value_key, missing_key, extras in _CLAUSES:
        obj = report.get(obj_key)
        if isinstance(obj, dict):
            value = obj.get(value_key)
            if value not in (None, ""):
                bits = [str(value)]
                bits.extend(
                    f"{extra} {obj[extra]}"
                    for extra in extras
                    if extra in obj and obj[extra] not in (None, "")
                )
                lines.append(f"- {label}: {', '.join(bits)}")
                continue
        missing = report.get(missing_key)
        if isinstance(missing, list):
            names = [str(item).strip() for item in missing if str(item).strip()]
            if names:
                lines.append(f"- {label}: missing: {', '.join(names)}")
    return lines


def situation_prompt(reads: list[SituationRead]) -> str:
    """Factual block for the model. Empty when this question named no crypto read."""
    if not reads:
        return ""
    parts = [_HEADER]
    for read in reads:
        parts.append(f"\n{read.symbol} live:")
        live = clause_lines(read.situation)
        parts.append("\n".join(live) if live else "- no clause was published")
        for entry in read.entries:
            where = [f"exchange={entry.exchange}"]
            if entry.book:
                where.append(f"book={entry.book}")
            if entry.entry_at:
                where.append(f"entryAt={entry.entry_at}")
            parts.append(
                "\nEntry read stored on the open paper buy (placement time, not now). "
                + " ".join(where)
            )
            stored = clause_lines(entry.situation)
            parts.append("\n".join(stored) if stored else "- no clause was stored")
    return "\n".join(parts)


def prefetch_situations(
    settings: Settings,
    message: str,
    *,
    client_id: str = "",
) -> list[SituationRead]:
    """Fetch get_coin_situation for crypto names, then any open paper entry read.

    Equities and unknown names are skipped. A failed call drops that piece and
    does not invent a clause. Portfolio failure leaves the live read in place
    and says nothing about a position.
    """
    cryptos = [inst for inst in find_instruments(message) if inst.kind == "crypto"][
        :_MAX_INSTRUMENTS
    ]
    if not cryptos:
        return []

    by_name = _tools(settings)
    situation_tool = by_name.get("get_coin_situation")
    if situation_tool is None:
        return []

    positions = _open_positions(by_name, client_id)
    reads: list[SituationRead] = []
    for inst in cryptos:
        emit("status", f"Reading situation {inst.symbol}…")
        emit("tool", f"prefetch → get_coin_situation(symbol={inst.symbol})")
        parsed = _parse_obj(_invoke(situation_tool, {"symbol": inst.symbol}))
        if not _is_situation(parsed):
            emit("tool_error", f"get_coin_situation returned no read for {inst.symbol}")
            continue
        reads.append(
            SituationRead(
                symbol=inst.symbol,
                situation=parsed,
                entries=_entries_for(positions, inst.symbol),
            )
        )
    return reads


def _tools(settings: Settings) -> dict[str, Any]:
    try:
        tools = build_market_tools(settings)
    except Exception as exc:  # noqa: BLE001 — a tool registry failure must not abort the turn
        emit("tool_error", f"situation tools failed: {exc}")
        return {}
    return {tool.name: tool for tool in tools}


def _invoke(tool: Any, args: dict[str, Any]) -> Any:
    try:
        return tool.invoke(args)
    except Exception as exc:  # noqa: BLE001 — one failed prefetch must not abort the turn
        name = getattr(tool, "name", "tool")
        emit("tool_error", f"{name} failed: {exc}")
        return ""


def _as_text(raw: Any) -> str:
    if raw is None:
        return ""
    if isinstance(raw, str):
        return raw
    if isinstance(raw, dict | list):
        try:
            return json.dumps(raw)
        except (TypeError, ValueError):
            return ""
    return str(raw)


def _parse_obj(raw: Any) -> dict[str, Any] | None:
    text = _as_text(raw).strip()
    if not text or text.startswith(("ERROR", "(")):
        return None
    try:
        data = json.loads(text)
    except json.JSONDecodeError:
        return None
    if not isinstance(data, dict) or "error" in data:
        return None
    return data


def _is_situation(data: dict[str, Any] | None) -> bool:
    if not data:
        return False
    return any(key in data for key in _SITUATION_KEYS)


def _open_positions(by_name: dict[str, Any], client_id: str) -> list[dict[str, Any]]:
    """Spot positions from the caller's books. Empty when the book cannot be read."""
    client_id = (client_id or "").strip()
    portfolio = by_name.get("get_portfolio")
    if not client_id or portfolio is None:
        return []

    rows: list[dict[str, Any]] = []
    seen: set[str] = set()

    def take(view: dict[str, Any] | None, book_name: str) -> None:
        if not view or "positions" not in view:
            return
        book_id = str(view.get("id") or "").strip()
        if book_id:
            if book_id in seen:
                return
            seen.add(book_id)
        name = book_name or str(view.get("name") or "").strip()
        listed = view.get("positions")
        if not isinstance(listed, list):
            return
        for pos in listed:
            if not isinstance(pos, dict):
                continue
            copied = dict(pos)
            if name:
                copied["_book"] = name
            rows.append(copied)

    primary = _parse_obj(_invoke(portfolio, {"client_id": client_id}))
    take(primary, "")

    lister = by_name.get("list_portfolios")
    listed = _parse_obj(_invoke(lister, {"client_id": client_id})) if lister is not None else None
    books = listed.get("portfolios") if isinstance(listed, dict) else None
    if not isinstance(books, list):
        return rows
    for book in books:
        if len(seen) >= _MAX_BOOKS:
            break
        if not isinstance(book, dict):
            continue
        book_id = str(book.get("id") or "").strip()
        if not book_id or book_id in seen:
            continue
        view = _parse_obj(_invoke(portfolio, {"client_id": client_id, "portfolio_id": book_id}))
        take(view, str(book.get("name") or "").strip())
    return rows


def _entries_for(rows: list[dict[str, Any]], symbol: str) -> list[SituationEntry]:
    want = symbol.strip().upper()
    found: list[SituationEntry] = []
    seen: set[tuple[str, str, str]] = set()
    for pos in rows:
        if str(pos.get("symbol") or "").strip().upper() != want:
            continue
        entry = _entry_from_position(pos)
        if entry is None:
            continue
        key = (entry.book.lower(), entry.exchange.lower(), entry.entry_at)
        if key in seen:
            continue
        seen.add(key)
        found.append(entry)
    return found


def _entry_from_position(pos: dict[str, Any]) -> SituationEntry | None:
    quantity = pos.get("quantity")
    if isinstance(quantity, int | float) and not isinstance(quantity, bool) and quantity <= 0:
        return None
    situation = pos.get("entrySituation")
    if not isinstance(situation, dict) or not situation:
        return None
    exchange = str(pos.get("exchange") or "").strip()
    if not exchange:
        return None
    return SituationEntry(
        exchange=exchange,
        situation=situation,
        entry_at=str(pos.get("entryAt") or "").strip(),
        book=str(pos.get("_book") or "").strip(),
    )
