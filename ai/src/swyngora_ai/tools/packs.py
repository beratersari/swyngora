"""Market HTTP tool packs so one ReAct loop never sees ~80 tools."""

from __future__ import annotations

from collections.abc import Iterable

from langchain_core.tools import BaseTool

TAPE_TOOLS: frozenset[str] = frozenset(
    {
        "health",
        "realtime_stream_info",
        "list_exchanges",
        "get_fx_rates",
        "get_ticker",
        "get_candles",
        "get_supply",
        "list_spot_markets",
        "get_indicators",
        "list_delist_schedule",
    }
)

BOOK_TOOLS: frozenset[str] = frozenset(
    {
        "get_spot_orderbook",
        "analyze_spot_orderbook",
        "analyze_market_orderbook",
        "get_liquidations",
        "get_market_liquidity",
        "estimate_market_impact",
        "analyze_swing",
        "scan_swing_setups",
        "detect_pump_events",
        "scan_pump_events",
    }
)

PAPER_TOOLS: frozenset[str] = frozenset(
    {
        "create_portfolio",
        "list_portfolios",
        "rename_portfolio",
        "delete_portfolio",
        "share_portfolio",
        "update_portfolio_share",
        "revoke_portfolio_share",
        "list_portfolio_shares",
        "list_shared_portfolios",
        "get_portfolio",
        "get_portfolio_performance",
        "deposit_portfolio_cash",
        "withdraw_portfolio_cash",
        "transfer_portfolio_cash",
        "list_portfolio_cash_movements",
        "get_portfolio_risk_limits",
        "set_portfolio_risk_limits",
        "clear_portfolio_risk_limits",
        "get_paper_trading_costs",
        "place_portfolio_order",
        "place_portfolio_pending_order",
        "place_portfolio_oco_order",
        "place_portfolio_bracket_order",
        "list_portfolio_orders",
        "get_portfolio_order",
        "amend_portfolio_order",
        "cancel_all_portfolio_orders",
        "cancel_portfolio_order",
        "list_portfolio_trades",
        "list_portfolio_lots",
        "create_recurring_buy",
        "update_recurring_buy",
        "list_recurring_buys",
        "get_recurring_buy",
        "pause_recurring_buy",
        "resume_recurring_buy",
        "delete_recurring_buy",
        "list_recurring_buy_runs",
        "create_portfolio_basket",
        "list_portfolio_baskets",
        "get_portfolio_basket",
        "update_portfolio_basket",
        "delete_portfolio_basket",
        "preview_portfolio_rebalance",
        "rebalance_portfolio_basket",
        "place_margin_order",
        "list_margin_positions",
        "close_margin_position",
        "set_margin_brackets",
        "list_margin_orders",
        "cancel_margin_order",
        "list_margin_trades",
        "create_price_diff_watch",
        "list_price_diff_watches",
        "get_price_diff_watch",
        "delete_price_diff_watch",
        "list_price_diff_opportunities",
        "get_price_diff_opportunity",
    }
)

ACCOUNT_TOOLS: frozenset[str] = frozenset(
    {
        "get_watchlist",
        "add_watchlist_item",
        "remove_watchlist_item",
        "share_watchlist",
        "update_watchlist_share",
        "revoke_watchlist_share",
        "list_watchlist_shares",
        "list_shared_watchlists",
        "list_watchlist_audit",
        "start_export",
        "get_export",
        "list_exports",
        "cancel_export",
        "preview_import",
        "confirm_import",
        "get_import",
        "list_imports",
        "cancel_import",
        "list_price_alerts",
        "create_price_alert",
        "create_orderbook_alert",
        "delete_price_alert",
        "get_alert_webhook",
        "set_alert_webhook",
        "delete_alert_webhook",
        "create_api_key",
        "list_api_keys",
        "revoke_api_key",
        "create_scanner_rule",
        "list_scanner_rules",
        "delete_scanner_rule",
        "list_scanner_results",
    }
)

PACKS: dict[str, frozenset[str]] = {
    "tape": TAPE_TOOLS,
    "book": BOOK_TOOLS,
    "paper": PAPER_TOOLS,
    "account": ACCOUNT_TOOLS,
}


def filter_tools(tools: Iterable[BaseTool], pack: str | None) -> list[BaseTool]:
    """Keep tools in ``pack``; unknown pack or None returns all."""
    if not pack:
        return list(tools)
    allowed = PACKS.get(pack)
    if allowed is None:
        return list(tools)
    return [t for t in tools if t.name in allowed]
