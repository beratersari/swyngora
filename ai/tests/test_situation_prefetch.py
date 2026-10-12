"""Coin-situation prefetch: same clauses as the coin card, blank stays blank."""

from __future__ import annotations

import json
from typing import Any

from langchain_core.messages import AIMessage, SystemMessage

from swyngora_ai.config import Settings
from swyngora_ai.graph.facts import extract_market_facts
from swyngora_ai.graph.situation_fetch import prefetch_situations, situation_prompt
from swyngora_ai.memory.finmem import FinMem, SessionMemory
from test_specialists_and_orchestrator import ScriptedModel


def _live() -> dict[str, Any]:
    return {
        "symbol": "BTCUSDT",
        "priceCvd": {"kind": "price_up_cvd_down"},
        "openInterestMissing": ["okx"],
        "liquidation": {"side": "long", "distancePct": 1.2},
        "printsMissing": ["binance", "bybit", "okx"],
    }


class _Tool:
    def __init__(self, name: str, payload: Any) -> None:
        self.name = name
        self.payload = payload
        self.calls: list[dict[str, Any]] = []

    def invoke(self, args: dict[str, Any]) -> str:
        self.calls.append(args)
        if isinstance(self.payload, Exception):
            raise self.payload
        if callable(self.payload):
            return self.payload(args)
        if isinstance(self.payload, str):
            return self.payload
        return json.dumps(self.payload)


def _install(monkeypatch, tools: list[_Tool]) -> None:
    from swyngora_ai.graph import situation_fetch as sf

    monkeypatch.setattr(sf, "build_market_tools", lambda settings, pack=None: tools)


def test_crypto_situation_leads_and_names_missing_venues(monkeypatch):
    situation = _Tool("get_coin_situation", _live())
    portfolio = _Tool("get_portfolio", {"id": "book-1", "name": "Main", "positions": []})
    listed = _Tool("list_portfolios", {"portfolios": [{"id": "book-1", "name": "Main"}]})
    _install(monkeypatch, [situation, portfolio, listed])

    reads = prefetch_situations(
        Settings(_env_file=None),
        "what is going on with bitcoin",
        client_id="caller-1",
    )
    assert situation.calls == [{"symbol": "BTCUSDT"}]
    assert portfolio.calls[0] == {"client_id": "caller-1"}
    assert len(reads) == 1
    assert reads[0].entries == []
    text = situation_prompt(reads)
    assert text.startswith("Coin situation")
    assert "do not invent a side, a flow, or a number" in text
    assert "BTCUSDT live:" in text
    assert "price vs futures CVD: price_up_cvd_down" in text
    assert "1h open interest: missing: okx" in text
    assert "estimated liquidation pocket: long, distancePct 1.2" in text
    assert "measured 24h prints: missing: binance, bybit, okx" in text
    missing = next(line for line in text.splitlines() if line.startswith("- measured 24h"))
    assert "0" not in missing
    assert "long" not in missing
    assert "flat" not in text
    assert "no position" not in text.lower()
    assert "Entry read" not in text
    facts = extract_market_facts(text)
    assert facts.last_price == ""
    assert facts.numbers == []


def test_greeting_does_not_call_situation(monkeypatch):
    def boom(settings, pack=None):
        raise AssertionError("greetings must not build situation tools")

    from swyngora_ai.graph import situation_fetch as sf

    monkeypatch.setattr(sf, "build_market_tools", boom)
    assert prefetch_situations(Settings(_env_file=None), "hello", client_id="c") == []


def test_equity_does_not_call_situation(monkeypatch):
    def boom(settings, pack=None):
        raise AssertionError("equity questions must not build situation tools")

    from swyngora_ai.graph import situation_fetch as sf

    monkeypatch.setattr(sf, "build_market_tools", boom)
    assert prefetch_situations(Settings(_env_file=None), "THY longest flight", client_id="c") == []


def test_equity_beside_crypto_fetches_only_the_coin(monkeypatch):
    situation = _Tool("get_coin_situation", _live())
    _install(monkeypatch, [situation])
    reads = prefetch_situations(Settings(_env_file=None), "THYAO and BTC", client_id="")
    assert [call["symbol"] for call in situation.calls] == ["BTCUSDT"]
    assert [read.symbol for read in reads] == ["BTCUSDT"]
    assert reads[0].entries == []


def test_open_buys_include_each_exchange_and_a_second_book(monkeypatch):
    def portfolio(args: dict[str, Any]) -> str:
        if args.get("portfolio_id") == "book-2":
            return json.dumps(
                {
                    "id": "book-2",
                    "name": "Swing",
                    "positions": [
                        {
                            "symbol": "BTCUSDT",
                            "exchange": "bybit",
                            "quantity": 2,
                            "entryAt": "2026-10-02T00:00:00Z",
                            "entrySituation": {"priceCvd": {"kind": "price_down_cvd_up"}},
                        }
                    ],
                }
            )
        return json.dumps(
            {
                "id": "book-1",
                "name": "Main",
                "positions": [
                    {
                        "symbol": "BTCUSDT",
                        "exchange": "binance",
                        "quantity": 1,
                        "entryAt": "2026-10-01T00:00:00Z",
                        "entrySituation": {"priceCvd": {"kind": "agrees"}},
                    },
                    {
                        "symbol": "ETHUSDT",
                        "exchange": "binance",
                        "quantity": 1,
                        "entrySituation": {"priceCvd": {"kind": "agrees"}},
                    },
                    {
                        "symbol": "BTCUSDT",
                        "exchange": "binance",
                        "quantity": 0,
                        "entrySituation": {"priceCvd": {"kind": "agrees"}},
                    },
                ],
                "marginPositions": [
                    {
                        "symbol": "BTCUSDT",
                        "exchange": "binance",
                        "entrySituation": {"priceCvd": {"kind": "agrees"}},
                    }
                ],
            }
        )

    situation = _Tool("get_coin_situation", _live())
    books = _Tool("get_portfolio", portfolio)
    listed = _Tool(
        "list_portfolios",
        {"portfolios": [{"id": "book-1", "name": "Main"}, {"id": "book-2", "name": "Swing"}]},
    )
    _install(monkeypatch, [situation, books, listed])
    reads = prefetch_situations(Settings(_env_file=None), "BTC", client_id="caller-1")
    assert len(reads) == 1
    entries = reads[0].entries
    assert [(e.exchange, e.book, e.entry_at) for e in entries] == [
        ("binance", "Main", "2026-10-01T00:00:00Z"),
        ("bybit", "Swing", "2026-10-02T00:00:00Z"),
    ]
    text = situation_prompt(reads)
    assert text.count("Entry read stored on the open paper buy") == 2
    assert "exchange=binance book=Main entryAt=2026-10-01T00:00:00Z" in text
    assert "price vs futures CVD: agrees" in text
    assert "exchange=bybit book=Swing entryAt=2026-10-02T00:00:00Z" in text
    assert "price vs futures CVD: price_down_cvd_up" in text
    payload = reads[0].as_dict()
    assert payload["entries"][0]["entrySituation"]["priceCvd"]["kind"] == "agrees"
    assert "margin" not in json.dumps(payload)


def test_portfolio_failure_keeps_the_live_read(monkeypatch):
    situation = _Tool("get_coin_situation", _live())
    portfolio = _Tool("get_portfolio", RuntimeError("down"))
    listed = _Tool("list_portfolios", RuntimeError("down"))
    _install(monkeypatch, [situation, portfolio, listed])
    reads = prefetch_situations(Settings(_env_file=None), "BTCUSDT", client_id="caller-1")
    assert len(reads) == 1
    assert reads[0].situation["priceCvd"]["kind"] == "price_up_cvd_down"
    assert reads[0].entries == []
    text = situation_prompt(reads)
    assert "BTCUSDT live:" in text
    assert "Entry read" not in text
    assert "no position" not in text.lower()


def test_situation_tool_failure_does_not_abort(monkeypatch):
    def situation(args: dict[str, Any]) -> str:
        if args["symbol"] == "BTCUSDT":
            raise RuntimeError("timeout")
        return json.dumps({"symbol": "ETHUSDT", "priceCvdMissing": ["bybit"]})

    tool = _Tool("get_coin_situation", situation)
    _install(monkeypatch, [tool])
    reads = prefetch_situations(Settings(_env_file=None), "BTC and ETH", client_id="")
    assert [read.symbol for read in reads] == ["ETHUSDT"]
    text = situation_prompt(reads)
    assert "missing: bybit" in text
    assert "price_up_cvd_down" not in text
    assert "0" not in text.split("missing: bybit", 1)[0].splitlines()[-1]


def test_orchestrator_puts_situation_above_tape_even_when_tape_is_skipped(monkeypatch):
    from swyngora_ai.graph.orchestrator import Orchestrator
    from swyngora_ai.graph.situation_fetch import SituationRead

    captured: dict[str, Any] = {}
    seen_clients: list[str] = []

    def _capture(graph: Any, messages: Any, config: Any, specialist: str | None = None):
        captured["messages"] = list(messages)
        return [AIMessage(content="The clauses above are the read.")]

    def _situations(settings, message: str, client_id: str = ""):
        seen_clients.append(client_id)
        return [SituationRead(symbol="BTCUSDT", situation=_live())]

    def _tape(settings, message: str, fallback_symbol: str = "", fallback_exchange: str = ""):
        return [json.dumps({"symbol": "BTCUSDT", "lastPrice": "100", "exchange": "binance"})]

    monkeypatch.setattr("swyngora_ai.graph.orchestrator.run_agent_with_progress", _capture)
    monkeypatch.setattr("swyngora_ai.graph.orchestrator.prefetch_situations", _situations)
    monkeypatch.setattr("swyngora_ai.graph.orchestrator.prefetch_tape", _tape)

    orch = Orchestrator(
        settings=Settings(_env_file=None),
        memory=SessionMemory(),
        model=ScriptedModel(responses=[AIMessage(content="unused")]),
    )
    bare = orch.chat("bitcoin", session_id="sit-1", client_id="caller-9")
    assert seen_clients == ["caller-9"]
    assert bare.reads[0]["symbol"] == "BTCUSDT"
    assert bare.reads[0]["situation"]["printsMissing"] == ["binance", "bybit", "okx"]
    systems = [m.content for m in captured["messages"] if isinstance(m, SystemMessage)]
    assert len(systems) == 1
    assert systems[0].startswith("Coin situation")
    assert "Fresh market tape" not in systems[0]
    assert "missing: binance, bybit, okx" in systems[0]

    orch.memory = FinMem(path=":memory:")
    both = orch.chat("What is BTC price?", session_id="sit-2", client_id="caller-9")
    systems = [m.content for m in captured["messages"] if isinstance(m, SystemMessage)]
    assert len(systems) == 2
    assert systems[0].startswith("Coin situation")
    assert systems[1].startswith("Fresh market tape")
    assert "100" in systems[1]
    assert both.reads[0]["symbol"] == "BTCUSDT"
