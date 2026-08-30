"""API-tool reachability: registered HTTP tools vs specialist packs.

Confirms the review finding that funding/OI/CVD/etc. exist as HTTP tools
but are not on any production specialist pack.
"""

from __future__ import annotations

import inspect

from swyngora_ai.config import Settings
from swyngora_ai.graph.orchestrator import Orchestrator
from swyngora_ai.tools.market_http import build_market_tools
from swyngora_ai.tools.packs import BOOK_TOOLS, PACKS

# Market analytics that docs list as assistant-capable; must sit on a specialist pack.
BOOK_FLOW_ANALYTICS = frozenset(
    {
        "get_open_interest",
        "get_funding_rate",
        "get_funding_arb",
        "scan_funding_arb",
        "get_funding_arb_history",
        "create_funding_arb_watch",
        "list_funding_arb_watches",
        "get_funding_arb_watch",
        "update_funding_arb_watch",
        "pause_funding_arb_watch",
        "resume_funding_arb_watch",
        "delete_funding_arb_watch",
        "list_funding_arb_signals",
        "get_long_short_ratio",
        "get_futures_history",
        "estimate_liquidation_hunt",
        "get_liquidation_heatmap",
        "get_liquidation_overview",
        "get_squeeze_risk",
        "get_positioning",
        "get_venue_divergence",
        "get_taker_flow",
        "get_cvd",
        "get_basis",
        "get_price_correlation",
        "get_market_breadth",
        "get_price_volatility",
        "get_market_snapshot",
        "get_support_resistance",
        "get_whale_trades",
        "get_orderbook_history",
        "compare_orderbook_history",
        "get_orderbook_icebergs",
    }
)

MARGIN_EXTRAS_MISSING_PYTHON = frozenset(
    {
        "set_margin_mode",
        "adjust_margin",
        "repay_margin_debt",
    }
)


def _all_tool_names() -> set[str]:
    settings = Settings(_env_file=None, api_base_url="http://test")
    return {t.name for t in build_market_tools(settings)}


def test_review_evidence_analytics_are_on_book_pack():
    names = _all_tool_names()
    missing = BOOK_FLOW_ANALYTICS - names
    assert not missing, f"HTTP tools no longer registered: {sorted(missing)}"
    not_in_book = BOOK_FLOW_ANALYTICS - BOOK_TOOLS
    assert not not_in_book, f"analytics missing from book pack: {sorted(not_in_book)}"
    assert BOOK_FLOW_ANALYTICS <= PACKS["book"]


def test_review_evidence_margin_extras_have_no_python_binding():
    names = _all_tool_names()
    present = MARGIN_EXTRAS_MISSING_PYTHON & names
    assert not present, f"margin extras now bound in Python: {sorted(present)}"


def test_review_evidence_orchestrator_chat_is_not_desk_graph():
    src = inspect.getsource(Orchestrator.chat)
    assert "build_desk_graph" not in src
    assert "run_debate" not in src
    assert "_should_refresh_tape" in src
    sig = inspect.signature(Orchestrator.chat)
    assert sig.parameters["can_trade"].default is True
    assert sig.parameters["can_manage_keys"].default is True
