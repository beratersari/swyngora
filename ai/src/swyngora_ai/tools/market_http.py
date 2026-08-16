"""Market tools via Swyngora HTTP API (mirror of Go MCP tools)."""

from __future__ import annotations

import json
import re
from contextvars import ContextVar
from typing import Any

import httpx
from langchain_core.tools import BaseTool, StructuredTool
from pydantic import BaseModel, Field

from swyngora_ai.config import Settings, get_settings
from swyngora_ai.constants import EXCHANGE_VENUES, EXCHANGE_VENUES_OR_ALL
from swyngora_ai.tools.packs import filter_tools

bound_client_id: ContextVar[str] = ContextVar("bound_client_id", default="")
# Defaults True for CLI/local so tools work without an HTTP scope envelope.
bound_can_trade: ContextVar[bool] = ContextVar("bound_can_trade", default=True)
bound_can_manage_keys: ContextVar[bool] = ContextVar("bound_can_manage_keys", default=True)
_turn_tool_json: ContextVar[list[str] | None] = ContextVar("turn_tool_json", default=None)

_READ_ONLY_DENY = (
    "ERROR 403: this AI session is read-only; a trade-permission API key is required "
    "for portfolio, alert, and other mutating tools"
)
_KEY_ADMIN_DENY = (
    "ERROR 403: API key and account-admin tools are not available in this AI session "
    "(user keys cannot manage other keys)"
)

# One-time secrets and similar must never reach the LLM / tool trace.
_SECRET_KEYS = frozenset(
    {
        "secret",
        "apikey",
        "api_key",
        "token",
        "access_token",
        "refresh_token",
        "password",
        "private_key",
        "privatekey",
    }
)
_SWY_SECRET_RE = re.compile(r"\bswy_[0-9a-fA-F]{32,}\b")


def redact_secrets(value: Any) -> Any:
    """Strip secret-bearing fields and swy_… tokens from tool payloads."""
    if isinstance(value, dict):
        out: dict[str, Any] = {}
        for k, v in value.items():
            key = str(k)
            if key.lower().replace("-", "_") in _SECRET_KEYS:
                out[key] = "[redacted]"
            else:
                out[key] = redact_secrets(v)
        return out
    if isinstance(value, list):
        return [redact_secrets(v) for v in value]
    if isinstance(value, str):
        return _SWY_SECRET_RE.sub("[redacted]", value)
    return value


def format_tool_json(payload: Any) -> str:
    return json.dumps(redact_secrets(payload), indent=2)


def bind_client_id(client_id: str) -> Any:
    """Force tenant tools to the authenticated subject for this chat turn."""
    return bound_client_id.set((client_id or "").strip())


def reset_bound_client_id(token: Any) -> None:
    bound_client_id.reset(token)


def bind_tool_scope(*, can_trade: bool = True, can_manage_keys: bool = True) -> tuple[Any, Any]:
    """Bind trade + key-admin scope for one chat turn (returns tokens for reset)."""
    return bound_can_trade.set(bool(can_trade)), bound_can_manage_keys.set(bool(can_manage_keys))


def reset_tool_scope(tokens: tuple[Any, Any]) -> None:
    trade_tok, keys_tok = tokens
    bound_can_trade.reset(trade_tok)
    bound_can_manage_keys.reset(keys_tok)


def begin_tool_json_turn() -> Any:
    """Start collecting successful tool JSON for this chat turn (grounding)."""
    return _turn_tool_json.set([])


def reset_tool_json_turn(token: Any) -> None:
    _turn_tool_json.reset(token)


def collected_tool_json() -> list[str]:
    acc = _turn_tool_json.get()
    return list(acc) if acc else []


def record_tool_json(text: str) -> None:
    acc = _turn_tool_json.get()
    if acc is None or not text or text.startswith("ERROR"):
        return
    acc.append(text)


def _is_key_or_account_admin_path(path: str) -> bool:
    p = (path or "").split("?", 1)[0]
    return "/api/v1/account/api-keys" in p or p in (
        "/api/v1/account/close",
        "/api/v1/account/reopen",
    )


class _HTTP:
    def __init__(self, base_url: str, timeout: float = 25.0, auth_token: str = "") -> None:
        self.base = base_url.rstrip("/")
        self.timeout = timeout
        self.auth_token = (auth_token or "").strip()
        self._client = httpx.Client(timeout=timeout)

    def _headers(self) -> dict[str, str]:
        headers: dict[str, str] = {}
        if self.auth_token:
            headers["Authorization"] = f"Bearer {self.auth_token}"
        cid = bound_client_id.get()
        if cid:
            headers["X-Client-Id"] = cid
        return headers

    def _apply_client_id(self, params: dict[str, Any] | None, body: dict[str, Any] | None) -> None:
        cid = bound_client_id.get()
        if not cid:
            return
        if params is not None:
            params["clientId"] = cid
        if body is not None and "clientId" in body:
            body["clientId"] = cid

    def _scope_error(self, path: str, *, mutating: bool) -> str | None:
        if _is_key_or_account_admin_path(path) and not bound_can_manage_keys.get():
            return _KEY_ADMIN_DENY
        if mutating and not bound_can_trade.get():
            return _READ_ONLY_DENY
        return None

    def _gate(self, path: str, *, mutating: bool) -> str | None:
        """Reject path-id traversal and apply scope to the normalized URL path."""
        raw = path or ""
        head = raw.split("?", 1)[0]
        if ".." in head or "\\" in head:
            return "ERROR 400: invalid path id"
        try:
            norm = httpx.URL(raw if "://" in raw else f"http://local{raw}").path
        except (httpx.InvalidURL, ValueError, TypeError):
            return "ERROR 400: invalid path"
        if err := self._scope_error(raw, mutating=mutating):
            return err
        return self._scope_error(norm, mutating=mutating)

    def _body_text(self, r: httpx.Response) -> str:
        if r.status_code >= 400:
            return f"ERROR {r.status_code}: {r.text[:500]}"
        try:
            text = format_tool_json(r.json())
        except (ValueError, TypeError):
            text = _SWY_SECRET_RE.sub("[redacted]", r.text)
        record_tool_json(text)
        return text

    def _request(self, method: str, url: str, **kwargs: Any) -> str:
        try:
            r = self._client.request(method, url, **kwargs)
        except httpx.TimeoutException as e:
            return f"ERROR timeout: {e}"[:500]
        except httpx.ConnectError as e:
            return f"ERROR connect: {e}"[:500]
        except httpx.HTTPError as e:
            return f"ERROR {type(e).__name__}: {e}"[:500]
        return self._body_text(r)

    def get(self, path: str, params: dict[str, Any] | None = None) -> str:
        if err := self._gate(path, mutating=False):
            return err
        params = dict(params or {})
        self._apply_client_id(params, None)
        return self._request("GET", f"{self.base}{path}", params=params, headers=self._headers())

    def post(self, path: str, body: dict[str, Any]) -> str:
        if err := self._gate(path, mutating=True):
            return err
        body = dict(body or {})
        self._apply_client_id(None, body)
        return self._request("POST", f"{self.base}{path}", json=body, headers=self._headers())

    def put(self, path: str, body: dict[str, Any]) -> str:
        if err := self._gate(path, mutating=True):
            return err
        body = dict(body or {})
        self._apply_client_id(None, body)
        return self._request("PUT", f"{self.base}{path}", json=body, headers=self._headers())

    def patch(self, path: str, body: dict[str, Any]) -> str:
        if err := self._gate(path, mutating=True):
            return err
        body = dict(body or {})
        self._apply_client_id(None, body)
        return self._request("PATCH", f"{self.base}{path}", json=body, headers=self._headers())

    def delete(self, path: str, params: dict[str, Any]) -> str:
        if err := self._gate(path, mutating=True):
            return err
        params = dict(params or {})
        self._apply_client_id(params, None)
        return self._request("DELETE", f"{self.base}{path}", params=params, headers=self._headers())

    def post_bytes(
        self,
        path: str,
        data: bytes,
        content_type: str,
        params: dict[str, Any] | None = None,
    ) -> str:
        if err := self._gate(path, mutating=True):
            return err
        params = dict(params or {})
        self._apply_client_id(params, None)
        headers = {**self._headers(), "Content-Type": content_type}
        return self._request(
            "POST", f"{self.base}{path}", content=data, params=params, headers=headers
        )


class TickerInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or BTC-USD")
    exchange: str = Field(default="binance", description=EXCHANGE_VENUES)


class SwingScanInput(BaseModel):
    client_id: str = Field(description="Opaque client id")
    exchange: str = Field(default="", description="Optional venue filter")
    limit: int = Field(default=25, ge=1, le=25)


class OrderBookInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or BTC-USD")
    exchange: str = Field(default="binance", description=EXCHANGE_VENUES)
    group: str = Field(
        default="", description="Price bucket e.g. 0.1 or 0.01; empty = suggested default"
    )
    limit: int = Field(default=20, ge=5, le=100, description="Grouped rows per side")
    range_pct: float = Field(
        default=2.0,
        ge=0.25,
        le=10,
        description="±% of mid for pressure/wall analysis (default 2)",
    )


class OrderBookAnalysisInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or BTC-USD")
    exchange: str = Field(default="binance", description=EXCHANGE_VENUES)
    range_pct: float = Field(
        default=2.0,
        ge=0.25,
        le=10,
        description="±% of mid to include (default 2)",
    )


class MarketOrderBookInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or BTC-USD")
    range_pct: float = Field(
        default=2.0,
        ge=0.25,
        le=10,
        description="±% of shared mid (default 2)",
    )


class LiquidationsInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both venues)",
    )


class MarketLiquidityInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or BTC-USD")
    exchange: str = Field(
        default="all",
        description=f"{EXCHANGE_VENUES_OR_ALL} (default all = per-venue + market-wide)",
    )


class MarketImpactInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or BTC-USD")
    side: str = Field(default="buy", description="buy (default) or sell")
    quantity: float = Field(
        default=0,
        description="Base size to fill (e.g. 5 for 5 BTC). Do not set together with notional.",
    )
    notional: float = Field(
        default=0,
        description="Quote size to spend/receive (e.g. 1000000000). Do not set together with quantity.",
    )
    exchange: str = Field(
        default="all",
        description=f"{EXCHANGE_VENUES_OR_ALL} (default all = cheapest-first merge)",
    )


class CandlesInput(BaseModel):
    symbol: str
    exchange: str = "binance"
    interval: str = "1h"
    limit: int = Field(default=50, ge=1, le=1000)


class SupplyInput(BaseModel):
    asset: str = Field(description="Base asset e.g. BTC")


class SpotInput(BaseModel):
    exchange: str = "binance"
    q: str = ""
    quote: str = ""
    sort: str = "quoteVolume"
    order: str = "desc"
    tag: str = ""
    limit: int = Field(default=15, ge=1, le=100)
    offset: int = 0


class IndicatorsInput(BaseModel):
    symbol: str
    exchange: str = "binance"
    interval: str = "1h"
    limit: int = 30
    # Snake_case only — LangChain tool kwargs use schema field names.
    rsi_period: int = Field(default=14, description="RSI period (default 14)")
    ema_periods: str = Field(default="12,26", description="Comma-separated EMA periods")


class WatchGetInput(BaseModel):
    client_id: str = Field(description="Actor opaque client id (required)")
    owner_client_id: str = Field(
        default="",
        description="List owner when viewing a shared watchlist; empty for own list",
    )


class WatchAddInput(BaseModel):
    client_id: str
    symbol: str
    exchange: str = "binance"
    note: str = ""
    owner_client_id: str = Field(
        default="",
        description="List owner when editing a shared list (owner or editor)",
    )


class WatchRemoveInput(BaseModel):
    client_id: str
    symbol: str
    exchange: str = "binance"
    owner_client_id: str = Field(
        default="",
        description="List owner when editing a shared list (owner or editor)",
    )


class WatchShareInput(BaseModel):
    client_id: str = Field(description="Owner client id")
    grantee_client_id: str = Field(description="Client to share with")
    role: str = Field(description="viewer or editor")


class WatchRevokeInput(BaseModel):
    client_id: str = Field(description="Owner client id")
    grantee_client_id: str = Field(description="Client to revoke")


class WatchClientInput(BaseModel):
    client_id: str = Field(description="Opaque client id")


class WatchAuditInput(BaseModel):
    client_id: str = Field(description="Owner client id")
    limit: int = Field(default=50, ge=1, le=200)
    offset: int = Field(default=0, ge=0)


class ExportStartInput(BaseModel):
    client_id: str = Field(description="Owner client id")
    format: str = Field(default="json", description="json or csv")
    sections: str = Field(
        default="",
        description="Comma-separated: watchlist,shares,alerts,backtests,portfolios (empty=all)",
    )


class ExportGetInput(BaseModel):
    client_id: str
    export_id: str


class AlertListInput(BaseModel):
    client_id: str = Field(description="Opaque client id")


class AlertCreateInput(BaseModel):
    client_id: str
    symbol: str
    condition: str = Field(description="above | below")
    target_price: float = Field(gt=0, description="Threshold price")
    exchange: str = "binance"
    mode: str = Field(
        default="one_time",
        description="one_time (default) or repeating (re-fire after re-cross)",
    )


class OrderBookAlertInput(BaseModel):
    client_id: str
    symbol: str
    kind: str = Field(description="imbalance | wall")
    condition: str = Field(description="imbalance: above|below; wall: bid|ask|any")
    threshold: float = Field(
        default=0.0,
        description="Imbalance |value| 0.05–0.95, or wall min share 0–1 (0 = any wall)",
    )
    exchange: str = "binance"
    range_pct: float = Field(default=2.0, ge=0.25, le=10)
    mode: str = Field(default="repeating", description="repeating (default) | one_time")


class AlertDeleteInput(BaseModel):
    client_id: str
    id: str = Field(description="Alert id")


class AlertWebhookGetInput(BaseModel):
    client_id: str


class APIKeyCreateInput(BaseModel):
    client_id: str
    name: str
    permission: str = Field(default="read", description="read or trade")


class APIKeyRevokeInput(BaseModel):
    client_id: str
    id: str


class PortfolioCreateInput(BaseModel):
    client_id: str
    starting_balance: float = Field(gt=0, description="Starting cash")
    currency: str = "USDT"
    name: str = Field(default="", description="Book name; default Main. Unique per client.")


class PortfolioGetInput(BaseModel):
    client_id: str
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class PaperTradingCostsInput(BaseModel):
    exchange: str = Field(
        default="",
        description=f"Optional venue {EXCHANGE_VENUES}; omit to list all",
    )


class PortfolioRenameInput(BaseModel):
    client_id: str
    portfolio_id: str
    name: str


class PortfolioDeleteInput(BaseModel):
    client_id: str
    portfolio_id: str


class PortfolioShareInput(BaseModel):
    client_id: str
    grantee_client_id: str
    role: str = Field(description="viewer or trader")
    portfolio_id: str = ""


class PortfolioShareRevokeInput(BaseModel):
    client_id: str
    grantee_client_id: str
    portfolio_id: str = ""


class PortfolioPerformanceInput(BaseModel):
    client_id: str
    period: str = Field(default="1w", description="Lookback window: 1d, 1w, 1m, or 3m")
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class PortfolioCashMoveInput(BaseModel):
    client_id: str
    amount: float = Field(gt=0, description="Positive cash amount")
    note: str = ""


class PortfolioTransferInput(BaseModel):
    client_id: str
    to_portfolio_id: str
    amount: float = Field(gt=0, description="Positive cash to move")
    from_portfolio_id: str = ""
    note: str = ""


class PortfolioCashListInput(BaseModel):
    client_id: str
    limit: int = 50
    offset: int = 0


class RiskLimitsSetInput(BaseModel):
    client_id: str
    max_daily_loss_pct: float = Field(
        default=0, description="e.g. 5 = stop new risk at 5% daily MTM loss; 0 disables"
    )
    max_asset_weight_pct: float = Field(
        default=0, description="e.g. 30 = max one coin % of equity; 0 disables"
    )


class PortfolioOrderInput(BaseModel):
    client_id: str
    symbol: str
    side: str = Field(description="buy | sell")
    quantity: float = Field(gt=0)
    exchange: str = "binance"
    lot_method: str = Field(default="", description="fifo (default) or lifo for sells")
    idempotency_key: str = Field(
        default="",
        description="Optional retry key; same key + same request returns the original fill",
    )


class PortfolioPendingOrderInput(BaseModel):
    client_id: str
    symbol: str
    order_type: str = Field(description="limit_buy | limit_sell | stop_loss")
    quantity: float = Field(gt=0)
    trigger_price: float = Field(gt=0, description="Limit or stop price")
    exchange: str = "binance"
    time_in_force: str = Field(default="gtc", description="gtc | ioc | fok")
    expires_at: str = Field(default="", description="RFC3339 expiry for GTC only")
    lot_method: str = Field(default="", description="fifo or lifo for sell types")
    idempotency_key: str = Field(default="", description="Optional retry key")


class PortfolioLotsInput(BaseModel):
    client_id: str
    exchange: str = ""
    symbol: str = ""
    status: str = Field(default="open", description="open | closed | all")


class PortfolioListOrdersInput(BaseModel):
    client_id: str
    status: str = Field(default="open", description="open|filled|canceled|rejected|all")


class PortfolioCancelOrderInput(BaseModel):
    client_id: str
    order_id: str


class PortfolioOCOInput(BaseModel):
    client_id: str
    symbol: str
    quantity: float = Field(gt=0)
    take_profit_price: float = Field(gt=0)
    stop_loss_price: float = Field(gt=0)
    exchange: str = Field(default="binance", description=EXCHANGE_VENUES)
    expires_at: str = ""
    idempotency_key: str = ""


class PortfolioBracketInput(BaseModel):
    client_id: str
    symbol: str
    quantity: float = Field(gt=0)
    entry_price: float = Field(gt=0)
    take_profit_price: float = Field(gt=0)
    stop_loss_price: float = Field(gt=0)
    exchange: str = Field(default="binance", description=EXCHANGE_VENUES)
    expires_at: str = ""
    idempotency_key: str = ""


class PortfolioAmendInput(BaseModel):
    client_id: str
    order_id: str
    trigger_price: float = Field(default=0, description="New limit/stop; 0 = leave")
    remaining_quantity: float = Field(default=0, description="New remaining size; 0 = leave")


class PortfolioCancelAllInput(BaseModel):
    client_id: str
    symbol: str = ""
    exchange: str = Field(default="", description=EXCHANGE_VENUES)


class PortfolioOrderGetInput(BaseModel):
    client_id: str
    order_id: str


class DelistInput(BaseModel):
    exchange: str = Field(default="binance", description=EXCHANGE_VENUES)


class ImportPreviewInput(BaseModel):
    client_id: str
    content: str = Field(description="Full export file text (JSON preferred)")
    format: str = Field(default="json", description="json or csv")
    file_name: str = ""


class ImportConfirmInput(BaseModel):
    client_id: str
    import_id: str
    mode: str = Field(description="merge or replace")


class ImportGetInput(BaseModel):
    client_id: str
    import_id: str


class RecurringBuyCreateInput(BaseModel):
    client_id: str
    symbol: str
    amount: float = Field(gt=0, description="Cash notional per run")
    frequency: str = Field(description="daily | weekly | monthly | interval")
    exchange: str = "binance"
    name: str = Field(default="", description="Label e.g. Salary Day Buy")
    weekday: str = Field(default="", description="Weekly: monday..sunday")
    day_of_month: int = Field(default=0, description="Monthly salary day 1-31")
    interval_hours: int = Field(default=0, description="Interval frequency hours 1-168")
    start_at: str = Field(default="", description="RFC3339 first run; default now")


class RecurringBuyUpdateInput(BaseModel):
    client_id: str
    plan_id: str
    name: str = ""
    amount: float = 0
    frequency: str = ""
    weekday: str = ""
    day_of_month: int = 0
    interval_hours: int = 0
    start_at: str = ""


class BasketCreateInput(BaseModel):
    client_id: str
    name: str
    targets_json: str = Field(
        description='JSON array e.g. [{"asset":"BTC","weightPct":50},{"asset":"ETH","weightPct":30},{"asset":"USDT","weightPct":20}]'
    )


class BasketIdInput(BaseModel):
    client_id: str
    basket_id: str


class BasketUpdateInput(BaseModel):
    client_id: str
    basket_id: str
    name: str = ""
    targets_json: str = ""


class RecurringBuyPlanInput(BaseModel):
    client_id: str
    plan_id: str


class RecurringBuyRunsInput(BaseModel):
    client_id: str
    plan_id: str
    limit: int = 50
    offset: int = 0


class MarginOrderInput(BaseModel):
    client_id: str
    symbol: str
    side: str = Field(description="long | short")
    quantity: float = Field(gt=0)
    leverage: int = Field(ge=1, le=10)
    order_type: str = Field(default="market", description="market | limit")
    exchange: str = "binance"
    limit_price: float = Field(default=0, description="Required for limit")
    stop_loss: float = Field(default=0, description="Optional; 0 = omit")
    take_profit: float = Field(default=0, description="Optional; 0 = omit")
    idempotency_key: str = Field(default="", description="Optional retry key")


class MarginPositionIdInput(BaseModel):
    client_id: str
    position_id: str


class MarginCloseInput(BaseModel):
    client_id: str
    position_id: str
    quantity: float = Field(default=0, description="0 = full close")
    idempotency_key: str = Field(default="", description="Optional retry key")


class MarginBracketsInput(BaseModel):
    client_id: str
    position_id: str
    stop_loss: float = 0
    take_profit: float = 0
    clear_stop_loss: bool = False
    clear_take_profit: bool = False


class PriceDiffWatchCreateInput(BaseModel):
    client_id: str
    symbol: str
    min_net_diff_pct: float = Field(gt=0, description="Minimum net % after fees e.g. 0.5")
    fee_binance_pct: float = Field(default=0, ge=0, description="Binance fee %")
    fee_coinbase_pct: float = Field(default=0, ge=0, description="Coinbase fee %")
    fee_bybit_pct: float = Field(default=0, ge=0, description="Bybit fee %")


class PriceDiffWatchIdInput(BaseModel):
    client_id: str
    watch_id: str


class PriceDiffOppListInput(BaseModel):
    client_id: str
    status: str = Field(default="open", description="open | closed | all")
    limit: int = 50
    offset: int = 0


class PriceDiffOppIdInput(BaseModel):
    client_id: str
    opportunity_id: str


class ScannerRuleCreateInput(BaseModel):
    client_id: str
    rule_type: str = Field(description="rsi | ma_crossover | volume_increase")
    interval: str = "1h"
    rsi_period: int = 14
    rsi_condition: str = "below"
    rsi_threshold: float = 30
    ma_fast_period: int = 12
    ma_slow_period: int = 26
    ma_direction: str = "golden_cross"
    volume_lookback: int = 20
    volume_min_ratio: float = 2


class ScannerRuleDeleteInput(BaseModel):
    client_id: str
    rule_id: str


class AlertWebhookSetInput(BaseModel):
    client_id: str
    url: str = Field(description="Absolute http(s) webhook URL")
    delivery_mode: str = Field(
        default="immediate",
        description="immediate | hourly_digest",
    )
    time_zone: str = Field(default="UTC", description="IANA timezone for quiet hours")
    quiet_enabled: bool = Field(default=False, description="Defer delivery during quiet hours")
    quiet_start: str = Field(default="", description="Local HH:MM quiet start")
    quiet_end: str = Field(default="", description="Local HH:MM quiet end (may cross midnight)")


class PumpDetectInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or JUVUSDT")
    exchange: str = Field(default="binance", description=EXCHANGE_VENUES)
    interval: str = Field(default="1h", description="Candle interval: 1m,5m,15m,1h,4h,1d…")
    lookback_hours: float = Field(default=48, description="Hours of history to scan")
    min_return_pct: float = Field(default=5, description="Threshold percent e.g. 5 = +5%")
    window_bars: int = Field(default=1, ge=1, le=100, description="Bars for close_return window")
    mode: str = Field(
        default="close_return",
        description="close_return | candle_body | high_from_low",
    )
    direction: str = Field(default="up", description="up | down | both")
    min_volume_ratio: float = Field(default=0, description="Volume ≥ N× median (0=off)")
    max_events: int = Field(default=20, ge=1, le=100)


class PumpScanInput(BaseModel):
    exchange: str = "binance"
    quote: str = Field(default="USDT", description="Quote filter USDT/USD")
    interval: str = Field(default="15m", description="Candle interval for scan")
    lookback_hours: float = Field(default=24, description="Hours to scan")
    min_return_pct: float = Field(default=8, description="Threshold percent")
    window_bars: int = Field(default=1, ge=1, le=100)
    mode: str = "close_return"
    direction: str = "up"
    min_volume_ratio: float = 0
    symbol_limit: int = Field(default=15, ge=1, le=40, description="Top volume symbols to scan")
    max_total_events: int = Field(
        default=30,
        ge=1,
        le=200,
        description="Cap on total events across all symbols (not hit count)",
    )


def build_market_tools(settings: Settings | None = None, pack: str | None = None) -> list[BaseTool]:
    cfg = settings or get_settings()
    http = _HTTP(cfg.api_base_url, auth_token=cfg.api_auth_token)

    def health() -> str:
        return http.get("/health")

    def realtime_stream_info() -> str:
        return http.get("/api/v1/realtime")

    def list_exchanges() -> str:
        return http.get("/api/v1/market/exchanges")

    def list_delist_schedule(exchange: str = "binance") -> str:
        return http.get("/api/v1/market/delist-schedule", {"exchange": exchange})

    def get_fx_rates() -> str:
        return http.get("/api/v1/market/fx")

    def get_ticker(symbol: str, exchange: str = "binance") -> str:
        return http.get(
            "/api/v1/market/ticker/24h",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_spot_orderbook(
        symbol: str,
        exchange: str = "binance",
        group: str = "",
        limit: int = 20,
        range_pct: float = 2.0,
    ) -> str:
        params: dict[str, Any] = {
            "symbol": symbol,
            "exchange": exchange,
            "limit": limit,
            "rangePct": range_pct,
        }
        if group:
            params["group"] = group
        return http.get("/api/v1/market/orderbook", params)

    def analyze_market_orderbook(symbol: str, range_pct: float = 2.0) -> str:
        return http.get(
            "/api/v1/market/orderbook/combined",
            {"symbol": symbol, "rangePct": range_pct},
        )

    def get_liquidations(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/liquidations",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_market_liquidity(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/orderbook/liquidity",
            {"symbol": symbol, "exchange": exchange},
        )

    def estimate_market_impact(
        symbol: str,
        side: str = "buy",
        quantity: float = 0,
        notional: float = 0,
        exchange: str = "all",
    ) -> str:
        params: dict[str, Any] = {
            "symbol": symbol,
            "side": side,
            "exchange": exchange,
        }
        if quantity and quantity > 0:
            params["quantity"] = quantity
        if notional and notional > 0:
            params["notional"] = notional
        return http.get("/api/v1/market/orderbook/impact", params)

    def analyze_spot_orderbook(
        symbol: str,
        exchange: str = "binance",
        range_pct: float = 2.0,
    ) -> str:
        raw = http.get(
            "/api/v1/market/orderbook",
            {
                "symbol": symbol,
                "exchange": exchange,
                "limit": 5,
                "rangePct": range_pct,
            },
        )
        try:
            data = json.loads(raw)
        except json.JSONDecodeError:
            return raw
        keep = {
            k: data[k]
            for k in (
                "exchange",
                "symbol",
                "lastPrice",
                "bestBid",
                "bestAsk",
                "spread",
                "spreadPct",
                "live",
                "source",
                "updatedAt",
                "analysis",
            )
            if k in data
        }
        return json.dumps(keep)

    def get_candles(
        symbol: str,
        exchange: str = "binance",
        interval: str = "1h",
        limit: int = 50,
    ) -> str:
        return http.get(
            "/api/v1/market/candles",
            {
                "symbol": symbol,
                "exchange": exchange,
                "interval": interval,
                "limit": limit,
            },
        )

    def get_supply(asset: str) -> str:
        return http.get("/api/v1/market/supply", {"asset": asset})

    def list_spot_markets(
        exchange: str = "binance",
        q: str = "",
        quote: str = "",
        sort: str = "quoteVolume",
        order: str = "desc",
        tag: str = "",
        limit: int = 15,
        offset: int = 0,
    ) -> str:
        params: dict[str, Any] = {
            "exchange": exchange,
            "sort": sort,
            "order": order,
            "limit": limit,
            "offset": offset,
        }
        if q:
            params["q"] = q
        if quote:
            params["quote"] = quote
        if tag:
            params["tag"] = tag
        return http.get("/api/v1/market/spot", params)

    def get_indicators(
        symbol: str,
        exchange: str = "binance",
        interval: str = "1h",
        limit: int = 30,
        rsi_period: int = 14,
        ema_periods: str = "12,26",
        # Accept camelCase from older prompts if the model still sends them.
        rsiPeriod: int | None = None,
        emaPeriods: str | None = None,
    ) -> str:
        rp = rsiPeriod if rsiPeriod is not None else rsi_period
        ep = emaPeriods if emaPeriods is not None else ema_periods
        return http.get(
            "/api/v1/market/indicators",
            {
                "symbol": symbol,
                "exchange": exchange,
                "interval": interval,
                "limit": limit,
                "rsiPeriod": rp,
                "emaPeriods": ep,
            },
        )

    def analyze_swing(symbol: str, exchange: str = "binance") -> str:
        return http.get("/api/v1/market/swing", {"symbol": symbol, "exchange": exchange})

    def scan_swing_setups(client_id: str, exchange: str = "", limit: int = 25) -> str:
        params: dict[str, Any] = {"clientId": client_id, "limit": limit}
        if exchange:
            params["exchange"] = exchange
        return http.get("/api/v1/swing/setups", params)

    def get_watchlist(client_id: str, owner_client_id: str = "") -> str:
        params: dict[str, Any] = {"clientId": client_id}
        if owner_client_id:
            params["ownerClientId"] = owner_client_id
        return http.get("/api/v1/watchlist", params)

    def add_watchlist_item(
        client_id: str,
        symbol: str,
        exchange: str = "binance",
        note: str = "",
        owner_client_id: str = "",
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "symbol": symbol,
            "exchange": exchange,
            "note": note,
        }
        if owner_client_id:
            body["ownerClientId"] = owner_client_id
        return http.post("/api/v1/watchlist/items", body)

    def remove_watchlist_item(
        client_id: str,
        symbol: str,
        exchange: str = "binance",
        owner_client_id: str = "",
    ) -> str:
        params: dict[str, Any] = {
            "clientId": client_id,
            "symbol": symbol,
            "exchange": exchange,
        }
        if owner_client_id:
            params["ownerClientId"] = owner_client_id
        return http.delete("/api/v1/watchlist/items", params)

    def share_watchlist(client_id: str, grantee_client_id: str, role: str) -> str:
        return http.post(
            "/api/v1/watchlist/shares",
            {
                "clientId": client_id,
                "granteeClientId": grantee_client_id,
                "role": role,
            },
        )

    def update_watchlist_share(client_id: str, grantee_client_id: str, role: str) -> str:
        return http.patch(
            "/api/v1/watchlist/shares",
            {
                "clientId": client_id,
                "granteeClientId": grantee_client_id,
                "role": role,
            },
        )

    def revoke_watchlist_share(client_id: str, grantee_client_id: str) -> str:
        return http.delete(
            "/api/v1/watchlist/shares",
            {"clientId": client_id, "granteeClientId": grantee_client_id},
        )

    def list_watchlist_shares(client_id: str) -> str:
        return http.get("/api/v1/watchlist/shares", {"clientId": client_id})

    def list_shared_watchlists(client_id: str) -> str:
        return http.get("/api/v1/watchlist/shared", {"clientId": client_id})

    def list_watchlist_audit(client_id: str, limit: int = 50, offset: int = 0) -> str:
        return http.get(
            "/api/v1/watchlist/audit",
            {"clientId": client_id, "limit": limit, "offset": offset},
        )

    def start_export(
        client_id: str,
        format: str = "json",
        sections: str = "",
    ) -> str:
        body: dict[str, Any] = {"clientId": client_id, "format": format or "json"}
        if sections:
            body["sections"] = [s.strip() for s in sections.split(",") if s.strip()]
        return http.post("/api/v1/export", body)

    def get_export(client_id: str, export_id: str) -> str:
        return http.get(f"/api/v1/export/{export_id}", {"clientId": client_id})

    def list_exports(client_id: str, limit: int = 20, offset: int = 0) -> str:
        return http.get(
            "/api/v1/export",
            {"clientId": client_id, "limit": limit, "offset": offset},
        )

    def cancel_export(client_id: str, export_id: str) -> str:
        return http.post(f"/api/v1/export/{export_id}/cancel?clientId={client_id}", {})

    def list_price_alerts(client_id: str) -> str:
        return http.get("/api/v1/alerts", {"clientId": client_id})

    def create_price_alert(
        client_id: str,
        symbol: str,
        condition: str,
        target_price: float,
        exchange: str = "binance",
        mode: str = "one_time",
    ) -> str:
        return http.post(
            "/api/v1/alerts",
            {
                "clientId": client_id,
                "symbol": symbol,
                "condition": condition,
                "targetPrice": target_price,
                "exchange": exchange,
                "mode": mode,
            },
        )

    def create_orderbook_alert(
        client_id: str,
        symbol: str,
        kind: str,
        condition: str,
        threshold: float = 0.0,
        exchange: str = "binance",
        range_pct: float = 2.0,
        mode: str = "repeating",
    ) -> str:
        return http.post(
            "/api/v1/alerts",
            {
                "clientId": client_id,
                "symbol": symbol,
                "kind": kind,
                "condition": condition,
                "targetPrice": threshold,
                "rangePct": range_pct,
                "exchange": exchange,
                "mode": mode,
            },
        )

    def delete_price_alert(client_id: str, id: str) -> str:
        return http.delete(f"/api/v1/alerts/{id}", {"clientId": client_id})

    def get_alert_webhook(client_id: str) -> str:
        return http.get("/api/v1/alerts/webhook", {"clientId": client_id})

    def set_alert_webhook(
        client_id: str,
        url: str,
        delivery_mode: str = "immediate",
        time_zone: str = "UTC",
        quiet_enabled: bool = False,
        quiet_start: str = "",
        quiet_end: str = "",
    ) -> str:
        return http.put(
            "/api/v1/alerts/webhook",
            {
                "clientId": client_id,
                "url": url,
                "deliveryMode": delivery_mode,
                "timeZone": time_zone,
                "quietHours": {
                    "enabled": quiet_enabled,
                    "start": quiet_start,
                    "end": quiet_end,
                },
            },
        )

    def delete_alert_webhook(client_id: str) -> str:
        return http.delete("/api/v1/alerts/webhook", {"clientId": client_id})

    def create_api_key(client_id: str, name: str, permission: str = "read") -> str:
        return http.post(
            "/api/v1/account/api-keys",
            {"clientId": client_id, "name": name, "permission": permission},
        )

    def list_api_keys(client_id: str) -> str:
        return http.get("/api/v1/account/api-keys", {"clientId": client_id})

    def revoke_api_key(client_id: str, id: str) -> str:
        return http.delete(
            f"/api/v1/account/api-keys/{id}",
            {"clientId": client_id},
        )

    def create_portfolio(
        client_id: str,
        starting_balance: float,
        currency: str = "USDT",
        name: str = "",
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "startingBalance": starting_balance,
            "currency": currency,
        }
        if name:
            body["name"] = name
        return http.post("/api/v1/portfolio", body)

    def list_portfolios(client_id: str) -> str:
        return http.get("/api/v1/portfolios", {"clientId": client_id})

    def rename_portfolio(client_id: str, portfolio_id: str, name: str) -> str:
        return http.patch(
            f"/api/v1/portfolios/{portfolio_id}?clientId={client_id}",
            {"name": name},
        )

    def delete_portfolio(client_id: str, portfolio_id: str) -> str:
        return http.delete(
            f"/api/v1/portfolios/{portfolio_id}",
            {"clientId": client_id},
        )

    def share_portfolio(
        client_id: str, grantee_client_id: str, role: str, portfolio_id: str = ""
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "granteeClientId": grantee_client_id,
            "role": role,
        }
        if portfolio_id:
            body["portfolioId"] = portfolio_id
        return http.post("/api/v1/portfolio/shares", body)

    def update_portfolio_share(
        client_id: str, grantee_client_id: str, role: str, portfolio_id: str = ""
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "granteeClientId": grantee_client_id,
            "role": role,
        }
        if portfolio_id:
            body["portfolioId"] = portfolio_id
        return http.patch("/api/v1/portfolio/shares", body)

    def revoke_portfolio_share(
        client_id: str, grantee_client_id: str, portfolio_id: str = ""
    ) -> str:
        params: dict[str, Any] = {
            "clientId": client_id,
            "granteeClientId": grantee_client_id,
        }
        if portfolio_id:
            params["portfolioId"] = portfolio_id
        return http.delete("/api/v1/portfolio/shares", params)

    def list_portfolio_shares(client_id: str, portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/shares", q)

    def list_shared_portfolios(client_id: str) -> str:
        return http.get("/api/v1/portfolios/shared", {"clientId": client_id})

    def get_portfolio_risk_limits(client_id: str) -> str:
        return http.get("/api/v1/portfolio/risk-limits", {"clientId": client_id})

    def get_paper_trading_costs(exchange: str = "") -> str:
        q: dict[str, Any] = {}
        if exchange:
            q["exchange"] = exchange
        return http.get("/api/v1/portfolio/trading-costs", q)

    def set_portfolio_risk_limits(
        client_id: str,
        max_daily_loss_pct: float = 0,
        max_asset_weight_pct: float = 0,
    ) -> str:
        body: dict[str, Any] = {}
        if max_daily_loss_pct:
            body["maxDailyLossPct"] = max_daily_loss_pct
        if max_asset_weight_pct:
            body["maxAssetWeightPct"] = max_asset_weight_pct
        return http.put(
            f"/api/v1/portfolio/risk-limits?clientId={client_id}",
            body,
        )

    def clear_portfolio_risk_limits(client_id: str) -> str:
        return http.delete("/api/v1/portfolio/risk-limits", {"clientId": client_id})

    def get_portfolio(client_id: str, portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio", q)

    def get_portfolio_performance(
        client_id: str, period: str = "1w", portfolio_id: str = ""
    ) -> str:
        q: dict[str, Any] = {"clientId": client_id, "period": period}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/performance", q)

    def deposit_portfolio_cash(client_id: str, amount: float, note: str = "") -> str:
        body: dict[str, Any] = {"clientId": client_id, "amount": amount}
        if note:
            body["note"] = note
        return http.post("/api/v1/portfolio/deposits", body)

    def transfer_portfolio_cash(
        client_id: str,
        to_portfolio_id: str,
        amount: float,
        from_portfolio_id: str = "",
        note: str = "",
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "toPortfolioId": to_portfolio_id,
            "amount": amount,
        }
        if from_portfolio_id:
            body["fromPortfolioId"] = from_portfolio_id
        if note:
            body["note"] = note
        return http.post("/api/v1/portfolio/transfers", body)

    def withdraw_portfolio_cash(client_id: str, amount: float, note: str = "") -> str:
        body: dict[str, Any] = {"clientId": client_id, "amount": amount}
        if note:
            body["note"] = note
        return http.post("/api/v1/portfolio/withdrawals", body)

    def list_portfolio_cash_movements(client_id: str, limit: int = 50, offset: int = 0) -> str:
        return http.get(
            "/api/v1/portfolio/cash-movements",
            {"clientId": client_id, "limit": limit, "offset": offset},
        )

    def place_portfolio_order(
        client_id: str,
        symbol: str,
        side: str,
        quantity: float,
        exchange: str = "binance",
        lot_method: str = "",
        idempotency_key: str = "",
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "symbol": symbol,
            "side": side,
            "quantity": quantity,
            "exchange": exchange,
        }
        if lot_method:
            body["lotMethod"] = lot_method
        if idempotency_key:
            body["idempotencyKey"] = idempotency_key
        return http.post("/api/v1/portfolio/orders", body)

    def place_portfolio_pending_order(
        client_id: str,
        symbol: str,
        order_type: str,
        quantity: float,
        trigger_price: float,
        exchange: str = "binance",
        time_in_force: str = "gtc",
        expires_at: str = "",
        lot_method: str = "",
        idempotency_key: str = "",
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "symbol": symbol,
            "type": order_type,
            "quantity": quantity,
            "triggerPrice": trigger_price,
            "exchange": exchange,
            "timeInForce": time_in_force,
        }
        if expires_at:
            body["expiresAt"] = expires_at
        if lot_method:
            body["lotMethod"] = lot_method
        if idempotency_key:
            body["idempotencyKey"] = idempotency_key
        return http.post("/api/v1/portfolio/orders", body)

    def list_portfolio_lots(
        client_id: str, exchange: str = "", symbol: str = "", status: str = "open"
    ) -> str:
        params: dict[str, Any] = {"clientId": client_id, "status": status}
        if exchange:
            params["exchange"] = exchange
        if symbol:
            params["symbol"] = symbol
        return http.get("/api/v1/portfolio/lots", params)

    def list_portfolio_orders(client_id: str, status: str = "open") -> str:
        return http.get(
            "/api/v1/portfolio/orders",
            {"clientId": client_id, "status": status},
        )

    def cancel_portfolio_order(client_id: str, order_id: str) -> str:
        return http.delete(
            f"/api/v1/portfolio/orders/{order_id}",
            {"clientId": client_id},
        )

    def get_portfolio_order(client_id: str, order_id: str) -> str:
        return http.get(f"/api/v1/portfolio/orders/{order_id}", {"clientId": client_id})

    def place_portfolio_oco_order(
        client_id: str,
        symbol: str,
        quantity: float,
        take_profit_price: float,
        stop_loss_price: float,
        exchange: str = "binance",
        expires_at: str = "",
        idempotency_key: str = "",
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "symbol": symbol,
            "type": "oco",
            "quantity": quantity,
            "takeProfitPrice": take_profit_price,
            "stopLossPrice": stop_loss_price,
            "exchange": exchange,
        }
        if expires_at:
            body["expiresAt"] = expires_at
        if idempotency_key:
            body["idempotencyKey"] = idempotency_key
        return http.post("/api/v1/portfolio/orders", body)

    def place_portfolio_bracket_order(
        client_id: str,
        symbol: str,
        quantity: float,
        entry_price: float,
        take_profit_price: float,
        stop_loss_price: float,
        exchange: str = "binance",
        expires_at: str = "",
        idempotency_key: str = "",
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "symbol": symbol,
            "type": "bracket",
            "quantity": quantity,
            "triggerPrice": entry_price,
            "takeProfitPrice": take_profit_price,
            "stopLossPrice": stop_loss_price,
            "exchange": exchange,
        }
        if expires_at:
            body["expiresAt"] = expires_at
        if idempotency_key:
            body["idempotencyKey"] = idempotency_key
        return http.post("/api/v1/portfolio/orders", body)

    def amend_portfolio_order(
        client_id: str,
        order_id: str,
        trigger_price: float = 0,
        remaining_quantity: float = 0,
    ) -> str:
        body: dict[str, Any] = {}
        if trigger_price > 0:
            body["triggerPrice"] = trigger_price
        if remaining_quantity > 0:
            body["remainingQuantity"] = remaining_quantity
        return http.patch(f"/api/v1/portfolio/orders/{order_id}?clientId={client_id}", body)

    def cancel_all_portfolio_orders(client_id: str, symbol: str = "", exchange: str = "") -> str:
        body: dict[str, Any] = {"clientId": client_id}
        if symbol:
            body["symbol"] = symbol
        if exchange:
            body["exchange"] = exchange
        return http.post("/api/v1/portfolio/orders/cancel-all", body)

    def preview_import(
        client_id: str, content: str, format: str = "json", file_name: str = ""
    ) -> str:
        params: dict[str, Any] = {"clientId": client_id, "format": format or "json"}
        if file_name:
            params["fileName"] = file_name
        ct = "text/csv" if (format or "").lower() == "csv" else "application/json"
        return http.post_bytes("/api/v1/import/preview", content.encode("utf-8"), ct, params)

    def confirm_import(client_id: str, import_id: str, mode: str) -> str:
        return http.post(
            f"/api/v1/import/{import_id}/confirm",
            {"clientId": client_id, "mode": mode},
        )

    def get_import(client_id: str, import_id: str) -> str:
        return http.get(f"/api/v1/import/{import_id}", {"clientId": client_id})

    def list_imports(client_id: str, limit: int = 20, offset: int = 0) -> str:
        return http.get(
            "/api/v1/import",
            {"clientId": client_id, "limit": limit, "offset": offset},
        )

    def cancel_import(client_id: str, import_id: str) -> str:
        return http.post(f"/api/v1/import/{import_id}/cancel?clientId={client_id}", {})

    def list_portfolio_trades(client_id: str) -> str:
        return http.get("/api/v1/portfolio/trades", {"clientId": client_id})

    def create_recurring_buy(
        client_id: str,
        symbol: str,
        amount: float,
        frequency: str,
        exchange: str = "binance",
        name: str = "",
        weekday: str = "",
        day_of_month: int = 0,
        interval_hours: int = 0,
        start_at: str = "",
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "symbol": symbol,
            "amount": amount,
            "frequency": frequency,
            "exchange": exchange,
        }
        if name:
            body["name"] = name
        if weekday:
            body["weekday"] = weekday
        if day_of_month:
            body["dayOfMonth"] = day_of_month
        if interval_hours:
            body["intervalHours"] = interval_hours
        if start_at:
            body["startAt"] = start_at
        return http.post("/api/v1/portfolio/recurring-buys", body)

    def update_recurring_buy(
        client_id: str,
        plan_id: str,
        name: str = "",
        amount: float = 0,
        frequency: str = "",
        weekday: str = "",
        day_of_month: int = 0,
        interval_hours: int = 0,
        start_at: str = "",
    ) -> str:
        body: dict[str, Any] = {}
        if name:
            body["name"] = name
        if amount:
            body["amount"] = amount
        if frequency:
            body["frequency"] = frequency
        if weekday:
            body["weekday"] = weekday
        if day_of_month:
            body["dayOfMonth"] = day_of_month
        if interval_hours:
            body["intervalHours"] = interval_hours
        if start_at:
            body["startAt"] = start_at
        return http.patch(
            f"/api/v1/portfolio/recurring-buys/{plan_id}?clientId={client_id}",
            body,
        )

    def list_recurring_buys(client_id: str) -> str:
        return http.get("/api/v1/portfolio/recurring-buys", {"clientId": client_id})

    def get_recurring_buy(client_id: str, plan_id: str) -> str:
        return http.get(
            f"/api/v1/portfolio/recurring-buys/{plan_id}",
            {"clientId": client_id},
        )

    def pause_recurring_buy(client_id: str, plan_id: str) -> str:
        # clientId is query/header only on these routes
        return http.post(
            f"/api/v1/portfolio/recurring-buys/{plan_id}/pause?clientId={client_id}",
            {},
        )

    def resume_recurring_buy(client_id: str, plan_id: str) -> str:
        return http.post(
            f"/api/v1/portfolio/recurring-buys/{plan_id}/resume?clientId={client_id}",
            {},
        )

    def delete_recurring_buy(client_id: str, plan_id: str) -> str:
        return http.delete(
            f"/api/v1/portfolio/recurring-buys/{plan_id}",
            {"clientId": client_id},
        )

    def list_recurring_buy_runs(
        client_id: str, plan_id: str, limit: int = 50, offset: int = 0
    ) -> str:
        return http.get(
            f"/api/v1/portfolio/recurring-buys/{plan_id}/runs",
            {"clientId": client_id, "limit": str(limit), "offset": str(offset)},
        )

    def create_portfolio_basket(client_id: str, name: str, targets_json: str) -> str:
        targets = json.loads(targets_json)
        return http.post(
            "/api/v1/portfolio/baskets",
            {"clientId": client_id, "name": name, "targets": targets},
        )

    def list_portfolio_baskets(client_id: str) -> str:
        return http.get("/api/v1/portfolio/baskets", {"clientId": client_id})

    def get_portfolio_basket(client_id: str, basket_id: str) -> str:
        return http.get(
            f"/api/v1/portfolio/baskets/{basket_id}",
            {"clientId": client_id},
        )

    def update_portfolio_basket(
        client_id: str, basket_id: str, name: str = "", targets_json: str = ""
    ) -> str:
        body: dict[str, Any] = {}
        if name:
            body["name"] = name
        if targets_json:
            body["targets"] = json.loads(targets_json)
        return http.patch(
            f"/api/v1/portfolio/baskets/{basket_id}?clientId={client_id}",
            body,
        )

    def delete_portfolio_basket(client_id: str, basket_id: str) -> str:
        return http.delete(
            f"/api/v1/portfolio/baskets/{basket_id}",
            {"clientId": client_id},
        )

    def preview_portfolio_rebalance(client_id: str, basket_id: str) -> str:
        return http.get(
            f"/api/v1/portfolio/baskets/{basket_id}/preview",
            {"clientId": client_id},
        )

    def rebalance_portfolio_basket(client_id: str, basket_id: str) -> str:
        return http.post(
            f"/api/v1/portfolio/baskets/{basket_id}/rebalance?clientId={client_id}",
            {},
        )

    def place_margin_order(
        client_id: str,
        symbol: str,
        side: str,
        quantity: float,
        leverage: int,
        order_type: str = "market",
        exchange: str = "binance",
        limit_price: float = 0,
        stop_loss: float = 0,
        take_profit: float = 0,
        idempotency_key: str = "",
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "symbol": symbol,
            "side": side,
            "quantity": quantity,
            "leverage": leverage,
            "type": order_type,
            "exchange": exchange,
        }
        if limit_price > 0:
            body["limitPrice"] = limit_price
        if stop_loss > 0:
            body["stopLoss"] = stop_loss
        if take_profit > 0:
            body["takeProfit"] = take_profit
        if idempotency_key:
            body["idempotencyKey"] = idempotency_key
        return http.post("/api/v1/portfolio/margin/orders", body)

    def list_margin_positions(client_id: str) -> str:
        return http.get("/api/v1/portfolio/margin/positions", {"clientId": client_id})

    def close_margin_position(
        client_id: str, position_id: str, quantity: float = 0, idempotency_key: str = ""
    ) -> str:
        body: dict[str, Any] = {}
        if quantity > 0:
            body["quantity"] = quantity
        if idempotency_key:
            body["idempotencyKey"] = idempotency_key
        return http.post(
            f"/api/v1/portfolio/margin/positions/{position_id}/close?clientId={client_id}",
            body,
        )

    def set_margin_brackets(
        client_id: str,
        position_id: str,
        stop_loss: float = 0,
        take_profit: float = 0,
        clear_stop_loss: bool = False,
        clear_take_profit: bool = False,
    ) -> str:
        body: dict[str, Any] = {
            "clearStopLoss": clear_stop_loss,
            "clearTakeProfit": clear_take_profit,
        }
        if stop_loss > 0:
            body["stopLoss"] = stop_loss
        if take_profit > 0:
            body["takeProfit"] = take_profit
        return http.put(
            f"/api/v1/portfolio/margin/positions/{position_id}/brackets?clientId={client_id}",
            body,
        )

    def list_margin_orders(client_id: str, status: str = "open") -> str:
        return http.get(
            "/api/v1/portfolio/margin/orders",
            {"clientId": client_id, "status": status},
        )

    def cancel_margin_order(client_id: str, order_id: str) -> str:
        return http.delete(
            f"/api/v1/portfolio/margin/orders/{order_id}",
            {"clientId": client_id},
        )

    def list_margin_trades(client_id: str) -> str:
        return http.get("/api/v1/portfolio/margin/trades", {"clientId": client_id})

    def create_price_diff_watch(
        client_id: str,
        symbol: str,
        min_net_diff_pct: float,
        fee_binance_pct: float = 0,
        fee_coinbase_pct: float = 0,
        fee_bybit_pct: float = 0,
    ) -> str:
        return http.post(
            "/api/v1/price-diff/watches",
            {
                "clientId": client_id,
                "symbol": symbol,
                "minNetDiffPct": min_net_diff_pct,
                "feeBinancePct": fee_binance_pct,
                "feeCoinbasePct": fee_coinbase_pct,
                "feeBybitPct": fee_bybit_pct,
            },
        )

    def list_price_diff_watches(client_id: str) -> str:
        return http.get("/api/v1/price-diff/watches", {"clientId": client_id})

    def get_price_diff_watch(client_id: str, watch_id: str) -> str:
        return http.get(
            f"/api/v1/price-diff/watches/{watch_id}",
            {"clientId": client_id},
        )

    def delete_price_diff_watch(client_id: str, watch_id: str) -> str:
        return http.delete(
            f"/api/v1/price-diff/watches/{watch_id}",
            {"clientId": client_id},
        )

    def list_price_diff_opportunities(
        client_id: str, status: str = "open", limit: int = 50, offset: int = 0
    ) -> str:
        return http.get(
            "/api/v1/price-diff/opportunities",
            {
                "clientId": client_id,
                "status": status,
                "limit": str(limit),
                "offset": str(offset),
            },
        )

    def get_price_diff_opportunity(client_id: str, opportunity_id: str) -> str:
        return http.get(
            f"/api/v1/price-diff/opportunities/{opportunity_id}",
            {"clientId": client_id},
        )

    def create_scanner_rule(
        client_id: str,
        rule_type: str,
        interval: str = "1h",
        rsi_period: int = 14,
        rsi_condition: str = "below",
        rsi_threshold: float = 30,
        ma_fast_period: int = 12,
        ma_slow_period: int = 26,
        ma_direction: str = "golden_cross",
        volume_lookback: int = 20,
        volume_min_ratio: float = 2,
    ) -> str:
        return http.post(
            "/api/v1/scanner/rules",
            {
                "clientId": client_id,
                "type": rule_type,
                "interval": interval,
                "rsiPeriod": rsi_period,
                "rsiCondition": rsi_condition,
                "rsiThreshold": rsi_threshold,
                "maFastPeriod": ma_fast_period,
                "maSlowPeriod": ma_slow_period,
                "maDirection": ma_direction,
                "volumeLookback": volume_lookback,
                "volumeMinRatio": volume_min_ratio,
            },
        )

    def list_scanner_rules(client_id: str) -> str:
        return http.get("/api/v1/scanner/rules", {"clientId": client_id})

    def delete_scanner_rule(client_id: str, rule_id: str) -> str:
        return http.delete(f"/api/v1/scanner/rules/{rule_id}", {"clientId": client_id})

    def list_scanner_results(client_id: str) -> str:
        return http.get("/api/v1/scanner/results", {"clientId": client_id})

    def detect_pump_events(
        symbol: str,
        exchange: str = "binance",
        interval: str = "1h",
        lookback_hours: float = 48,
        min_return_pct: float = 5,
        window_bars: int = 1,
        mode: str = "close_return",
        direction: str = "up",
        min_volume_ratio: float = 0,
        max_events: int = 20,
    ) -> str:
        params: dict[str, Any] = {
            "symbol": symbol,
            "exchange": exchange,
            "interval": interval,
            "lookbackHours": lookback_hours,
            "minReturnPct": min_return_pct,
            "windowBars": window_bars,
            "mode": mode,
            "direction": direction,
            "maxEvents": max_events,
        }
        if min_volume_ratio and min_volume_ratio > 0:
            params["minVolumeRatio"] = min_volume_ratio
        return http.get("/api/v1/market/pumps", params)

    def scan_pump_events(
        exchange: str = "binance",
        quote: str = "USDT",
        interval: str = "15m",
        lookback_hours: float = 24,
        min_return_pct: float = 8,
        window_bars: int = 1,
        mode: str = "close_return",
        direction: str = "up",
        min_volume_ratio: float = 0,
        symbol_limit: int = 15,
        max_total_events: int = 30,
    ) -> str:
        params: dict[str, Any] = {
            "exchange": exchange,
            "quote": quote,
            "interval": interval,
            "lookbackHours": lookback_hours,
            "minReturnPct": min_return_pct,
            "windowBars": window_bars,
            "mode": mode,
            "direction": direction,
            "symbolLimit": symbol_limit,
            "maxTotalEvents": max_total_events,
        }
        if min_volume_ratio and min_volume_ratio > 0:
            params["minVolumeRatio"] = min_volume_ratio
        return http.get("/api/v1/market/pumps/scan", params)

    tools = [
        StructuredTool.from_function(
            health,
            name="health",
            description="Check Swyngora API health.",
        ),
        StructuredTool.from_function(
            realtime_stream_info,
            name="realtime_stream_info",
            description="How to use the WebSocket for live coin prices and paper portfolio order/position updates.",
        ),
        StructuredTool.from_function(
            list_exchanges,
            name="list_exchanges",
            description="List configured market venues.",
        ),
        StructuredTool.from_function(
            list_delist_schedule,
            name="list_delist_schedule",
            description="Scheduled spot delistings (symbol + UTC time). Default exchange=binance.",
            args_schema=DelistInput,
        ),
        StructuredTool.from_function(
            get_fx_rates,
            name="get_fx_rates",
            description=(
                "Spot FX rates (units per 1 USD) to convert BIST TRY, Nasdaq USD, "
                "and crypto USDT display values. USDT is treated as USD. Display only."
            ),
        ),
        StructuredTool.from_function(
            get_ticker,
            name="get_ticker",
            description="24h ticker (price, volume, change) for a pair.",
            args_schema=TickerInput,
        ),
        StructuredTool.from_function(
            get_spot_orderbook,
            name="get_spot_orderbook",
            description=(
                "Grouped live spot order book plus analysis: pressure, imbalance, and "
                "walls from depth within ±range_pct of mid (default 2%). Each wall has "
                "behavior short | persistent (resting) | suspicious (flicker). "
                f"Works on {EXCHANGE_VENUES}."
            ),
            args_schema=OrderBookInput,
        ),
        StructuredTool.from_function(
            analyze_spot_orderbook,
            name="analyze_spot_orderbook",
            description=(
                "Buy/sell pressure, notional imbalance, and large walls from live spot "
                "depth within ±range_pct of mid. Wall behavior: short, persistent "
                "(resting support/resistance), or suspicious (added/removed often). "
                "Prefer this when the question is pressure or walls rather than the full ladder."
            ),
            args_schema=OrderBookAnalysisInput,
        ),
        StructuredTool.from_function(
            analyze_market_orderbook,
            name="analyze_market_orderbook",
            description=(
                "Market-wide buy/sell pressure for one coin: sums Binance + Coinbase + "
                "Bybit bid/ask notional only in a symmetric ±% both sides can reach "
                "(requested ±range_pct when all cover it both ways). Use for overall pressure."
            ),
            args_schema=MarketOrderBookInput,
        ),
        StructuredTool.from_function(
            get_liquidations,
            name="get_liquidations",
            description=(
                "Futures liquidations for a coin over the last 5 minutes, 1 hour, "
                "4 hours, and 24 hours. Returns long vs short notional, count, and "
                "the biggest hit. Binance USD-M + Bybit linear. exchange=all sums both. "
                "complete only counts time the websocket was actually live for that coin and venue; "
                "coverage does not grow if the stream never connects or drops."
            ),
            args_schema=LiquidationsInput,
        ),
        StructuredTool.from_function(
            get_market_liquidity,
            name="get_market_liquidity",
            description=(
                "How liquid a coin is right now. Scores resting buy/sell notional "
                "only in ±0.1 / ±0.5 / ±1% bands the book actually reaches on both "
                "sides (usedRangePct). Market-wide uses the overlap all venues can "
                "reach. 0–100 plus grade; weakerSide is the thinner side. "
                "exchange=all (default) returns Binance, Coinbase, Bybit, and a market-wide score."
            ),
            args_schema=MarketLiquidityInput,
        ),
        StructuredTool.from_function(
            estimate_market_impact,
            name="estimate_market_impact",
            description=(
                "Simulate a market buy or sell against live order-book depth. Walks asks "
                "(buy) or bids (sell) level by level. Use quantity for base size (e.g. 5 BTC) "
                "or notional for quote size (e.g. 1000000000 USDT). exchange=all (default) "
                "merges Binance+Coinbase+Bybit cheapest-first. Returns average fill, slippage, "
                "and price impact as the new best ask/bid after leftover size (0 if the touch "
                "level still has quantity). If impactAvailable is false the visible book was "
                "wiped and impact cannot be calculated. Simulation only — not a quote."
            ),
            args_schema=MarketImpactInput,
        ),
        StructuredTool.from_function(
            get_candles,
            name="get_candles",
            description="OHLCV candles for a pair.",
            args_schema=CandlesInput,
        ),
        StructuredTool.from_function(
            get_supply,
            name="get_supply",
            description="Circulating/total/max supply for a base asset.",
            args_schema=SupplyInput,
        ),
        StructuredTool.from_function(
            list_spot_markets,
            name="list_spot_markets",
            description="Search/sort spot markets (volume, mcap, tags).",
            args_schema=SpotInput,
        ),
        StructuredTool.from_function(
            get_indicators,
            name="get_indicators",
            description="RSI + EMA indicators. Informational only.",
            args_schema=IndicatorsInput,
        ),
        StructuredTool.from_function(
            analyze_swing,
            name="analyze_swing",
            description=(
                "4h+1d swing engine: Wilder RSI/ADX/ATR, SuperTrend, MACD, BTC regime, "
                "quality gates, ATR/structure stop and 1.8R target. Informational only."
            ),
            args_schema=TickerInput,
        ),
        StructuredTool.from_function(
            scan_swing_setups,
            name="scan_swing_setups",
            description="Scan the client's watchlist for quality-gated swing setups.",
            args_schema=SwingScanInput,
        ),
        StructuredTool.from_function(
            get_watchlist,
            name="get_watchlist",
            description=(
                "Get a watchlist. Optional owner_client_id reads a list shared with the actor "
                "(viewer/editor/owner). Response includes role."
            ),
            args_schema=WatchGetInput,
        ),
        StructuredTool.from_function(
            add_watchlist_item,
            name="add_watchlist_item",
            description="Add symbol to watchlist (owner or editor). Optional owner_client_id for shared lists.",
            args_schema=WatchAddInput,
        ),
        StructuredTool.from_function(
            remove_watchlist_item,
            name="remove_watchlist_item",
            description="Remove symbol from watchlist (owner or editor). Optional owner_client_id for shared lists.",
            args_schema=WatchRemoveInput,
        ),
        StructuredTool.from_function(
            share_watchlist,
            name="share_watchlist",
            description=(
                "Share your watchlist as viewer (read-only) or editor (add/remove symbols). "
                "Owner only; cannot share twice with the same user."
            ),
            args_schema=WatchShareInput,
        ),
        StructuredTool.from_function(
            update_watchlist_share,
            name="update_watchlist_share",
            description="Change role (viewer|editor) for an existing share. Owner only.",
            args_schema=WatchShareInput,
        ),
        StructuredTool.from_function(
            revoke_watchlist_share,
            name="revoke_watchlist_share",
            description="Remove a user's access to your watchlist. Owner only.",
            args_schema=WatchRevokeInput,
        ),
        StructuredTool.from_function(
            list_watchlist_shares,
            name="list_watchlist_shares",
            description="List who has access to your watchlist. Owner only.",
            args_schema=WatchClientInput,
        ),
        StructuredTool.from_function(
            list_shared_watchlists,
            name="list_shared_watchlists",
            description="List watchlists shared with this clientId.",
            args_schema=WatchClientInput,
        ),
        StructuredTool.from_function(
            list_watchlist_audit,
            name="list_watchlist_audit",
            description="List who changed the watchlist and when. Owner only.",
            args_schema=WatchAuditInput,
        ),
        StructuredTool.from_function(
            start_export,
            name="start_export",
            description=(
                "Start a background export of the user's watchlist, shares, alerts, "
                "backtests, and/or paper portfolios as json or csv. One active export per client. "
                "Poll get_export for progress; download via HTTP when completed."
            ),
            args_schema=ExportStartInput,
        ),
        StructuredTool.from_function(
            get_export,
            name="get_export",
            description="Get export job status and progressPct (0-100).",
            args_schema=ExportGetInput,
        ),
        StructuredTool.from_function(
            list_exports,
            name="list_exports",
            description="List recent data export jobs for a clientId.",
            args_schema=WatchAuditInput,
        ),
        StructuredTool.from_function(
            cancel_export,
            name="cancel_export",
            description="Cancel a pending or running data export.",
            args_schema=ExportGetInput,
        ),
        StructuredTool.from_function(
            preview_import,
            name="preview_import",
            description="Preview restoring an export (JSON/CSV text). Returns counts; then confirm_import.",
            args_schema=ImportPreviewInput,
        ),
        StructuredTool.from_function(
            confirm_import,
            name="confirm_import",
            description="Apply a previewed import (mode=merge or replace).",
            args_schema=ImportConfirmInput,
        ),
        StructuredTool.from_function(
            get_import,
            name="get_import",
            description="Import job status and section counts.",
            args_schema=ImportGetInput,
        ),
        StructuredTool.from_function(
            list_imports,
            name="list_imports",
            description="List recent import jobs.",
            args_schema=WatchClientInput,
        ),
        StructuredTool.from_function(
            cancel_import,
            name="cancel_import",
            description="Cancel a previewed or running import.",
            args_schema=ImportGetInput,
        ),
        StructuredTool.from_function(
            list_price_alerts,
            name="list_price_alerts",
            description="List one-shot price alerts for a clientId (active and triggered).",
            args_schema=AlertListInput,
        ),
        StructuredTool.from_function(
            create_price_alert,
            name="create_price_alert",
            description=(
                "Create a price alert when last price goes above or below target_price. "
                "mode=one_time (default) fires once; mode=repeating re-fires on each re-cross "
                "after price returns to the safe side."
            ),
            args_schema=AlertCreateInput,
        ),
        StructuredTool.from_function(
            create_orderbook_alert,
            name="create_orderbook_alert",
            description=(
                "Create a live order-book alert. kind=imbalance (above=buy pressure, below=sell) "
                "or kind=wall (bid|ask|any). Checked in the background; repeating by default so "
                "it does not re-fire while the same condition stays true."
            ),
            args_schema=OrderBookAlertInput,
        ),
        StructuredTool.from_function(
            delete_price_alert,
            name="delete_price_alert",
            description="Delete a price alert by id for a clientId.",
            args_schema=AlertDeleteInput,
        ),
        StructuredTool.from_function(
            get_alert_webhook,
            name="get_alert_webhook",
            description="Get the client's price-alert webhook URL.",
            args_schema=AlertWebhookGetInput,
        ),
        StructuredTool.from_function(
            set_alert_webhook,
            name="set_alert_webhook",
            description=(
                "Set absolute http(s) webhook URL for price-alert notifications. "
                "delivery_mode=immediate posts each fire soon; hourly_digest batches fires into one POST per UTC hour."
            ),
            args_schema=AlertWebhookSetInput,
        ),
        StructuredTool.from_function(
            delete_alert_webhook,
            name="delete_alert_webhook",
            description="Clear the client's price-alert webhook URL.",
            args_schema=AlertWebhookGetInput,
        ),
        StructuredTool.from_function(
            create_api_key,
            name="create_api_key",
            description="Create a named API key (read or trade). Secret is returned once.",
            args_schema=APIKeyCreateInput,
        ),
        StructuredTool.from_function(
            list_api_keys,
            name="list_api_keys",
            description="List named API keys for a client (no secrets).",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            revoke_api_key,
            name="revoke_api_key",
            description="Revoke a named API key so it stops working.",
            args_schema=APIKeyRevokeInput,
        ),
        StructuredTool.from_function(
            create_portfolio,
            name="create_portfolio",
            description="Create a named paper-trading portfolio with starting cash. Multiple books per client (e.g. Main, Risky). Simulated only.",
            args_schema=PortfolioCreateInput,
        ),
        StructuredTool.from_function(
            list_portfolios,
            name="list_portfolios",
            description="List paper portfolios (id, name, cash). Use portfolio_id on other tools to select a book.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            rename_portfolio,
            name="rename_portfolio",
            description="Rename a paper portfolio. Names must be unique per client.",
            args_schema=PortfolioRenameInput,
        ),
        StructuredTool.from_function(
            delete_portfolio,
            name="delete_portfolio",
            description="Delete a paper portfolio and all of its positions, orders, and history.",
            args_schema=PortfolioDeleteInput,
        ),
        StructuredTool.from_function(
            share_portfolio,
            name="share_portfolio",
            description="Share one of your paper books with another client as viewer (read) or trader (can place orders). Owner only.",
            args_schema=PortfolioShareInput,
        ),
        StructuredTool.from_function(
            update_portfolio_share,
            name="update_portfolio_share",
            description="Change a paper portfolio share role to viewer or trader. Owner only.",
            args_schema=PortfolioShareInput,
        ),
        StructuredTool.from_function(
            revoke_portfolio_share,
            name="revoke_portfolio_share",
            description="Remove another client's access to a paper portfolio. Owner only.",
            args_schema=PortfolioShareRevokeInput,
        ),
        StructuredTool.from_function(
            list_portfolio_shares,
            name="list_portfolio_shares",
            description="List who you shared a paper portfolio with.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            list_shared_portfolios,
            name="list_shared_portfolios",
            description="List paper portfolios shared with you and your role.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            get_portfolio,
            name="get_portfolio",
            description="Get paper portfolio cash, positions, realized/unrealized P&L. Pass portfolio_id when the client has more than one book.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            get_portfolio_performance,
            name="get_portfolio_performance",
            description="Paper portfolio equity history for 1d/1w/1m/3m plus P&L amount and percent from the start of that window.",
            args_schema=PortfolioPerformanceInput,
        ),
        StructuredTool.from_function(
            deposit_portfolio_cash,
            name="deposit_portfolio_cash",
            description="Add virtual cash to a paper portfolio. Not trading profit.",
            args_schema=PortfolioCashMoveInput,
        ),
        StructuredTool.from_function(
            withdraw_portfolio_cash,
            name="withdraw_portfolio_cash",
            description="Withdraw available virtual cash from a paper portfolio. Not trading loss.",
            args_schema=PortfolioCashMoveInput,
        ),
        StructuredTool.from_function(
            transfer_portfolio_cash,
            name="transfer_portfolio_cash",
            description="Move virtual cash between your own paper portfolios. Owner only. Not a deposit or withdrawal.",
            args_schema=PortfolioTransferInput,
        ),
        StructuredTool.from_function(
            list_portfolio_cash_movements,
            name="list_portfolio_cash_movements",
            description="List paper deposits and withdrawals (newest first).",
            args_schema=PortfolioCashListInput,
        ),
        StructuredTool.from_function(
            get_portfolio_risk_limits,
            name="get_portfolio_risk_limits",
            description="Get optional paper risk limits and live daily-loss / concentration status.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            set_portfolio_risk_limits,
            name="set_portfolio_risk_limits",
            description="Set optional risk limits (daily loss % and/or max coin weight %). Does not close positions.",
            args_schema=RiskLimitsSetInput,
        ),
        StructuredTool.from_function(
            clear_portfolio_risk_limits,
            name="clear_portfolio_risk_limits",
            description="Remove all paper risk limits.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            get_paper_trading_costs,
            name="get_paper_trading_costs",
            description=(
                "Paper taker fee and slippage rates per exchange. Fills use slipped last price; "
                "buy cash/lot cost include the fee; sell PnL is after the fee."
            ),
            args_schema=PaperTradingCostsInput,
        ),
        StructuredTool.from_function(
            place_portfolio_order,
            name="place_portfolio_order",
            description=(
                "Paper market buy/sell. Fill is last price plus adverse slippage; a taker fee is charged. "
                "Buy lot cost includes the fee; sell realized PnL is after the fee. "
                "Pass idempotency_key on retries so a timeout does not create a second fill. Not real money."
            ),
            args_schema=PortfolioOrderInput,
        ),
        StructuredTool.from_function(
            place_portfolio_pending_order,
            name="place_portfolio_pending_order",
            description=(
                "Paper pending order: limit_buy, limit_sell, or stop_loss. "
                "time_in_force gtc|ioc|fok; optional expires_at for GTC. "
                "Pass idempotency_key on retries. Simulated only."
            ),
            args_schema=PortfolioPendingOrderInput,
        ),
        StructuredTool.from_function(
            list_portfolio_orders,
            name="list_portfolio_orders",
            description="List paper pending orders (default status=open).",
            args_schema=PortfolioListOrdersInput,
        ),
        StructuredTool.from_function(
            cancel_portfolio_order,
            name="cancel_portfolio_order",
            description="Cancel an open paper pending order. Canceled orders never fill.",
            args_schema=PortfolioCancelOrderInput,
        ),
        StructuredTool.from_function(
            get_portfolio_order,
            name="get_portfolio_order",
            description="Get one paper pending order plus last price and amend hints.",
            args_schema=PortfolioOrderGetInput,
        ),
        StructuredTool.from_function(
            place_portfolio_oco_order,
            name="place_portfolio_oco_order",
            description="Paper OCO: take-profit limit + stop-loss for the same quantity.",
            args_schema=PortfolioOCOInput,
        ),
        StructuredTool.from_function(
            place_portfolio_bracket_order,
            name="place_portfolio_bracket_order",
            description="Paper bracket: limit-buy entry with pending take-profit + stop-loss.",
            args_schema=PortfolioBracketInput,
        ),
        StructuredTool.from_function(
            amend_portfolio_order,
            name="amend_portfolio_order",
            description="Amend open GTC limit/stop trigger price and/or remaining size.",
            args_schema=PortfolioAmendInput,
        ),
        StructuredTool.from_function(
            cancel_all_portfolio_orders,
            name="cancel_all_portfolio_orders",
            description="Cancel all open paper orders, or one market when symbol is set.",
            args_schema=PortfolioCancelAllInput,
        ),
        StructuredTool.from_function(
            list_portfolio_trades,
            name="list_portfolio_trades",
            description="List paper trade history for a clientId.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            list_portfolio_lots,
            name="list_portfolio_lots",
            description="List paper tax lots (remaining buys). Sells match FIFO or LIFO.",
            args_schema=PortfolioLotsInput,
        ),
        StructuredTool.from_function(
            create_recurring_buy,
            name="create_recurring_buy",
            description=(
                "Create a named paper recurring buy: cash amount at market on "
                "daily|weekly|monthly|interval. weekday for weekly, dayOfMonth for salary day, "
                "intervalHours (e.g. 12) for interval. Simulated only."
            ),
            args_schema=RecurringBuyCreateInput,
        ),
        StructuredTool.from_function(
            update_recurring_buy,
            name="update_recurring_buy",
            description="Update a paper recurring buy name, amount, or schedule.",
            args_schema=RecurringBuyUpdateInput,
        ),
        StructuredTool.from_function(
            list_recurring_buys,
            name="list_recurring_buys",
            description="List paper recurring buy plans for a clientId.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            get_recurring_buy,
            name="get_recurring_buy",
            description="Get one paper recurring buy plan by id.",
            args_schema=RecurringBuyPlanInput,
        ),
        StructuredTool.from_function(
            pause_recurring_buy,
            name="pause_recurring_buy",
            description="Pause a paper recurring buy plan.",
            args_schema=RecurringBuyPlanInput,
        ),
        StructuredTool.from_function(
            resume_recurring_buy,
            name="resume_recurring_buy",
            description="Resume a paused paper recurring buy plan.",
            args_schema=RecurringBuyPlanInput,
        ),
        StructuredTool.from_function(
            delete_recurring_buy,
            name="delete_recurring_buy",
            description="Delete a paper recurring buy plan and its run history.",
            args_schema=RecurringBuyPlanInput,
        ),
        StructuredTool.from_function(
            list_recurring_buy_runs,
            name="list_recurring_buy_runs",
            description="List execution history for a paper recurring buy plan.",
            args_schema=RecurringBuyRunsInput,
        ),
        StructuredTool.from_function(
            create_portfolio_basket,
            name="create_portfolio_basket",
            description="Create a named paper allocation basket (target % mix). Does not trade.",
            args_schema=BasketCreateInput,
        ),
        StructuredTool.from_function(
            list_portfolio_baskets,
            name="list_portfolio_baskets",
            description="List saved paper allocation baskets.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            get_portfolio_basket,
            name="get_portfolio_basket",
            description="Get a basket with live actual vs target weights (no trades).",
            args_schema=BasketIdInput,
        ),
        StructuredTool.from_function(
            update_portfolio_basket,
            name="update_portfolio_basket",
            description="Update basket name/targets. Does not trade.",
            args_schema=BasketUpdateInput,
        ),
        StructuredTool.from_function(
            delete_portfolio_basket,
            name="delete_portfolio_basket",
            description="Delete a paper allocation basket. Does not trade.",
            args_schema=BasketIdInput,
        ),
        StructuredTool.from_function(
            preview_portfolio_rebalance,
            name="preview_portfolio_rebalance",
            description="Preview sells/buys to move toward a basket. Does not trade. Never automatic.",
            args_schema=BasketIdInput,
        ),
        StructuredTool.from_function(
            rebalance_portfolio_basket,
            name="rebalance_portfolio_basket",
            description="USER-TRIGGERED paper rebalance toward a basket. Drift is allowed until this is called.",
            args_schema=BasketIdInput,
        ),
        StructuredTool.from_function(
            place_margin_order,
            name="place_margin_order",
            description=(
                "Paper isolated margin open: long|short, leverage 1-10, market or limit. "
                "Optional stop_loss/take_profit. Pass idempotency_key on retries. Simulated only."
            ),
            args_schema=MarginOrderInput,
        ),
        StructuredTool.from_function(
            list_margin_positions,
            name="list_margin_positions",
            description="List open paper margin positions with marks and liquidation price.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            close_margin_position,
            name="close_margin_position",
            description="Close all or part of a paper margin position at market. Pass idempotency_key on retries.",
            args_schema=MarginCloseInput,
        ),
        StructuredTool.from_function(
            set_margin_brackets,
            name="set_margin_brackets",
            description="Set or clear stop-loss / take-profit on an open paper margin position.",
            args_schema=MarginBracketsInput,
        ),
        StructuredTool.from_function(
            list_margin_orders,
            name="list_margin_orders",
            description="List paper margin orders (default status=open).",
            args_schema=PortfolioListOrdersInput,
        ),
        StructuredTool.from_function(
            cancel_margin_order,
            name="cancel_margin_order",
            description="Cancel an open paper margin limit order.",
            args_schema=PortfolioCancelOrderInput,
        ),
        StructuredTool.from_function(
            list_margin_trades,
            name="list_margin_trades",
            description="List paper margin trade history.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            create_price_diff_watch,
            name="create_price_diff_watch",
            description=(
                "Track cross-exchange price differences for a coin (Binance/Coinbase/Bybit). "
                "Opens opportunities when net edge after fees exceeds min_net_diff_pct."
            ),
            args_schema=PriceDiffWatchCreateInput,
        ),
        StructuredTool.from_function(
            list_price_diff_watches,
            name="list_price_diff_watches",
            description="List cross-exchange price difference watches for a clientId.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            get_price_diff_watch,
            name="get_price_diff_watch",
            description="Get one price-diff watch by id.",
            args_schema=PriceDiffWatchIdInput,
        ),
        StructuredTool.from_function(
            delete_price_diff_watch,
            name="delete_price_diff_watch",
            description="Delete a price-diff watch and its opportunities.",
            args_schema=PriceDiffWatchIdInput,
        ),
        StructuredTool.from_function(
            list_price_diff_opportunities,
            name="list_price_diff_opportunities",
            description="List price-diff opportunities (status open|closed|all).",
            args_schema=PriceDiffOppListInput,
        ),
        StructuredTool.from_function(
            get_price_diff_opportunity,
            name="get_price_diff_opportunity",
            description="Get one price-diff opportunity by id.",
            args_schema=PriceDiffOppIdInput,
        ),
        StructuredTool.from_function(
            create_scanner_rule,
            name="create_scanner_rule",
            description=(
                "Create a technical scanner rule for the client's watchlist: "
                "rsi, ma_crossover, or volume_increase. Informational only."
            ),
            args_schema=ScannerRuleCreateInput,
        ),
        StructuredTool.from_function(
            list_scanner_rules,
            name="list_scanner_rules",
            description="List technical scanner rules for a clientId.",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            delete_scanner_rule,
            name="delete_scanner_rule",
            description="Delete a scanner rule by id.",
            args_schema=ScannerRuleDeleteInput,
        ),
        StructuredTool.from_function(
            list_scanner_results,
            name="list_scanner_results",
            description="List saved scanner match history (deduped by rule/symbol/bar).",
            args_schema=PortfolioGetInput,
        ),
        StructuredTool.from_function(
            detect_pump_events,
            name="detect_pump_events",
            description=(
                "Detect pump/dump events on ONE symbol from candles. "
                "Set min_return_pct, lookback_hours, interval (1m/5m/15m/1h…), "
                "window_bars, mode (close_return|candle_body|high_from_low), direction. "
                "Mechanical filter only — not a trade signal."
            ),
            args_schema=PumpDetectInput,
        ),
        StructuredTool.from_function(
            scan_pump_events,
            name="scan_pump_events",
            description=(
                "Scan top-volume symbols for recent pumps/dumps with thresholds "
                "(min_return_pct, lookback_hours, interval, mode, direction, max_total_events). "
                "max_total_events caps aggregate events across symbols. "
                "Response metadata echoes resolved defaults. "
                "Use for 'what pumped in the last N hours'."
            ),
            args_schema=PumpScanInput,
        ),
    ]
    return filter_tools(tools, pack)
