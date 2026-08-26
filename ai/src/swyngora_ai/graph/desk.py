"""LangGraph desk state machine: route → parallel gather → optional debate → synthesize → ground."""

from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import Any, TypedDict

from langgraph.graph import END, START, StateGraph

from swyngora_ai.agents.debate import run_debate
from swyngora_ai.agents.specialists import SpecialistRunner
from swyngora_ai.constants import (
    SPECIALIST_ACCOUNT,
    SPECIALIST_BOOK,
    SPECIALIST_PAPER,
    SPECIALIST_TAPE,
    SPECIALIST_WEB,
    SPECIALIST_X,
)
from swyngora_ai.graph.facts import MarketFacts, extract_market_facts
from swyngora_ai.graph.route import Route, classify_route
from swyngora_ai.graph.tape_fetch import prefetch_tape
from swyngora_ai.grounding import apply_grounding
from swyngora_ai.language import detect_reply_lang
from swyngora_ai.progress import emit
from swyngora_ai.tools.market_http import submit_with_bound_context


class DeskState(TypedDict, total=False):
    message: str
    memory_context: str
    route: dict[str, bool]
    tape: str
    book: str
    paper: str
    account: str
    web: str
    social: str
    bull: str
    bear: str
    facts: dict[str, Any]
    packet: str
    reply: str
    tool_blobs: list[str]
    fallback_symbol: str
    fallback_exchange: str


def _route_obj(state: DeskState) -> Route:
    raw = state.get("route") or {}
    return Route(
        tape=bool(raw.get("tape")),
        book=bool(raw.get("book")),
        paper=bool(raw.get("paper")),
        account=bool(raw.get("account")),
        web=bool(raw.get("web")),
        social=bool(raw.get("social")),
        debate=bool(raw.get("debate")),
        desk_note=bool(raw.get("desk_note")),
    )


def build_desk_graph(runner: SpecialistRunner) -> Any:
    """Compile the desk graph closed over a SpecialistRunner."""

    def route_node(state: DeskState) -> DeskState:
        emit("status", "Planning…")
        route = classify_route(state.get("message") or "")
        emit("thinking", f"Route: {route.as_dict()}")
        return {"route": route.as_dict()}

    def gather_node(state: DeskState) -> DeskState:
        route = _route_obj(state)
        msg = state.get("message") or ""
        jobs: list[tuple[str, str, str]] = []
        if route.tape:
            jobs.append(("tape", SPECIALIST_TAPE, msg))
        if route.book:
            jobs.append(("book", SPECIALIST_BOOK, msg))
        if route.paper:
            jobs.append(("paper", SPECIALIST_PAPER, msg))
        if route.account:
            jobs.append(("account", SPECIALIST_ACCOUNT, msg))
        if route.web:
            jobs.append(("web", SPECIALIST_WEB, msg))
        if route.social:
            jobs.append(("social", SPECIALIST_X, msg))

        out: DeskState = {
            "tape": "",
            "book": "",
            "paper": "",
            "account": "",
            "web": "",
            "social": "",
            "tool_blobs": [],
        }
        if not jobs:
            return out

        emit("status", f"Gathering {len(jobs)} specialist(s) in parallel…")

        def _one(item: tuple[str, str, str]) -> tuple[str, str]:
            key, name, task = item
            return key, runner.run(name, task)

        def _assign(key: str, text: str) -> None:
            if key == "tape":
                out["tape"] = text
            elif key == "book":
                out["book"] = text
            elif key == "paper":
                out["paper"] = text
            elif key == "account":
                out["account"] = text
            elif key == "web":
                out["web"] = text
            elif key == "social":
                out["social"] = text

        if len(jobs) == 1:
            key, text = _one(jobs[0])
            _assign(key, text)
            out["tool_blobs"] = [text]
            return out

        with ThreadPoolExecutor(max_workers=min(4, len(jobs))) as pool:
            futs = [submit_with_bound_context(pool, _one, job) for job in jobs]
            blobs: list[str] = []
            for fut in as_completed(futs):
                key, text = fut.result()
                _assign(key, text)
                if text:
                    blobs.append(text)
            out["tool_blobs"] = blobs
        return out

    def facts_node(state: DeskState) -> DeskState:
        blobs = list(state.get("tool_blobs") or [])
        for key in ("tape", "book", "paper", "account", "web", "social"):
            val = state.get(key) or ""
            if val and val not in blobs:
                blobs.append(val)
        facts = extract_market_facts(*blobs)
        route = _route_obj(state)
        if route.tape and not facts.last_price:
            emit("status", "Prefetching live tape…")
            extra = prefetch_tape(
                runner.settings,
                state.get("message") or "",
                fallback_symbol=state.get("fallback_symbol") or "",
                fallback_exchange=state.get("fallback_exchange") or "",
            )
            blobs = extra + blobs
            facts = extract_market_facts(*blobs)
        return {"facts": facts.as_dict(), "tool_blobs": blobs}

    def debate_node(state: DeskState) -> DeskState:
        route = _route_obj(state)
        if not route.debate:
            return {"bull": "", "bear": ""}
        packet = _packet(state)
        bull, bear = run_debate(runner.model, packet)
        return {"bull": bull, "bear": bear}

    def synthesize_node(state: DeskState) -> DeskState:
        emit("status", "Composing answer…")
        packet = _packet(state)
        reply = runner.analyze(packet)
        return {"packet": packet, "reply": reply}

    def ground_node(state: DeskState) -> DeskState:
        facts = MarketFacts(**(state.get("facts") or {}))
        reply = apply_grounding(
            state.get("reply") or "",
            facts,
            list(state.get("tool_blobs") or []),
            user_message=state.get("message") or "",
        )
        return {"reply": reply}

    g = StateGraph(DeskState)  # ty: ignore[invalid-argument-type]
    g.add_node("route", route_node)
    g.add_node("gather", gather_node)
    g.add_node("facts", facts_node)
    g.add_node("debate", debate_node)
    g.add_node("synthesize", synthesize_node)
    g.add_node("ground", ground_node)
    g.add_edge(START, "route")
    g.add_edge("route", "gather")
    g.add_edge("gather", "facts")
    g.add_edge("facts", "debate")
    g.add_edge("debate", "synthesize")
    g.add_edge("synthesize", "ground")
    g.add_edge("ground", END)
    return g.compile()


def _packet(state: DeskState) -> str:
    facts = MarketFacts(**(state.get("facts") or {}))
    parts = [
        f"User question: {state.get('message') or ''}",
        facts.as_prompt(),
    ]
    # Do not pass internal cache/TTL notes to the analyst — it will quote them.
    for label, key in (
        ("Tape", "tape"),
        ("Book", "book"),
        ("Paper", "paper"),
        ("Account", "account"),
        ("Web / filings", "web"),
        ("Social (weak)", "social"),
        ("Bull case", "bull"),
        ("Bear case", "bear"),
    ):
        val = (state.get(key) or "").strip()
        if val:
            parts.append(f"### {label}\n{val}")
    lang = detect_reply_lang(state.get("message") or "")
    desk_note = bool((state.get("route") or {}).get("desk_note"))
    if desk_note:
        if lang == "tr":
            parts.append(
                "Kullanıcı 1–2 günlük analiz istedi. Yanıtı TÜRKÇE yaz. "
                "MarketFacts'te last_price varsa onu yaz. "
                "Yalnızca listedeki URL'leri kaynak göster."
            )
        else:
            parts.append(
                "User asked for a 1–2 day desk note. Same language as the question. "
                "If MarketFacts has last_price, quote it. Cite only listed URLs."
            )
    elif lang == "tr":
        parts.append(
            "Kullanıcı sorusu Türkçe. Sadece soruyu yanıtla. "
            "1–2 günlük brifing, Bottom line, Market facts, bias/güven ekleme. "
            "Fiyat sormadıysa bant/RSI yazma."
        )
    else:
        parts.append(
            "Answer the question directly in the same language. "
            "Do NOT use Bottom line / Market facts / 1–2 day outline. "
            "Do not add tape, bias, or a watch list unless they asked."
        )
    return "\n\n".join(parts)
