"""Market tools via Swyngora HTTP API (mirror of Go MCP tools)."""

from __future__ import annotations

import json
import re
import threading
from collections.abc import Callable
from concurrent.futures import Future, ThreadPoolExecutor
from contextvars import ContextVar, copy_context
from dataclasses import dataclass
from typing import Any, TypeVar

import httpx2
from langchain_core.tools import BaseTool, StructuredTool
from pydantic import BaseModel, Field

from swyngora_ai.config import Settings, get_settings
from swyngora_ai.constants import EXCHANGE_VENUES, EXCHANGE_VENUES_OR_ALL
from swyngora_ai.tools.errors import ToolTransientError
from swyngora_ai.tools.packs import filter_tools

bound_client_id: ContextVar[str] = ContextVar("bound_client_id", default="")
# Defaults True for CLI/local so tools work without an HTTP scope envelope.
# Bound CLI/legacy turns default to trade. Unbound workers use _AMBIGUOUS_BIND
# (deny). Orchestrator always calls bind_tool_scope for HTTP-scoped chats.
bound_can_trade: ContextVar[bool] = ContextVar("bound_can_trade", default=True)
bound_can_manage_keys: ContextVar[bool] = ContextVar("bound_can_manage_keys", default=True)
_turn_tool_json: ContextVar[list[str] | None] = ContextVar("turn_tool_json", default=None)


@dataclass(frozen=True)
class _ToolBind:
    client_id: str
    can_trade: bool
    can_manage_keys: bool
    ambiguous: bool = False


# LangChain / desk pool threads may not inherit ContextVars (same hole
# progress.py documents). Each chat still pushes a stack frame so a *single*
# in-flight bind can be recovered. Merge from this task's ContextVars, never
# from stack[-1] (that is the other tenant under overlapping chats). Pop the
# matching frame. If two tenants are on the stack and this task has no
# ContextVar, refuse rather than guess X-Client-Id.
_bind_lock = threading.Lock()
_bind_stack: list[_ToolBind] = []
_bind_live: ContextVar[bool] = ContextVar("bind_live", default=False)
_P = TypeVar("_P")
_AMBIGUOUS_BIND = _ToolBind("", False, False, ambiguous=True)
_UNBOUND = _ToolBind("", False, False, ambiguous=False)


def _context_bind() -> _ToolBind:
    return _ToolBind(
        client_id=bound_client_id.get(),
        can_trade=bound_can_trade.get(),
        can_manage_keys=bound_can_manage_keys.get(),
    )


def _push_bind(
    *,
    client_id: str | None = None,
    can_trade: bool | None = None,
    can_manage_keys: bool | None = None,
) -> _ToolBind:
    prev = _context_bind()
    frame = _ToolBind(
        client_id=prev.client_id if client_id is None else client_id,
        can_trade=prev.can_trade if can_trade is None else can_trade,
        can_manage_keys=prev.can_manage_keys if can_manage_keys is None else can_manage_keys,
    )
    with _bind_lock:
        _bind_stack.append(frame)
    return frame


def _pop_matching(match: Callable[[_ToolBind], bool]) -> None:
    with _bind_lock:
        for i in range(len(_bind_stack) - 1, -1, -1):
            if match(_bind_stack[i]):
                _bind_stack.pop(i)
                return


def _active_bind() -> _ToolBind:
    """Use this task's ContextVars only. Never guess from the process stack.

    LangChain/desk pool workers must go through submit_with_bound_context.
    A missing bind fails closed so overlapping chats cannot inherit
    another tenant's clientId or can_trade flag.
    """
    if _bind_live.get() or bound_client_id.get():
        return _context_bind()
    with _bind_lock:
        clients = {b.client_id for b in _bind_stack if b.client_id}
        scopes = {(b.can_trade, b.can_manage_keys) for b in _bind_stack}
        if len(clients) > 1 or len(scopes) > 1:
            return _AMBIGUOUS_BIND
    # No ContextVar: do not guess tenant/scope from the stack (that leaked
    # can_trade across overlapping same-tenant chats). Mutating tools deny.
    return _UNBOUND


def submit_with_bound_context(
    pool: ThreadPoolExecutor, fn: Callable[..., _P], /, *args: Any, **kwargs: Any
) -> Future[_P]:
    """Run fn on a pool worker with this task's ContextVars (tenant + scope)."""
    ctx = copy_context()
    if kwargs:
        return pool.submit(ctx.run, lambda: fn(*args, **kwargs))  # ty: ignore[invalid-return-type]
    return pool.submit(ctx.run, fn, *args)  # ty: ignore[invalid-return-type]


_READ_ONLY_DENY = (
    "ERROR 403: this AI session is read-only; a trade-permission API key is required "
    "for portfolio, alert, and other mutating tools"
)
_KEY_ADMIN_DENY = (
    "ERROR 403: API key and account-admin tools are not available in this AI session "
    "(user keys cannot manage other keys)"
)
_AMBIGUOUS_DENY = (
    "ERROR 403: tenant bind is ambiguous under concurrent chats; refusing to guess X-Client-Id"
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
    cid = (client_id or "").strip()
    _push_bind(client_id=cid)
    return bound_client_id.set(cid), _bind_live.set(True)


def reset_bound_client_id(token: Any) -> None:
    cid = bound_client_id.get()
    _pop_matching(lambda b: b.client_id == cid)
    if isinstance(token, tuple) and len(token) == 2:
        bound_client_id.reset(token[0])
        _bind_live.reset(token[1])
        return
    bound_client_id.reset(token)


def bind_tool_scope(*, can_trade: bool = True, can_manage_keys: bool = True) -> tuple[Any, ...]:
    """Bind trade + key-admin scope for one chat turn (returns tokens for reset)."""
    _push_bind(can_trade=bool(can_trade), can_manage_keys=bool(can_manage_keys))
    return (
        bound_can_trade.set(bool(can_trade)),
        bound_can_manage_keys.set(bool(can_manage_keys)),
        _bind_live.set(True),
    )


def reset_tool_scope(tokens: tuple[Any, ...]) -> None:
    trade_tok, keys_tok = tokens[0], tokens[1]
    want = _context_bind()
    _pop_matching(
        lambda b: (
            b.client_id == want.client_id
            and b.can_trade == want.can_trade
            and b.can_manage_keys == want.can_manage_keys
        )
    )
    bound_can_trade.reset(trade_tok)
    bound_can_manage_keys.reset(keys_tok)
    if len(tokens) > 2:
        _bind_live.reset(tokens[2])


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
        self._client = httpx2.Client(timeout=timeout)

    def _headers(self) -> dict[str, str]:
        headers: dict[str, str] = {}
        if self.auth_token:
            headers["Authorization"] = f"Bearer {self.auth_token}"
        cid = _active_bind().client_id
        if cid:
            headers["X-Client-Id"] = cid
        return headers

    def _apply_client_id(self, params: dict[str, Any] | None, body: dict[str, Any] | None) -> None:
        cid = _active_bind().client_id
        if not cid:
            return
        if params is not None:
            params["clientId"] = cid
        if body is not None and "clientId" in body:
            body["clientId"] = cid

    def _scope_error(self, path: str, *, mutating: bool) -> str | None:
        bind = _active_bind()
        if bind.ambiguous:
            return _AMBIGUOUS_DENY
        if _is_key_or_account_admin_path(path) and not bind.can_manage_keys:
            return _KEY_ADMIN_DENY
        if mutating and not bind.can_trade:
            return _READ_ONLY_DENY
        return None

    def _gate(self, path: str, *, mutating: bool) -> str | None:
        """Reject path-id traversal and apply scope to the normalized URL path."""
        raw = path or ""
        head = raw.split("?", 1)[0]
        if ".." in head or "\\" in head:
            return "ERROR 400: invalid path id"
        try:
            norm = httpx2.URL(raw if "://" in raw else f"http://local{raw}").path
        except (httpx2.InvalidURL, ValueError, TypeError):
            return "ERROR 400: invalid path"
        if err := self._scope_error(raw, mutating=mutating):
            return err
        return self._scope_error(norm, mutating=mutating)

    def _body_text(self, r: httpx2.Response, *, retryable: bool = False) -> str:
        if r.status_code >= 400:
            if retryable and (r.status_code == 429 or r.status_code >= 500):
                raise ToolTransientError(f"{r.status_code}: {r.text[:500]}")
            return f"ERROR {r.status_code}: {r.text[:500]}"
        try:
            text = format_tool_json(r.json())
        except (ValueError, TypeError):
            text = _SWY_SECRET_RE.sub("[redacted]", r.text)
        record_tool_json(text)
        return text

    def _request(self, method: str, url: str, *, retryable: bool = False, **kwargs: Any) -> str:
        try:
            r = self._client.request(method, url, **kwargs)
        except httpx2.TimeoutException as e:
            msg = f"timeout: {e}"
            if retryable:
                raise ToolTransientError(msg[:500]) from e
            return f"ERROR {msg}"[:500]
        except httpx2.ConnectError as e:
            msg = f"connect: {e}"
            if retryable:
                raise ToolTransientError(msg[:500]) from e
            return f"ERROR {msg}"[:500]
        except httpx2.HTTPError as e:
            msg = f"{type(e).__name__}: {e}"
            if retryable:
                raise ToolTransientError(msg[:500]) from e
            return f"ERROR {msg}"[:500]
        return self._body_text(r, retryable=retryable)

    def get(self, path: str, params: dict[str, Any] | None = None) -> str:
        if err := self._gate(path, mutating=False):
            return err
        params = dict(params or {})
        self._apply_client_id(params, None)
        return self._request(
            "GET",
            f"{self.base}{path}",
            retryable=True,
            params=params,
            headers=self._headers(),
        )

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


class CustomSignalProfileInput(BaseModel):
    client_id: str
    name: str
    weights_json: str = Field(
        description='JSON object of factor id to percent summing to 100, e.g. {"hunt_up":40,"rsi_recovery":60}'
    )
    is_default: bool = Field(default=False, description="Open Custom tab on this mix")


class CustomSignalProfileUpdateInput(BaseModel):
    client_id: str
    profile_id: str
    name: str = ""
    weights_json: str = Field(default="", description="JSON object of factor id to percent")
    is_default: bool | None = None


class CustomSignalProfileDeleteInput(BaseModel):
    client_id: str
    profile_id: str


class CustomSignalScanInput(BaseModel):
    client_id: str
    profile_id: str = ""
    exchange: str = ""
    limit: int = Field(default=25, ge=1, le=25)
    weight_ema_crossover: float | None = None
    weight_ema_trend: float | None = None
    weight_supertrend: float | None = None
    weight_rsi_recovery: float | None = None
    weight_macd: float | None = None
    weight_adx: float | None = None
    weight_accumulation: float | None = None
    weight_volume_breakout: float | None = None
    weight_hunt_up: float | None = None


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


class IcebergsInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or BTC-USD")
    exchange: str = Field(
        default="all",
        description="binance|coinbase|bybit|all (default all)",
    )
    min_notional: float = Field(
        default=0,
        description="Minimum visible clip in USD (default 25000)",
    )


class OrderBookLiquidityFlowInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|coinbase|bybit|all (default all)",
    )
    window: str = Field(
        default="5m",
        description="5m|15m|1h (default 5m). Ignored when start_time/end_time are set.",
    )
    start_time: str = Field(
        default="",
        description="Range start (RFC3339 or unix ms). Max span 1h.",
    )
    end_time: str = Field(
        default="",
        description="Range end (RFC3339 or unix ms)",
    )
    min_price: float = Field(
        default=0,
        description="Only include this price and above (e.g. 65400).",
    )
    max_price: float = Field(
        default=0,
        description="Only include this price and below (e.g. 65600).",
    )


class OrderBookHistoryInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or BTC-USD")
    exchange: str = Field(default="binance", description="binance|coinbase|bybit")
    at: str = Field(default="", description="Point in time RFC3339 or unix ms")
    from_time: str = Field(default="", description="List window start")
    to_time: str = Field(default="", description="List window end")
    limit: int = Field(default=0, description="Max list rows (default 60)")


class OrderBookCompareInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or BTC-USD")
    exchange: str = Field(default="binance", description="binance|coinbase|bybit")
    from_time: str = Field(description="Earlier time RFC3339 or unix ms")
    to_time: str = Field(description="Later time RFC3339 or unix ms")


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


class LiquidationLevelsInput(BaseModel):
    symbol: str = Field(
        default="all",
        description="Pair e.g. BTCUSDT, or all for market-wide time bars",
    )
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = combined)",
    )
    range: str = Field(
        default="24h",
        description="12h|24h|3d|7d (totals clamp to 24h)",
    )


class LiquidationCascadeInput(BaseModel):
    symbol: str = Field(
        default="all",
        description="Pair e.g. BTCUSDT, or all for market-wide pooled risk",
    )
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all)",
    )


class LiquidationCascadeScanInput(BaseModel):
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all)",
    )


class LiquidationOverviewInput(BaseModel):
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both venues)",
    )
    window: str = Field(
        default="24h",
        description="1h|4h|12h|24h — ranks coins (default 24h)",
    )
    limit: int = Field(
        default=50,
        description="Max coins (default 50, max 100)",
    )


class OptionsOverviewInput(BaseModel):
    asset: str = Field(description="BTC or ETH (BTCUSDT / ETHUSDT accepted)")
    exchange: str = Field(default="deribit", description="deribit (default)")


class OpenInterestInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both venues)",
    )


class SpotPerpLiquidityInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    range_pct: float = Field(
        default=1,
        description="Headline ± percent of mid (default 1, max 1). Bands are 0.1 / 0.5 / 1.",
    )


class PerpDepthInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both + combined)",
    )
    range_pct: float = Field(
        default=1,
        description="Headline ± percent of mid (default 1, max 5)",
    )


class DerivativesBoardInput(BaseModel):
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all)",
    )
    limit: int = Field(
        default=36,
        description="How many liquid perpetuals (default 36, max 100)",
    )


class PerpBookHistoryInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all)",
    )
    window_sec: int = Field(
        default=86400,
        description="Lookback in seconds (default 24h, max 30 days)",
    )


class PerpDepthOverviewInput(BaseModel):
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both + combined per coin)",
    )
    range_pct: float = Field(
        default=1,
        description="Headline ± percent of mid (default 1, max 5)",
    )
    limit: int = Field(
        default=20,
        description="How many liquid perps to include (default 20, max 40)",
    )


class LiquidationHuntInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both venues, never averaged)",
    )


class LiquidationHeatmapInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    range: str = Field(
        default="24h",
        description=(
            "12h | 24h | 3d | 7d | 14d | 30d | 90d (default 24h). "
            "90d is one column per day. Binance public open interest "
            "and long/short history are 30 days."
        ),
    )
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all = three venues + combined)",
    )


class LiquidationPrintHeatmapInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    range: str = Field(
        default="24h",
        description=(
            "12h | 24h | 3d | 7d | 14d | 30d | 90d (default 24h). "
            "Stored force-order prints only. Public refill is about 7 days."
        ),
    )
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all = three venues + combined)",
    )


class LiquidationOpenLevelsInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    range: str = Field(
        default="24h",
        description=(
            "12h | 24h | 3d | 7d | 14d | 30d | 90d (default 24h). "
            "Levels stay until a later candle trades through them."
        ),
    )
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all = three venues + combined)",
    )


class SqueezeRiskInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both + combined)",
    )


class PositioningInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both + combined market direction)",
    )


class VenueDivergenceInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")


class TakerFlowInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both + combined)",
    )


class CvdInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all = venues + combined)",
    )
    range: str = Field(
        default="24h",
        description="24h | 7d | 30d | 90d. 24h is 5-minute bars. 7d is 30-minute, 30d is 2-hour, 90d is 4-hour, from stored minutes. Pick a range that reaches anchor.",
    )
    anchor: str = Field(
        default="",
        description="Optional RFC3339 candle time. Adds a click anchor that sums bars from that time. Empty keeps the day and 4h anchors only.",
    )


class PositioningAnchorInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    anchor: str = Field(description="RFC3339 candle time. Restarts open interest, funding, and liquidation notional from that bar.")
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all)",
    )
    bucket: str = Field(
        default="",
        description="5m | 30m | 2h | 4h. Empty follows the from..to span the same way the coin chart does.",
    )
    start: str = Field(default="", description="Optional RFC3339 start of the stored window")
    end: str = Field(default="", description="Optional RFC3339 end of the stored window")


class TapeBacktestInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="cvd, liquidation, funding, oi, and size: binance|bybit|okx|all. book: binance|coinbase|bybit|all. gap: binance|okx|all.",
    )
    condition: str = Field(
        default="cvd",
        description="cvd | liquidation | book | funding | oi | gap | size",
    )
    side: str = Field(
        default="",
        description="buy|sell, long|short, bid|ask, up|down, top|crowd, or whale|retail. Empty uses the condition default.",
    )
    threshold: float = Field(
        default=0,
        description="Share 0.55-0.90, liquidation multiple 2-12, funding percent 0.005-0.20, open-interest percent 1-50, crowd gap 1-20 points, or large-print share 0.15-0.80. 0 uses the default.",
    )
    start: str = Field(default="", description="RFC3339 start. Empty is 7 days before end.")
    end: str = Field(default="", description="RFC3339 end. Empty is now.")


class RSIHeatmapInput(BaseModel):
    exchange: str = Field(
        default="binance",
        description="binance|coinbase|bybit|nasdaq|bist (default binance)",
    )
    quote: str = Field(default="USDT", description="Quote asset")
    interval: str = Field(
        default="1h",
        description="Candle interval (default 1h)",
    )
    limit: int = Field(default=0, description="How many top pairs (default 24, max 40)")
    period: int = Field(default=0, description="RSI period (default 14)")
    sort: str = Field(
        default="",
        description="quoteVolume or marketCapCirculating",
    )


class PrintSizeOverviewInput(BaseModel):
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all)",
    )
    quote: str = Field(default="USDT", description="Quote asset")
    limit: int = Field(
        default=0,
        description="How many names from the recorder list (default 100, max 100)",
    )


class OrderCountOverviewInput(BaseModel):
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all)",
    )
    quote: str = Field(default="USDT", description="Quote asset")
    limit: int = Field(
        default=0,
        description="How many names from the recorder list (default 100, max 100)",
    )


class VolumeSurgeScanInput(BaseModel):
    exchange: str = Field(
        default="binance",
        description="binance|bybit|all (default binance)",
    )
    quote: str = Field(default="USDT", description="Quote asset")
    min_ratio: float = Field(
        default=0,
        description="Minimum current/typical to include (default 2)",
    )
    limit: int = Field(
        default=0,
        description="How many top-volume coins to scan (default 30, max 50)",
    )


class AroundMovesInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all)",
    )
    lookback: str = Field(
        default="24h",
        description="4h|12h|24h|3d|7d (default 24h)",
    )
    interval: str = Field(
        default="15m",
        description="15m|1h (default 15m)",
    )
    direction: str = Field(
        default="both",
        description="up|down|both (default both)",
    )
    min_return_pct: float = Field(
        default=0,
        description="Minimum |return| percent (default 1.5 on 15m, 2.5 on 1h)",
    )
    limit: int = Field(
        default=0,
        description="How many moves to keep (default 8, max 15)",
    )
    window: str = Field(
        default="1h",
        description="Around lookback 15m|30m|1h|2h|4h (default 1h)",
    )
    during: str = Field(
        default="",
        description="Override move window 5m|15m|30m|1h (default from the leg)",
    )


class AroundSimilarInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all)",
    )
    lookback: str = Field(
        default="7d",
        description="4h|12h|24h|3d|7d (default 7d)",
    )
    interval: str = Field(
        default="15m",
        description="15m|1h (default 15m)",
    )
    direction: str = Field(
        default="both",
        description="up|down|both (default both)",
    )
    min_return_pct: float = Field(
        default=0,
        description="Minimum |return| percent for past moves",
    )
    limit: int = Field(
        default=0,
        description="How many nearest cases (default 5, max 10)",
    )
    window: str = Field(
        default="1h",
        description="Current/before window 15m|30m|1h|2h|4h (default 1h)",
    )
    during: str = Field(
        default="",
        description="Override past move window",
    )
    fields: str = Field(
        default="",
        description="Tape to compare: price,volume,takers,oi,book,options (default all). Example: volume,book,oi",
    )
    weights: str = Field(
        default="",
        description="Importance per field, e.g. book:3,oi:3,volume:1 (defaults: book and oi 3, options 2, takers 1.5, volume and price 1)",
    )
    min_coverage: float = Field(
        default=-1,
        description="Minimum % of selected data that must be present (default 60). Below that goes to skipped, not matches. 0 disables the floor.",
    )
    horizons: str = Field(
        default="",
        description="After-windows CSV, e.g. 5m,30m,2h (default 15m,1h,4h). Matching candle size. Each reports avg, median, up, down, sample, and similarity bands 40-60/60-80/80-100.",
    )
    at: str = Field(
        default="",
        description="Optional event time (RFC3339 or unix ms). When set, compares the tape before this time instead of the current last window.",
    )


class ForensicsInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    at: str = Field(
        description="Event time (RFC3339 or unix ms). Start of the during window.",
    )
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both + combined)",
    )
    window: str = Field(
        default="1h",
        description="Before/after lookback: 15m|30m|1h|2h|4h (default 1h)",
    )
    during: str = Field(
        default="15m",
        description="Move window starting at at: 5m|15m|30m|1h (default 15m)",
    )
    lookback: str = Field(
        default="7d",
        description="4h|12h|24h|3d|7d (default 7d)",
    )
    interval: str = Field(
        default="15m",
        description="15m|1h (default 15m)",
    )
    direction: str = Field(
        default="both",
        description="up|down|both (default both)",
    )
    min_return_pct: float = Field(
        default=0,
        description="Minimum |return| percent for past moves",
    )
    limit: int = Field(
        default=0,
        description="How many nearest cases (default 5, max 10)",
    )
    fields: str = Field(
        default="",
        description="Tape to compare: price,volume,takers,oi,book,options (default all)",
    )
    weights: str = Field(
        default="",
        description="Importance per field, e.g. book:3,oi:3,volume:1",
    )
    min_coverage: float = Field(
        default=-1,
        description="Minimum % of selected data that must be present (default 60)",
    )
    horizons: str = Field(
        default="",
        description="After-windows CSV, e.g. 15m,1h,4h",
    )


class ForensicsScanInput(BaseModel):
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all)",
    )
    quote: str = Field(default="USDT", description="Quote asset (default USDT)")
    lookback: str = Field(
        default="3d",
        description="4h|12h|24h|3d|7d (default 3d)",
    )
    interval: str = Field(default="15m", description="15m|1h")
    direction: str = Field(default="both", description="up|down|both")
    min_return_pct: float = Field(default=0, description="Minimum |return| percent")
    limit: int = Field(default=0, description="How many top-volume coins (default 12, max 20)")
    window: str = Field(default="1h", description="Current/before window")
    during: str = Field(default="", description="Override past move window")
    fields: str = Field(default="", description="Tape to compare")
    weights: str = Field(default="", description="Importance per field")
    min_coverage: float = Field(default=-1, description="Minimum coverage percent")
    horizons: str = Field(default="", description="After-windows CSV")


class SetupRadarReviewInput(BaseModel):
    exchange: str = Field(default="all", description="binance | bybit | all")
    symbol: str = Field(default="", description="Optional pair filter")
    since: str = Field(default="7d", description="24h | 7d | 30d")
    limit: int = Field(default=50, description="How many saved hits")


class AroundCompareInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    from_time: str = Field(
        description="First event time (RFC3339 or unix ms)",
    )
    to_time: str = Field(
        description="Second event time (RFC3339 or unix ms)",
    )
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both + combined)",
    )
    window: str = Field(
        default="1h",
        description="Before/after lookback: 15m|30m|1h|2h|4h (default 1h)",
    )
    during: str = Field(
        default="15m",
        description="Move window starting at each time: 5m|15m|30m|1h (default 15m)",
    )


class AroundInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    at: str = Field(
        description="Event time (RFC3339 or unix ms). Start of the during window.",
    )
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both + combined)",
    )
    window: str = Field(
        default="1h",
        description="Before/after lookback: 15m|30m|1h|2h|4h (default 1h)",
    )
    during: str = Field(
        default="15m",
        description="Move window starting at at: 5m|15m|30m|1h (default 15m)",
    )


class VWAPInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both + combined)",
    )
    window: str = Field(
        default="24h",
        description="1h|4h|24h|7d|30d (default 24h). Ignored when start_time is set.",
    )
    start_time: str = Field(
        default="",
        description="Range start (RFC3339 or unix ms)",
    )
    end_time: str = Field(
        default="",
        description="Range end (RFC3339 or unix ms). Default now.",
    )


class VolumeProfileInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all). combined only when every venue reaches the window open.",
    )
    window: str = Field(
        default="24h",
        description="1h|4h|24h|7d|30d (default 24h). Ignored when start_time/end_time are set.",
    )
    start_time: str = Field(
        default="",
        description="Range start (RFC3339 or unix ms)",
    )
    end_time: str = Field(
        default="",
        description="Range end (RFC3339 or unix ms)",
    )
    tick_size: float = Field(
        default=0,
        description="Price row width (e.g. 50). Omit or 0 to pick one automatically.",
    )


class FlowAtPricesInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both + combined)",
    )
    window: str = Field(
        default="15m",
        description="5m|15m|1h|4h|1d|3d|7d (default 15m). Ignored when start_time/end_time are set.",
    )
    start_time: str = Field(
        default="",
        description="Range start (RFC3339 or unix ms). Max span 7d. Longer than the live tape uses stored minutes.",
    )
    end_time: str = Field(
        default="",
        description="Range end (RFC3339 or unix ms)",
    )
    tick_size: float = Field(
        default=0,
        description="Price row width (e.g. 10). Omit or 0 to pick one automatically.",
    )
    min_price: float = Field(
        default=0,
        description="Only include prints at or above this price (e.g. 65400).",
    )
    max_price: float = Field(
        default=0,
        description="Only include prints at or below this price (e.g. 65600).",
    )


class CandleFootprintInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all). Combined only when every venue recorded.",
    )
    interval: str = Field(description="Candle size, same as the chart: 1m, 5m, 15m, 1h, 4h, 1d, 1w, 1M")
    start_time: str = Field(
        default="",
        description="Range start (RFC3339 or unix ms). Set with end_time. Otherwise the newest 120 candles.",
    )
    end_time: str = Field(
        default="",
        description="Range end (RFC3339 or unix ms). Set with start_time.",
    )


class BasisInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all = three venues; a missing quote is unknown)",
    )


class CorrelationInput(BaseModel):
    symbol: str = Field(description="Pair e.g. SOLUSDT")
    exchange: str = Field(
        default="binance",
        description="binance|bybit|coinbase (default binance)",
    )


class VolatilityInput(BaseModel):
    symbol: str = Field(description="Pair e.g. SOLUSDT")
    exchange: str = Field(
        default="binance",
        description="binance|bybit|coinbase (default binance)",
    )


class SnapshotInput(BaseModel):
    symbol: str = Field(description="Pair e.g. SOLUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all)",
    )


class LevelsInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="binance",
        description="binance|bybit|coinbase (default binance)",
    )


class WhalesInput(BaseModel):
    symbol: str = Field(
        default="",
        description="Pair e.g. BTCUSDT. Empty scans top liquid USDT coins.",
    )
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all)",
    )
    min_notional: float = Field(
        default=0,
        description="Minimum USD size (default 100000)",
    )
    limit: int = Field(
        default=0,
        description="Max events (default 30, max 100)",
    )


class BreadthInput(BaseModel):
    exchange: str = Field(
        default="binance",
        description="binance|bybit|coinbase (default binance)",
    )
    limit: int = Field(
        default=80,
        description="How many liquid coins to include (default 80, max 150)",
    )


class FearGreedInput(BaseModel):
    """Market-wide index. No symbol."""


class FuturesHistoryInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    metric: str = Field(description="open_interest | funding | long_short | liquidations")
    exchange: str = Field(default="all", description="binance|bybit|okx|all")
    limit: int = Field(default=200, ge=1, le=1000)
    start: str = Field(default="", description="RFC3339 or unix ms")
    end: str = Field(default="", description="RFC3339 or unix ms")
    bucket: str = Field(
        default="",
        description="5m|30m|2h|4h. One sample per step across the window, capped at 90 days. Omit for the newest raw rows.",
    )


class HistorySpanInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")


class UnusualInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all, venues stay separate)",
    )


class UnusualScanInput(BaseModel):
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all, venues stay separate)",
    )
    metric: str = Field(
        default="all",
        description="all|funding|open_interest|liquidations|cvd|spread",
    )
    limit: int = Field(default=36, ge=5, le=36, description="Liquid names to score")


class LongShortInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|all (default all = both venues, not averaged)",
    )
    limit: int = Field(default=24, ge=1, le=100, description="5m history size")


class FundingRateInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    exchange: str = Field(
        default="all",
        description="binance|bybit|okx|all (default all = per-venue rows plus OI-weighted pressure)",
    )
    limit: int = Field(default=12, ge=1, le=30, description="Settled history size")


class FundingArbInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    notional: float = Field(
        default=0,
        description="Quote size on each leg (default 10000)",
    )
    hold_hours: float = Field(
        default=0,
        description="Hold window hours for the horizon payout (default 24)",
    )
    fee_binance_pct: float | None = Field(
        default=None,
        description="Binance taker fee percent (default paper 0.10)",
    )
    fee_bybit_pct: float | None = Field(
        default=None,
        description="Bybit taker fee percent (default paper 0.10)",
    )


class FundingArbScanInput(BaseModel):
    notional: float = Field(
        default=0,
        description="Quote size on each leg (default 10000)",
    )
    hold_hours: float = Field(
        default=0,
        description="Hold window hours for the horizon payout (default 24)",
    )
    fee_binance_pct: float | None = Field(
        default=None,
        description="Binance taker fee percent (default paper 0.10)",
    )
    fee_bybit_pct: float | None = Field(
        default=None,
        description="Bybit taker fee percent (default paper 0.10)",
    )
    quote: str = Field(default="USDT", description="Quote asset")
    limit: int = Field(default=0, description="How many top-volume coins (default 15, max 40)")


class FundingArbHistoryInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    start: str = Field(description="Range start RFC3339 or YYYY-MM-DD (UTC)")
    end: str = Field(description="Range end RFC3339 or YYYY-MM-DD (UTC)")
    notional: float = Field(default=0, description="Quote size on each leg (default 10000)")
    fee_binance_pct: float | None = Field(default=None)
    fee_bybit_pct: float | None = Field(default=None)


class FundingArbWatchCreateInput(BaseModel):
    client_id: str
    symbol: str = Field(
        default="",
        description="Pair e.g. BTCUSDT, or omit / scan / * to follow the full scan",
    )
    min_profit: float = Field(gt=0, description="Minimum after-fee profit in quote currency")
    notional: float = Field(default=0, description="Quote size on each leg (default 10000)")
    hold_hours: float = Field(default=0, description="Hold window hours (default 24)")
    quote: str = Field(default="", description="Scan quote asset (default USDT)")
    limit: int = Field(default=0, description="How many top-volume coins to scan")
    duration_hours: float = Field(
        default=0,
        description="How long the follow should run; 0 or omit to keep running",
    )
    fee_binance_pct: float | None = Field(default=None)
    fee_bybit_pct: float | None = Field(default=None)


class FundingArbWatchIdInput(BaseModel):
    client_id: str
    watch_id: str


class FundingArbWatchUpdateInput(BaseModel):
    client_id: str
    watch_id: str
    min_profit: float | None = Field(default=None, description="New after-fee profit floor")
    notional: float | None = None
    hold_hours: float | None = None
    quote: str | None = None
    limit: int | None = None
    duration_hours: float | None = Field(
        default=None,
        description="Reset how long the follow should run from now; 0 = no end",
    )
    fee_binance_pct: float | None = None
    fee_bybit_pct: float | None = None


class FundingArbSignalListInput(BaseModel):
    client_id: str
    status: str = Field(default="open", description="open | closed | all")
    limit: int = 50


class FundingArbSignalIdInput(BaseModel):
    client_id: str
    signal_id: str


class MarketLiquidityInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or BTC-USD")
    exchange: str = Field(
        default="all",
        description=f"{EXCHANGE_VENUES_OR_ALL} (default all = per-venue + market-wide)",
    )


class OrderBookHeatmapInput(BaseModel):
    symbol: str = Field(description="Pair e.g. BTCUSDT or BTC-USD")
    exchange: str = Field(default="binance", description=EXCHANGE_VENUES)
    group: str = Field(default="", description="Price bucket e.g. 0.1; empty = suggested default")
    window: int = Field(
        default=600,
        ge=60,
        le=604800,
        description="Lookback seconds (60–604800, default 600). Above 1800 uses stored 1-minute ladders, including brief walls and size moves.",
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


class BookGapsInput(BaseModel):
    symbol: str
    exchange: str = Field(default="all", description="binance|coinbase|bybit|all")
    range_pct: float = Field(default=0, description="±% of mid (default 2)")


class BookGapScanInput(BaseModel):
    exchange: str = Field(default="all", description="binance|coinbase|bybit|all")
    quote: str = Field(default="USDT", description="Quote asset")
    range_pct: float = Field(default=0, description="±% of mid (default 2)")
    limit: int = Field(default=0, description="How many liquid pairs (default 20, max 30)")


class WideSpreadInput(BaseModel):
    symbol: str
    exchange: str = Field(default="all", description="binance|coinbase|bybit|all")


class WideSpreadScanInput(BaseModel):
    exchange: str = Field(default="all", description="binance|coinbase|bybit|all")
    quote: str = Field(default="USDT", description="Quote label")
    limit: int = Field(default=0, description="How many coins (default 20, max 40)")


class WallScanInput(BaseModel):
    exchange: str = Field(
        default="all",
        description="binance|coinbase|bybit|all (default all)",
    )
    quote: str = Field(default="USDT", description="Quote asset")
    limit: int = Field(
        default=0,
        description="How many walls to return (default 40, max 80)",
    )


class MarketImpactScanInput(BaseModel):
    exchange: str = Field(
        default="all",
        description="binance|coinbase|bybit|all (default all)",
    )
    quote: str = Field(default="USDT", description="Quote asset")
    side: str = Field(default="buy", description="buy or sell")
    notional: float = Field(
        default=0,
        description="Shared quote size for every coin (default 100000)",
    )
    limit: int = Field(
        default=0,
        description="How many liquid pairs (default 20, max 30)",
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
        description="Comma-separated: watchlist,shares,alerts,backtests,portfolios,customSignals (empty=all)",
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


class LiquidationFeedAlertInput(BaseModel):
    client_id: str
    exchange: str = Field(default="all", description="binance | bybit | all")
    min_down_seconds: float = Field(
        default=300,
        ge=30,
        description="Fire after the feed stays down this many seconds (default 300)",
    )
    mode: str = Field(default="repeating", description="repeating (default) | one_time")


class LiquidationNotionalAlertInput(BaseModel):
    client_id: str
    symbol: str = Field(description="Pair e.g. BTCUSDT")
    notional: float = Field(gt=0, description="USDT threshold e.g. 20000000")
    side: str = Field(default="both", description="long | short | both")
    exchange: str = Field(default="all", description="binance | bybit | all")
    window: str = Field(default="5m", description="1m | 5m | 15m | 1h")
    mode: str = Field(default="repeating", description="repeating (default) | one_time")


class LiquidationCascadeAlertInput(BaseModel):
    client_id: str
    symbol: str = Field(description="Pair e.g. BTCUSDT, or all for a market scan")
    exchange: str = Field(default="all", description="binance | bybit | all")
    min_grade: str = Field(
        default="cascade",
        description="cascade (default) | extreme | elevated",
    )
    mode: str = Field(default="repeating", description="repeating (default) | one_time")


class SetupRadarAlertInput(BaseModel):
    client_id: str
    symbol: str = Field(
        default="all",
        description="Pair e.g. BTCUSDT, or all for any scanned coin",
    )
    exchange: str = Field(default="all", description="binance | bybit | all")
    min_similarity: float = Field(
        default=60,
        description="Min closeness 40–100 (0 = 60)",
    )
    lean: str = Field(
        default="any",
        description="any (default) | up | down — usual after-path",
    )
    mode: str = Field(default="repeating", description="repeating (default) | one_time")


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


class ClientIdInput(BaseModel):
    client_id: str


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
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class PortfolioTransferInput(BaseModel):
    client_id: str
    to_portfolio_id: str
    amount: float = Field(gt=0, description="Positive cash to move")
    from_portfolio_id: str = ""
    note: str = ""
    idempotency_key: str = Field(
        default="",
        description="Retry key; same key + same request returns the original transfer",
    )


class TransferAttemptsInput(BaseModel):
    client_id: str
    limit: int = 50
    offset: int = 0
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class PortfolioCashListInput(BaseModel):
    client_id: str
    limit: int = 50
    offset: int = 0
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class RiskLimitsSetInput(BaseModel):
    client_id: str
    max_daily_loss_pct: float = Field(
        default=0, description="e.g. 5 = stop new risk at 5% daily MTM loss; 0 disables"
    )
    max_asset_weight_pct: float = Field(
        default=0, description="e.g. 30 = max one coin % of equity; 0 disables"
    )
    max_exposure_pct: float = Field(
        default=0, description="e.g. 60 = max all coins together as % of equity; 0 disables"
    )
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


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
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class PortfolioPendingOrderInput(BaseModel):
    client_id: str
    symbol: str
    order_type: str = Field(description="limit_buy | limit_sell | stop_loss | trailing_stop")
    quantity: float = Field(gt=0)
    trigger_price: float = Field(
        default=0, description="Limit or stop price (omit for trailing_stop)"
    )
    exchange: str = "binance"
    time_in_force: str = Field(default="gtc", description="gtc | ioc | fok")
    expires_at: str = Field(default="", description="RFC3339 expiry for GTC only")
    lot_method: str = Field(default="", description="fifo or lifo for sell types")
    idempotency_key: str = Field(default="", description="Optional retry key")
    trail_type: str = Field(default="", description="trailing_stop: percent | offset")
    trail_value: float = Field(
        default=0, description="trailing_stop: fraction e.g. 0.05 or fixed offset"
    )
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class PortfolioLotsInput(BaseModel):
    client_id: str
    exchange: str = ""
    symbol: str = ""
    status: str = Field(default="open", description="open | closed | all")
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class PortfolioListOrdersInput(BaseModel):
    client_id: str
    status: str = Field(default="open", description="open|filled|canceled|rejected|all")
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class PortfolioCancelOrderInput(BaseModel):
    client_id: str
    order_id: str
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class PortfolioOCOInput(BaseModel):
    client_id: str
    symbol: str
    quantity: float = Field(gt=0)
    take_profit_price: float = Field(gt=0)
    stop_loss_price: float = Field(gt=0)
    exchange: str = Field(default="binance", description=EXCHANGE_VENUES)
    expires_at: str = ""
    idempotency_key: str = ""
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


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
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class PortfolioAmendInput(BaseModel):
    client_id: str
    order_id: str
    trigger_price: float = Field(default=0, description="New limit/stop; 0 = leave")
    remaining_quantity: float = Field(default=0, description="New remaining size; 0 = leave")


class PortfolioCancelAllInput(BaseModel):
    client_id: str
    symbol: str = ""
    exchange: str = Field(default="", description=EXCHANGE_VENUES)
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class PortfolioOrderGetInput(BaseModel):
    client_id: str
    order_id: str


class DelistInput(BaseModel):
    exchange: str = Field(default="binance", description=EXCHANGE_VENUES)


class PostDelistInput(BaseModel):
    symbol: str = Field(description="Pair e.g. VICUSDT")
    exchange: str = Field(default="binance", description=EXCHANGE_VENUES)
    interval: str = Field(default="1d", description="Candle interval hint; CoinGecko is daily")
    limit: int = Field(default=30, ge=1, le=200)


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
    time_zone: str = Field(default="", description="IANA timezone e.g. Europe/Istanbul")
    hour: int | None = Field(default=None, description="Local hour 0-23")
    minute: int = Field(default=0, description="Local minute 0-59")
    max_price: float = Field(
        default=0, description="Skip if last or fee+slippage unit exceeds this; 0 = no cap"
    )
    budget: float = Field(default=0, description="Total cash cap across succeeded runs; 0 = no cap")
    end_date: str = Field(default="", description="Last inclusive local day YYYY-MM-DD")
    ends_at: str = Field(default="", description="RFC3339 inclusive last scheduled instant")
    start_at: str = Field(default="", description="RFC3339 first run; default now")
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


class RecurringBuyUpdateInput(BaseModel):
    client_id: str
    plan_id: str
    name: str = ""
    amount: float = 0
    frequency: str = ""
    weekday: str = ""
    day_of_month: int = 0
    interval_hours: int = 0
    time_zone: str = ""
    hour: int | None = None
    minute: int = 0
    max_price: float | None = None
    budget: float | None = None
    end_date: str | None = None
    ends_at: str | None = None
    start_at: str = ""


class BasketCreateInput(BaseModel):
    client_id: str
    name: str
    targets_json: str = Field(
        description='JSON array e.g. [{"asset":"BTC","weightPct":50},{"asset":"ETH","weightPct":30},{"asset":"USDT","weightPct":20}]'
    )
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


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


class RecurringBuyCompleteInput(BaseModel):
    client_id: str
    plan_id: str
    run_id: str
    portfolio_id: str = ""


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
    portfolio_id: str = Field(default="", description="Book id or name when multiple exist")


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
    notional: float = Field(gt=0, description="Quote size to walk on live books e.g. 10000")
    min_profit: float = Field(
        ge=0, description="Minimum after-fee profit in quote currency e.g. 20"
    )
    min_duration_sec: float = Field(
        default=0,
        ge=0,
        description="Seconds the fill must stay qualifying before opening; 0 = first tick",
    )
    min_net_diff_pct: float = Field(default=0, ge=0, description="Optional extra profit % floor")
    fee_binance_pct: float = Field(default=0, ge=0, description="Binance fee %")
    fee_coinbase_pct: float = Field(default=0, ge=0, description="Coinbase fee %")
    fee_bybit_pct: float = Field(default=0, ge=0, description="Bybit fee %")
    exchanges: list[str] | None = Field(
        default=None,
        description="Venues to walk, e.g. [binance, bybit]. Default all three. At least two.",
    )


class PriceDiffWatchIdInput(BaseModel):
    client_id: str
    watch_id: str


class PriceDiffWatchUpdateInput(BaseModel):
    client_id: str
    watch_id: str
    notional: float | None = Field(default=None, description="New quote size to walk")
    min_profit: float | None = Field(default=None, description="New after-fee profit floor")
    min_duration_sec: float | None = Field(
        default=None,
        description="Seconds the fill must stay qualifying; 0 = first tick",
    )
    min_net_diff_pct: float | None = None
    fee_binance_pct: float | None = None
    fee_coinbase_pct: float | None = None
    fee_bybit_pct: float | None = None
    exchanges: list[str] | None = Field(
        default=None,
        description="Replace venues, e.g. [binance, bybit]. At least two.",
    )


class PriceDiffOppListInput(BaseModel):
    client_id: str
    status: str = Field(default="open", description="open | closed | all")
    limit: int = 50
    offset: int = 0


class PriceDiffOppIdInput(BaseModel):
    client_id: str
    opportunity_id: str


class PriceDiffQuoteInput(BaseModel):
    symbol: str
    buy_exchange: str
    sell_exchange: str
    notional: float | None = Field(
        default=None, description="USDT to spend on the buy book before the buy fee"
    )
    quantity: float | None = Field(
        default=None, description="Base size to buy and sell (instead of notional)"
    )
    fee_buy_pct: float = Field(default=0, ge=0, description="Buy venue taker fee %")
    fee_sell_pct: float = Field(default=0, ge=0, description="Sell venue taker fee %")
    min_net_diff_pct: float = Field(
        default=0, ge=0, description="Optional threshold for meetsMinNet"
    )


class PriceDiffOppQuoteInput(BaseModel):
    client_id: str
    opportunity_id: str
    notional: float | None = Field(
        default=None, description="USDT to spend on the buy book before the buy fee"
    )
    quantity: float | None = Field(
        default=None, description="Base size to buy and sell (instead of notional)"
    )


class PriceDiffScanInput(BaseModel):
    symbol: str
    notional: float | None = Field(
        default=None, description="USDT to spend on the buy book before the buy fee"
    )
    quantity: float | None = Field(
        default=None, description="Base size to buy and sell (instead of notional)"
    )
    fee_binance_pct: float = Field(default=0, ge=0, description="Binance taker fee %")
    fee_coinbase_pct: float = Field(default=0, ge=0, description="Coinbase taker fee %")
    fee_bybit_pct: float = Field(default=0, ge=0, description="Bybit taker fee %")
    min_net_diff_pct: float = Field(
        default=0, ge=0, description="Optional threshold for meetsMinNet"
    )
    min_profit_pct: float = Field(
        default=0, ge=0, description="Only list routes whose after-fee profit % is at least this"
    )
    min_profit_amount: float = Field(
        default=0,
        ge=0,
        description="Only list routes whose after-fee profit (USDT) is at least this",
    )


class PriceDiffWatchQuoteInput(BaseModel):
    client_id: str
    watch_id: str
    notional: float | None = Field(
        default=None, description="USDT to spend on the buy book before the buy fee"
    )
    quantity: float | None = Field(
        default=None, description="Base size to buy and sell (instead of notional)"
    )
    min_profit_pct: float = Field(
        default=0, ge=0, description="Only list routes whose after-fee profit % is at least this"
    )
    min_profit_amount: float = Field(
        default=0,
        ge=0,
        description="Only list routes whose after-fee profit (USDT) is at least this",
    )


class ScannerRuleCreateInput(BaseModel):
    client_id: str
    rule_type: str = Field(
        default="",
        description="rsi | ma_crossover | volume_increase | combo (optional if conditions set)",
    )
    conditions: str = Field(
        default="",
        description="Comma-separated rsi,ma_crossover,volume_increase",
    )
    match_mode: str = Field(
        default="all",
        description="all = every selected condition; any = one is enough",
    )
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


class ScannerRuleUpdateInput(BaseModel):
    client_id: str
    rule_id: str
    enabled: bool | None = Field(
        default=None, description="true to run the rule, false to pause it"
    )
    interval: str | None = None
    conditions: str | None = Field(
        default=None,
        description="Comma-separated rsi,ma_crossover,volume_increase",
    )
    match_mode: str | None = Field(default=None, description="all | any")
    rsi_period: int | None = None
    rsi_condition: str | None = None
    rsi_threshold: float | None = None
    ma_fast_period: int | None = None
    ma_slow_period: int | None = None
    ma_direction: str | None = None
    volume_lookback: int | None = None
    volume_min_ratio: float | None = None


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

    def get_post_delist(
        symbol: str,
        exchange: str = "binance",
        interval: str = "1d",
        limit: int = 30,
    ) -> str:
        return http.get(
            "/api/v1/market/post-delist",
            {
                "symbol": symbol,
                "exchange": exchange,
                "interval": interval,
                "limit": limit,
            },
        )

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

    def get_orderbook_icebergs(
        symbol: str,
        exchange: str = "all",
        min_notional: float = 0,
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if min_notional:
            params["minNotional"] = min_notional
        return http.get("/api/v1/market/orderbook/icebergs", params)

    def get_orderbook_liquidity_flow(
        symbol: str,
        exchange: str = "all",
        window: str = "5m",
        start_time: str = "",
        end_time: str = "",
        min_price: float = 0,
        max_price: float = 0,
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if window:
            params["window"] = window
        if start_time:
            params["startTime"] = start_time
        if end_time:
            params["endTime"] = end_time
        if min_price:
            params["minPrice"] = min_price
        if max_price:
            params["maxPrice"] = max_price
        return http.get("/api/v1/market/orderbook/liquidity-flow", params)

    def get_orderbook_history(
        symbol: str,
        exchange: str = "binance",
        at: str = "",
        from_time: str = "",
        to_time: str = "",
        limit: int = 0,
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if at:
            params["at"] = at
        if from_time:
            params["from"] = from_time
        if to_time:
            params["to"] = to_time
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/orderbook/history", params)

    def compare_orderbook_history(
        symbol: str,
        exchange: str = "binance",
        from_time: str = "",
        to_time: str = "",
    ) -> str:
        return http.get(
            "/api/v1/market/orderbook/history/compare",
            {"symbol": symbol, "exchange": exchange, "from": from_time, "to": to_time},
        )

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

    def get_liquidation_levels(
        symbol: str = "all", exchange: str = "all", range: str = "24h"
    ) -> str:
        return http.get(
            "/api/v1/market/liquidation-levels",
            {"symbol": symbol, "exchange": exchange, "range": range},
        )

    def get_liquidation_cascade(symbol: str = "all", exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/liquidation-cascade",
            {"symbol": symbol, "exchange": exchange},
        )

    def scan_liquidation_cascades(exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/liquidation-cascade/scan",
            {"exchange": exchange},
        )

    def get_liquidation_overview(
        exchange: str = "all", window: str = "24h", limit: int = 50
    ) -> str:
        return http.get(
            "/api/v1/market/liquidations/overview",
            {"exchange": exchange, "window": window, "limit": limit},
        )

    def get_options_overview(asset: str, exchange: str = "deribit") -> str:
        return http.get(
            "/api/v1/market/options",
            {"asset": asset, "exchange": exchange},
        )

    def get_open_interest(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/open-interest",
            {"symbol": symbol, "exchange": exchange},
        )

    def compare_spot_perp_liquidity(symbol: str, range_pct: float = 1) -> str:
        params: dict[str, Any] = {"symbol": symbol}
        if range_pct:
            params["rangePct"] = range_pct
        return http.get("/api/v1/market/orderbook/spot-perp", params)

    def get_perp_depth(symbol: str, exchange: str = "all", range_pct: float = 1) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if range_pct:
            params["rangePct"] = range_pct
        return http.get("/api/v1/market/perp-depth", params)

    def get_derivatives_board(exchange: str = "all", limit: int = 36) -> str:
        params: dict[str, Any] = {"exchange": exchange}
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/derivatives-board", params)

    def get_perp_book_history(
        symbol: str, exchange: str = "all", window_sec: int = 86400
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if window_sec:
            params["windowSec"] = window_sec
        return http.get("/api/v1/market/perp-depth/book", params)

    def get_perp_depth_overview(
        exchange: str = "all", range_pct: float = 1, limit: int = 20
    ) -> str:
        params: dict[str, Any] = {"exchange": exchange}
        if range_pct:
            params["rangePct"] = range_pct
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/perp-depth/overview", params)

    def estimate_liquidation_hunt(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/liquidation-hunt",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_liquidation_heatmap(symbol: str, range: str = "24h", exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/liquidation-hunt/heatmap",
            {"symbol": symbol, "range": range, "exchange": exchange},
        )

    def get_liquidation_print_heatmap(symbol: str, range: str = "24h", exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/liquidation-prints/heatmap",
            {"symbol": symbol, "range": range, "exchange": exchange},
        )

    def get_liquidation_open_levels(symbol: str, range: str = "24h", exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/liquidation-open-levels",
            {"symbol": symbol, "range": range, "exchange": exchange},
        )

    def get_squeeze_risk(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/squeeze-risk",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_positioning(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/positioning",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_venue_divergence(symbol: str) -> str:
        return http.get(
            "/api/v1/market/venue-divergence",
            {"symbol": symbol},
        )

    def get_taker_flow(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/taker-flow",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_net_position(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/net-position",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_volume_share(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/volume-share",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_participation(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/participation",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_print_size(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/print-size",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_print_size_overview(
        exchange: str = "all", quote: str = "USDT", limit: int = 0
    ) -> str:
        params: dict[str, Any] = {"exchange": exchange, "quote": quote}
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/print-size/overview", params)

    def get_bot_tracker(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/bot-tracker",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_order_count_ratio(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/order-count",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_order_count_overview(
        exchange: str = "all", quote: str = "USDT", limit: int = 0
    ) -> str:
        params: dict[str, Any] = {"exchange": exchange, "quote": quote}
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/order-count/overview", params)

    def get_market_pressure(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/pressure",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_cvd(
        symbol: str, exchange: str = "all", range: str = "24h", anchor: str = ""
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if range:
            params["range"] = range
        if anchor:
            params["anchor"] = anchor
        return http.get("/api/v1/market/cvd", params)

    def get_positioning_anchor(
        symbol: str,
        anchor: str,
        exchange: str = "all",
        bucket: str = "",
        start: str = "",
        end: str = "",
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "anchor": anchor, "exchange": exchange}
        if bucket:
            params["bucket"] = bucket
        if start:
            params["from"] = start
        if end:
            params["to"] = end
        return http.get("/api/v1/market/positioning-anchor", params)

    def get_tape_backtest(
        symbol: str,
        exchange: str = "all",
        condition: str = "cvd",
        side: str = "",
        threshold: float = 0,
        start: str = "",
        end: str = "",
    ) -> str:
        params: dict[str, Any] = {
            "symbol": symbol,
            "exchange": exchange,
            "condition": condition,
        }
        if side:
            params["side"] = side
        if threshold:
            params["threshold"] = threshold
        if start:
            params["from"] = start
        if end:
            params["to"] = end
        return http.get("/api/v1/market/tape-backtest", params)

    def get_absorption(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/absorption",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_sweep_recovery(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/orderbook/recovery",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_liquidity_sweeps(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/liquidity-sweeps",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_volume_surge(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/volume-surge",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_rsi_heatmap(
        exchange: str = "binance",
        quote: str = "USDT",
        interval: str = "1h",
        limit: int = 0,
        period: int = 0,
        sort: str = "",
    ) -> str:
        params: dict[str, Any] = {"exchange": exchange, "quote": quote}
        if interval:
            params["interval"] = interval
        if limit:
            params["limit"] = limit
        if period:
            params["period"] = period
        if sort:
            params["sort"] = sort
        return http.get("/api/v1/market/rsi-heatmap", params)

    def scan_volume_surges(
        exchange: str = "binance",
        quote: str = "USDT",
        min_ratio: float = 0,
        limit: int = 0,
    ) -> str:
        params: dict[str, Any] = {"exchange": exchange, "quote": quote}
        if min_ratio:
            params["minRatio"] = min_ratio
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/volume-surge/scan", params)

    def get_around(
        symbol: str,
        at: str,
        exchange: str = "all",
        window: str = "1h",
        during: str = "15m",
    ) -> str:
        params: dict[str, Any] = {
            "symbol": symbol,
            "at": at,
            "exchange": exchange,
        }
        if window:
            params["window"] = window
        if during:
            params["during"] = during
        return http.get("/api/v1/market/around", params)

    def compare_around(
        symbol: str,
        from_time: str,
        to_time: str,
        exchange: str = "all",
        window: str = "1h",
        during: str = "15m",
    ) -> str:
        params: dict[str, Any] = {
            "symbol": symbol,
            "from": from_time,
            "to": to_time,
            "exchange": exchange,
        }
        if window:
            params["window"] = window
        if during:
            params["during"] = during
        return http.get("/api/v1/market/around/compare", params)

    def find_around_moves(
        symbol: str,
        exchange: str = "all",
        lookback: str = "24h",
        interval: str = "15m",
        direction: str = "both",
        min_return_pct: float = 0,
        limit: int = 0,
        window: str = "1h",
        during: str = "",
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if lookback:
            params["lookback"] = lookback
        if interval:
            params["interval"] = interval
        if direction:
            params["direction"] = direction
        if min_return_pct:
            params["minReturnPct"] = min_return_pct
        if limit:
            params["limit"] = limit
        if window:
            params["window"] = window
        if during:
            params["during"] = during
        return http.get("/api/v1/market/around/moves", params)

    def find_around_precursors(
        symbol: str,
        exchange: str = "all",
        lookback: str = "7d",
        interval: str = "15m",
        direction: str = "both",
        min_return_pct: float = 0,
        limit: int = 0,
        window: str = "1h",
        during: str = "",
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if lookback:
            params["lookback"] = lookback
        if interval:
            params["interval"] = interval
        if direction:
            params["direction"] = direction
        if min_return_pct:
            params["minReturnPct"] = min_return_pct
        if limit:
            params["limit"] = limit
        if window:
            params["window"] = window
        if during:
            params["during"] = during
        return http.get("/api/v1/market/around/precursors", params)

    def find_around_similar(
        symbol: str,
        exchange: str = "all",
        lookback: str = "7d",
        interval: str = "15m",
        direction: str = "both",
        min_return_pct: float = 0,
        limit: int = 0,
        window: str = "1h",
        during: str = "",
        fields: str = "",
        weights: str = "",
        min_coverage: float = -1,
        horizons: str = "",
        at: str = "",
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if lookback:
            params["lookback"] = lookback
        if interval:
            params["interval"] = interval
        if direction:
            params["direction"] = direction
        if min_return_pct:
            params["minReturnPct"] = min_return_pct
        if limit:
            params["limit"] = limit
        if window:
            params["window"] = window
        if during:
            params["during"] = during
        if fields:
            params["fields"] = fields
        if weights:
            params["weights"] = weights
        if min_coverage >= 0:
            params["minCoverage"] = min_coverage
        if horizons:
            params["horizons"] = horizons
        if at:
            params["at"] = at
        return http.get("/api/v1/market/around/similar", params)

    def get_market_forensics(
        symbol: str,
        at: str,
        exchange: str = "all",
        window: str = "1h",
        during: str = "15m",
        lookback: str = "7d",
        interval: str = "15m",
        direction: str = "both",
        min_return_pct: float = 0,
        limit: int = 0,
        fields: str = "",
        weights: str = "",
        min_coverage: float = -1,
        horizons: str = "",
        as_of: str = "",
    ) -> str:
        params: dict[str, Any] = {
            "symbol": symbol,
            "at": at,
            "exchange": exchange,
        }
        if window:
            params["window"] = window
        if during:
            params["during"] = during
        if lookback:
            params["lookback"] = lookback
        if interval:
            params["interval"] = interval
        if direction:
            params["direction"] = direction
        if min_return_pct:
            params["minReturnPct"] = min_return_pct
        if limit:
            params["limit"] = limit
        if fields:
            params["fields"] = fields
        if weights:
            params["weights"] = weights
        if min_coverage >= 0:
            params["minCoverage"] = min_coverage
        if horizons:
            params["horizons"] = horizons
        if as_of:
            params["asOf"] = as_of
        return http.get("/api/v1/market/forensics", params)

    def scan_market_forensics(
        exchange: str = "all",
        quote: str = "USDT",
        lookback: str = "3d",
        interval: str = "15m",
        direction: str = "both",
        min_return_pct: float = 0,
        limit: int = 0,
        window: str = "1h",
        during: str = "",
        fields: str = "",
        weights: str = "",
        min_coverage: float = -1,
        horizons: str = "",
    ) -> str:
        params: dict[str, Any] = {"exchange": exchange}
        if quote:
            params["quote"] = quote
        if lookback:
            params["lookback"] = lookback
        if interval:
            params["interval"] = interval
        if direction:
            params["direction"] = direction
        if min_return_pct:
            params["minReturnPct"] = min_return_pct
        if limit:
            params["limit"] = limit
        if window:
            params["window"] = window
        if during:
            params["during"] = during
        if fields:
            params["fields"] = fields
        if weights:
            params["weights"] = weights
        if min_coverage >= 0:
            params["minCoverage"] = min_coverage
        if horizons:
            params["horizons"] = horizons
        return http.get("/api/v1/market/forensics/scan", params)

    def review_setup_radar(
        exchange: str = "all",
        symbol: str = "",
        since: str = "7d",
        limit: int = 50,
    ) -> str:
        params: dict[str, Any] = {"exchange": exchange, "since": since}
        if symbol:
            params["symbol"] = symbol
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/forensics/radar", params)

    def get_vwap(
        symbol: str,
        exchange: str = "all",
        window: str = "24h",
        start_time: str = "",
        end_time: str = "",
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if window:
            params["window"] = window
        if start_time:
            params["startTime"] = start_time
        if end_time:
            params["endTime"] = end_time
        return http.get("/api/v1/market/vwap", params)

    def get_volume_profile(
        symbol: str,
        exchange: str = "all",
        window: str = "24h",
        start_time: str = "",
        end_time: str = "",
        tick_size: float = 0,
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if window:
            params["window"] = window
        if start_time:
            params["startTime"] = start_time
        if end_time:
            params["endTime"] = end_time
        if tick_size:
            params["tickSize"] = tick_size
        return http.get("/api/v1/market/volume-profile", params)

    def get_flow_at_prices(
        symbol: str,
        exchange: str = "all",
        window: str = "15m",
        start_time: str = "",
        end_time: str = "",
        tick_size: float = 0,
        min_price: float = 0,
        max_price: float = 0,
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if window:
            params["window"] = window
        if start_time:
            params["startTime"] = start_time
        if end_time:
            params["endTime"] = end_time
        if tick_size:
            params["tickSize"] = tick_size
        if min_price:
            params["minPrice"] = min_price
        if max_price:
            params["maxPrice"] = max_price
        return http.get("/api/v1/market/flow-at-prices", params)

    def get_candle_footprint(
        symbol: str,
        exchange: str = "all",
        interval: str = "1h",
        start_time: str = "",
        end_time: str = "",
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "interval": interval, "exchange": exchange}
        if start_time:
            params["startTime"] = start_time
        if end_time:
            params["endTime"] = end_time
        return http.get("/api/v1/market/flow-at-prices/candles", params)

    def get_basis(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/basis",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_price_correlation(symbol: str, exchange: str = "binance") -> str:
        return http.get(
            "/api/v1/market/correlation",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_price_volatility(symbol: str, exchange: str = "binance") -> str:
        return http.get(
            "/api/v1/market/volatility",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_market_snapshot(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/snapshot",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_support_resistance(symbol: str, exchange: str = "binance") -> str:
        return http.get(
            "/api/v1/market/levels",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_whale_trades(
        symbol: str = "",
        exchange: str = "all",
        min_notional: float = 0,
        limit: int = 0,
    ) -> str:
        params: dict[str, Any] = {"exchange": exchange}
        if symbol:
            params["symbol"] = symbol
        if min_notional:
            params["minNotional"] = min_notional
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/whales", params)

    def get_market_breadth(exchange: str = "binance", limit: int = 80) -> str:
        return http.get(
            "/api/v1/market/breadth",
            {"exchange": exchange, "limit": limit},
        )

    def get_fear_greed() -> str:
        return http.get("/api/v1/market/fear-greed", {})

    def get_futures_history(
        symbol: str,
        metric: str,
        exchange: str = "all",
        limit: int = 200,
        start: str = "",
        end: str = "",
        bucket: str = "",
    ) -> str:
        params: dict[str, Any] = {
            "symbol": symbol,
            "metric": metric,
            "exchange": exchange,
            "limit": limit,
        }
        if start:
            params["from"] = start
        if end:
            params["to"] = end
        if bucket:
            params["bucket"] = bucket
        return http.get("/api/v1/market/futures-history", params)

    def get_history_span(symbol: str) -> str:
        return http.get("/api/v1/market/history-span", {"symbol": symbol})

    def get_unusual_context(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/unusual",
            {"symbol": symbol, "exchange": exchange},
        )

    def scan_unusual(exchange: str = "all", metric: str = "all", limit: int = 36) -> str:
        return http.get(
            "/api/v1/market/unusual/scan",
            {"exchange": exchange, "metric": metric, "limit": limit},
        )

    def get_long_short_ratio(symbol: str, exchange: str = "all", limit: int = 24) -> str:
        return http.get(
            "/api/v1/market/long-short-ratio",
            {"symbol": symbol, "exchange": exchange, "limit": limit},
        )

    def get_funding_rate(symbol: str, exchange: str = "all", limit: int = 12) -> str:
        return http.get(
            "/api/v1/market/funding-rate",
            {"symbol": symbol, "exchange": exchange, "limit": limit},
        )

    def get_funding_arb(
        symbol: str,
        notional: float = 0,
        hold_hours: float = 0,
        fee_binance_pct: float | None = None,
        fee_bybit_pct: float | None = None,
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol}
        if notional:
            params["notional"] = notional
        if hold_hours:
            params["holdHours"] = hold_hours
        if fee_binance_pct is not None:
            params["feeBinancePct"] = fee_binance_pct
        if fee_bybit_pct is not None:
            params["feeBybitPct"] = fee_bybit_pct
        return http.get("/api/v1/market/funding-arb", params)

    def scan_funding_arb(
        notional: float = 0,
        hold_hours: float = 0,
        fee_binance_pct: float | None = None,
        fee_bybit_pct: float | None = None,
        quote: str = "USDT",
        limit: int = 0,
    ) -> str:
        params: dict[str, Any] = {"quote": quote}
        if notional:
            params["notional"] = notional
        if hold_hours:
            params["holdHours"] = hold_hours
        if fee_binance_pct is not None:
            params["feeBinancePct"] = fee_binance_pct
        if fee_bybit_pct is not None:
            params["feeBybitPct"] = fee_bybit_pct
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/funding-arb/scan", params)

    def get_funding_arb_history(
        symbol: str,
        start: str,
        end: str,
        notional: float = 0,
        fee_binance_pct: float | None = None,
        fee_bybit_pct: float | None = None,
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "from": start, "to": end}
        if notional:
            params["notional"] = notional
        if fee_binance_pct is not None:
            params["feeBinancePct"] = fee_binance_pct
        if fee_bybit_pct is not None:
            params["feeBybitPct"] = fee_bybit_pct
        return http.get("/api/v1/market/funding-arb/history", params)

    def create_funding_arb_watch(
        client_id: str,
        min_profit: float,
        symbol: str = "",
        notional: float = 0,
        hold_hours: float = 0,
        quote: str = "",
        limit: int = 0,
        duration_hours: float = 0,
        fee_binance_pct: float | None = None,
        fee_bybit_pct: float | None = None,
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "minProfit": min_profit,
        }
        if symbol:
            body["symbol"] = symbol
        if notional:
            body["notional"] = notional
        if hold_hours:
            body["holdHours"] = hold_hours
        if quote:
            body["quote"] = quote
        if limit:
            body["limit"] = limit
        if duration_hours:
            body["durationHours"] = duration_hours
        if fee_binance_pct is not None:
            body["feeBinancePct"] = fee_binance_pct
        if fee_bybit_pct is not None:
            body["feeBybitPct"] = fee_bybit_pct
        return http.post("/api/v1/funding-arb/watches", body)

    def list_funding_arb_watches(client_id: str) -> str:
        return http.get("/api/v1/funding-arb/watches", {"clientId": client_id})

    def get_funding_arb_watch(client_id: str, watch_id: str) -> str:
        return http.get(
            f"/api/v1/funding-arb/watches/{watch_id}",
            {"clientId": client_id},
        )

    def update_funding_arb_watch(
        client_id: str,
        watch_id: str,
        min_profit: float | None = None,
        notional: float | None = None,
        hold_hours: float | None = None,
        quote: str | None = None,
        limit: int | None = None,
        duration_hours: float | None = None,
        fee_binance_pct: float | None = None,
        fee_bybit_pct: float | None = None,
    ) -> str:
        body: dict[str, Any] = {"clientId": client_id}
        if min_profit is not None:
            body["minProfit"] = min_profit
        if notional is not None:
            body["notional"] = notional
        if hold_hours is not None:
            body["holdHours"] = hold_hours
        if quote:
            body["quote"] = quote
        if limit is not None:
            body["limit"] = limit
        if duration_hours is not None:
            body["durationHours"] = duration_hours
        if fee_binance_pct is not None:
            body["feeBinancePct"] = fee_binance_pct
        if fee_bybit_pct is not None:
            body["feeBybitPct"] = fee_bybit_pct
        return http.patch(f"/api/v1/funding-arb/watches/{watch_id}", body)

    def pause_funding_arb_watch(client_id: str, watch_id: str) -> str:
        return http.post(
            f"/api/v1/funding-arb/watches/{watch_id}/pause?clientId={client_id}",
            {},
        )

    def resume_funding_arb_watch(client_id: str, watch_id: str) -> str:
        return http.post(
            f"/api/v1/funding-arb/watches/{watch_id}/resume?clientId={client_id}",
            {},
        )

    def delete_funding_arb_watch(client_id: str, watch_id: str) -> str:
        return http.delete(
            f"/api/v1/funding-arb/watches/{watch_id}",
            {"clientId": client_id},
        )

    def list_funding_arb_signals(client_id: str, status: str = "open", limit: int = 50) -> str:
        params: dict[str, Any] = {"clientId": client_id, "status": status}
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/funding-arb/signals", params)

    def get_funding_arb_signal(client_id: str, signal_id: str) -> str:
        return http.get(
            f"/api/v1/funding-arb/signals/{signal_id}",
            {"clientId": client_id},
        )

    def get_market_liquidity(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/orderbook/liquidity",
            {"symbol": symbol, "exchange": exchange},
        )

    def get_orderbook_heatmap(
        symbol: str,
        exchange: str = "binance",
        group: str = "",
        window: int = 600,
    ) -> str:
        params: dict[str, Any] = {
            "symbol": symbol,
            "exchange": exchange,
            "window": window,
        }
        if group:
            params["group"] = group
        return http.get("/api/v1/market/orderbook/heatmap", params)

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

    def scan_market_impact(
        exchange: str = "all",
        quote: str = "USDT",
        side: str = "buy",
        notional: float = 0,
        limit: int = 0,
    ) -> str:
        params: dict[str, Any] = {"exchange": exchange, "quote": quote, "side": side}
        if notional and notional > 0:
            params["notional"] = notional
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/orderbook/impact/scan", params)

    def scan_suspicious_walls(
        exchange: str = "all",
        quote: str = "USDT",
        limit: int = 0,
    ) -> str:
        params: dict[str, Any] = {"exchange": exchange, "quote": quote}
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/orderbook/walls/scan", params)

    def get_book_gaps(
        symbol: str,
        exchange: str = "all",
        range_pct: float = 0,
    ) -> str:
        params: dict[str, Any] = {"symbol": symbol, "exchange": exchange}
        if range_pct and range_pct > 0:
            params["rangePct"] = range_pct
        return http.get("/api/v1/market/orderbook/gaps", params)

    def scan_book_gaps(
        exchange: str = "all",
        quote: str = "USDT",
        range_pct: float = 0,
        limit: int = 0,
    ) -> str:
        params: dict[str, Any] = {"exchange": exchange, "quote": quote}
        if range_pct and range_pct > 0:
            params["rangePct"] = range_pct
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/orderbook/gaps/scan", params)

    def get_wide_spread(symbol: str, exchange: str = "all") -> str:
        return http.get(
            "/api/v1/market/orderbook/spread",
            {"symbol": symbol, "exchange": exchange},
        )

    def scan_wide_spreads(
        exchange: str = "all", quote: str = "USDT", limit: int = 0
    ) -> str:
        params: dict[str, Any] = {"exchange": exchange, "quote": quote}
        if limit:
            params["limit"] = limit
        return http.get("/api/v1/market/orderbook/spreads/scan", params)

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

    def get_holders(asset: str) -> str:
        return http.get("/api/v1/market/holders", {"asset": asset})

    def get_asset_profile(asset: str) -> str:
        return http.get("/api/v1/market/asset-profile", {"asset": asset})

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

    def list_custom_signal_factors() -> str:
        return http.get("/api/v1/swing/custom/factors", {})

    def list_custom_signal_profiles(client_id: str) -> str:
        return http.get("/api/v1/swing/custom/profiles", {"clientId": client_id})

    def create_custom_signal_profile(
        client_id: str, name: str, weights_json: str, is_default: bool = False
    ) -> str:
        weights = json.loads(weights_json) if weights_json else {}
        return http.post(
            "/api/v1/swing/custom/profiles",
            {"clientId": client_id, "name": name, "weights": weights, "isDefault": is_default},
        )

    def update_custom_signal_profile(
        client_id: str,
        profile_id: str,
        name: str = "",
        weights_json: str = "",
        is_default: bool | None = None,
    ) -> str:
        body: dict[str, Any] = {"clientId": client_id}
        if name:
            body["name"] = name
        if weights_json:
            body["weights"] = json.loads(weights_json)
        if is_default is not None:
            body["isDefault"] = is_default
        return http.patch(f"/api/v1/swing/custom/profiles/{profile_id}", body)

    def duplicate_custom_signal_profile(client_id: str, profile_id: str, name: str = "") -> str:
        body: dict[str, Any] = {"clientId": client_id}
        if name:
            body["name"] = name
        return http.post(f"/api/v1/swing/custom/profiles/{profile_id}/duplicate", body)

    def set_default_custom_signal_profile(client_id: str, profile_id: str) -> str:
        return http.post(
            f"/api/v1/swing/custom/profiles/{profile_id}/default",
            {"clientId": client_id},
        )

    def delete_custom_signal_profile(client_id: str, profile_id: str) -> str:
        return http.delete(
            f"/api/v1/swing/custom/profiles/{profile_id}",
            {"clientId": client_id},
        )

    def scan_custom_signals(
        client_id: str,
        profile_id: str = "",
        exchange: str = "",
        limit: int = 25,
        weight_ema_crossover: float | None = None,
        weight_ema_trend: float | None = None,
        weight_supertrend: float | None = None,
        weight_rsi_recovery: float | None = None,
        weight_macd: float | None = None,
        weight_adx: float | None = None,
        weight_accumulation: float | None = None,
        weight_volume_breakout: float | None = None,
        weight_hunt_up: float | None = None,
    ) -> str:
        params: dict[str, Any] = {"clientId": client_id, "limit": limit}
        if profile_id:
            params["profileId"] = profile_id
        if exchange:
            params["exchange"] = exchange
        weights = {
            "weightEmaCrossover": weight_ema_crossover,
            "weightEmaTrend": weight_ema_trend,
            "weightSupertrend": weight_supertrend,
            "weightRsiRecovery": weight_rsi_recovery,
            "weightMacd": weight_macd,
            "weightAdx": weight_adx,
            "weightAccumulation": weight_accumulation,
            "weightVolumeBreakout": weight_volume_breakout,
            "weightHuntUp": weight_hunt_up,
        }
        params.update({key: val for key, val in weights.items() if val is not None})
        return http.get("/api/v1/swing/custom/scan", params)

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

    def create_liquidation_feed_alert(
        client_id: str,
        exchange: str = "all",
        min_down_seconds: float = 300.0,
        mode: str = "repeating",
    ) -> str:
        return http.post(
            "/api/v1/alerts",
            {
                "clientId": client_id,
                "kind": "liquidation_feed",
                "exchange": exchange,
                "targetPrice": min_down_seconds,
                "mode": mode,
            },
        )

    def create_liquidation_cascade_alert(
        client_id: str,
        symbol: str,
        exchange: str = "all",
        min_grade: str = "cascade",
        mode: str = "repeating",
    ) -> str:
        return http.post(
            "/api/v1/alerts",
            {
                "clientId": client_id,
                "kind": "liquidation_cascade",
                "exchange": exchange,
                "symbol": symbol,
                "condition": min_grade,
                "mode": mode,
            },
        )

    def create_liquidation_notional_alert(
        client_id: str,
        symbol: str,
        notional: float,
        side: str = "both",
        exchange: str = "all",
        window: str = "5m",
        mode: str = "repeating",
    ) -> str:
        return http.post(
            "/api/v1/alerts",
            {
                "clientId": client_id,
                "kind": "liquidation_notional",
                "exchange": exchange,
                "symbol": symbol,
                "condition": side,
                "targetPrice": notional,
                "window": window,
                "mode": mode,
            },
        )

    def create_setup_radar_alert(
        client_id: str,
        symbol: str = "all",
        exchange: str = "all",
        min_similarity: float = 60.0,
        lean: str = "any",
        mode: str = "repeating",
    ) -> str:
        return http.post(
            "/api/v1/alerts",
            {
                "clientId": client_id,
                "kind": "setup_radar",
                "exchange": exchange,
                "symbol": symbol,
                "condition": lean,
                "targetPrice": min_similarity,
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

    def get_portfolio_risk_limits(client_id: str, portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/risk-limits", q)

    def get_paper_trading_costs(exchange: str = "") -> str:
        q: dict[str, Any] = {}
        if exchange:
            q["exchange"] = exchange
        return http.get("/api/v1/portfolio/trading-costs", q)

    def set_portfolio_risk_limits(
        client_id: str,
        max_daily_loss_pct: float = 0,
        max_asset_weight_pct: float = 0,
        max_exposure_pct: float = 0,
        portfolio_id: str = "",
    ) -> str:
        body: dict[str, Any] = {}
        if max_daily_loss_pct:
            body["maxDailyLossPct"] = max_daily_loss_pct
        if max_asset_weight_pct:
            body["maxAssetWeightPct"] = max_asset_weight_pct
        if max_exposure_pct:
            body["maxExposurePct"] = max_exposure_pct
        if portfolio_id:
            body["portfolioId"] = portfolio_id
        q = f"/api/v1/portfolio/risk-limits?clientId={client_id}"
        if portfolio_id:
            q += f"&portfolioId={portfolio_id}"
        return http.put(q, body)

    def clear_portfolio_risk_limits(client_id: str, portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.delete("/api/v1/portfolio/risk-limits", q)

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

    def deposit_portfolio_cash(
        client_id: str, amount: float, note: str = "", portfolio_id: str = ""
    ) -> str:
        body: dict[str, Any] = {"clientId": client_id, "amount": amount}
        if note:
            body["note"] = note
        if portfolio_id:
            body["portfolioId"] = portfolio_id
        return http.post("/api/v1/portfolio/deposits", body)

    def transfer_portfolio_cash(
        client_id: str,
        to_portfolio_id: str,
        amount: float,
        from_portfolio_id: str = "",
        note: str = "",
        idempotency_key: str = "",
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
        if idempotency_key:
            body["idempotencyKey"] = idempotency_key
        return http.post("/api/v1/portfolio/transfers", body)

    def list_transfer_attempts(
        client_id: str, limit: int = 50, offset: int = 0, portfolio_id: str = ""
    ) -> str:
        q: dict[str, Any] = {
            "clientId": client_id,
            "limit": str(limit),
            "offset": str(offset),
        }
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/transfers/attempts", q)

    def withdraw_portfolio_cash(
        client_id: str, amount: float, note: str = "", portfolio_id: str = ""
    ) -> str:
        body: dict[str, Any] = {"clientId": client_id, "amount": amount}
        if note:
            body["note"] = note
        if portfolio_id:
            body["portfolioId"] = portfolio_id
        return http.post("/api/v1/portfolio/withdrawals", body)

    def list_portfolio_cash_movements(
        client_id: str, limit: int = 50, offset: int = 0, portfolio_id: str = ""
    ) -> str:
        q: dict[str, Any] = {"clientId": client_id, "limit": limit, "offset": offset}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/cash-movements", q)

    def place_portfolio_order(
        client_id: str,
        symbol: str,
        side: str,
        quantity: float,
        exchange: str = "binance",
        lot_method: str = "",
        idempotency_key: str = "",
        portfolio_id: str = "",
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
        if portfolio_id:
            body["portfolioId"] = portfolio_id
        return http.post("/api/v1/portfolio/orders", body)

    def place_portfolio_pending_order(
        client_id: str,
        symbol: str,
        order_type: str,
        quantity: float,
        trigger_price: float = 0,
        exchange: str = "binance",
        time_in_force: str = "gtc",
        expires_at: str = "",
        lot_method: str = "",
        idempotency_key: str = "",
        trail_type: str = "",
        trail_value: float = 0,
        portfolio_id: str = "",
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "symbol": symbol,
            "type": order_type,
            "quantity": quantity,
            "exchange": exchange,
            "timeInForce": time_in_force,
        }
        if trigger_price:
            body["triggerPrice"] = trigger_price
        if expires_at:
            body["expiresAt"] = expires_at
        if lot_method:
            body["lotMethod"] = lot_method
        if idempotency_key:
            body["idempotencyKey"] = idempotency_key
        if trail_type:
            body["trailType"] = trail_type
        if trail_value:
            body["trailValue"] = trail_value
        if portfolio_id:
            body["portfolioId"] = portfolio_id
        return http.post("/api/v1/portfolio/orders", body)

    def list_portfolio_lots(
        client_id: str,
        exchange: str = "",
        symbol: str = "",
        status: str = "open",
        portfolio_id: str = "",
    ) -> str:
        params: dict[str, Any] = {"clientId": client_id, "status": status}
        if exchange:
            params["exchange"] = exchange
        if symbol:
            params["symbol"] = symbol
        if portfolio_id:
            params["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/lots", params)

    def list_portfolio_orders(client_id: str, status: str = "open", portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id, "status": status}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/orders", q)

    def cancel_portfolio_order(client_id: str, order_id: str, portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.delete(f"/api/v1/portfolio/orders/{order_id}", q)

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
        portfolio_id: str = "",
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
        if portfolio_id:
            body["portfolioId"] = portfolio_id
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
        portfolio_id: str = "",
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
        if portfolio_id:
            body["portfolioId"] = portfolio_id
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

    def cancel_all_portfolio_orders(
        client_id: str, symbol: str = "", exchange: str = "", portfolio_id: str = ""
    ) -> str:
        body: dict[str, Any] = {"clientId": client_id}
        if symbol:
            body["symbol"] = symbol
        if exchange:
            body["exchange"] = exchange
        if portfolio_id:
            body["portfolioId"] = portfolio_id
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

    def list_portfolio_trades(client_id: str, portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/trades", q)

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
        time_zone: str = "",
        hour: int | None = None,
        minute: int = 0,
        max_price: float = 0,
        budget: float = 0,
        end_date: str = "",
        ends_at: str = "",
        start_at: str = "",
        portfolio_id: str = "",
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
        if time_zone:
            body["timeZone"] = time_zone
        if hour is not None:
            body["hour"] = hour
            body["minute"] = minute
        if max_price:
            body["maxPrice"] = max_price
        if budget:
            body["budget"] = budget
        if end_date:
            body["endDate"] = end_date
        if ends_at:
            body["endsAt"] = ends_at
        if start_at:
            body["startAt"] = start_at
        if portfolio_id:
            body["portfolioId"] = portfolio_id
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
        time_zone: str = "",
        hour: int | None = None,
        minute: int = 0,
        max_price: float | None = None,
        budget: float | None = None,
        end_date: str | None = None,
        ends_at: str | None = None,
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
        if time_zone:
            body["timeZone"] = time_zone
        if hour is not None:
            body["hour"] = hour
            body["minute"] = minute
        if max_price is not None:
            body["maxPrice"] = max_price
        if budget is not None:
            body["budget"] = budget
        if end_date is not None:
            body["endDate"] = end_date
        if ends_at is not None:
            body["endsAt"] = ends_at
        if start_at:
            body["startAt"] = start_at
        return http.patch(
            f"/api/v1/portfolio/recurring-buys/{plan_id}?clientId={client_id}",
            body,
        )

    def list_recurring_buys(client_id: str, portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/recurring-buys", q)

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

    def complete_recurring_buy_run(
        client_id: str, plan_id: str, run_id: str, portfolio_id: str = ""
    ) -> str:
        q: dict[str, Any] = {"clientId": client_id}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.post(
            f"/api/v1/portfolio/recurring-buys/{plan_id}/runs/{run_id}/complete",
            q,
        )

    def create_portfolio_basket(
        client_id: str, name: str, targets_json: str, portfolio_id: str = ""
    ) -> str:
        targets = json.loads(targets_json)
        body: dict[str, Any] = {"clientId": client_id, "name": name, "targets": targets}
        if portfolio_id:
            body["portfolioId"] = portfolio_id
        return http.post("/api/v1/portfolio/baskets", body)

    def list_portfolio_baskets(client_id: str, portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/baskets", q)

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
        portfolio_id: str = "",
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
        if portfolio_id:
            body["portfolioId"] = portfolio_id
        return http.post("/api/v1/portfolio/margin/orders", body)

    def list_margin_positions(client_id: str, portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/margin/positions", q)

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

    def list_margin_orders(client_id: str, status: str = "open", portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id, "status": status}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/margin/orders", q)

    def cancel_margin_order(client_id: str, order_id: str, portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.delete(f"/api/v1/portfolio/margin/orders/{order_id}", q)

    def list_margin_trades(client_id: str, portfolio_id: str = "") -> str:
        q: dict[str, Any] = {"clientId": client_id}
        if portfolio_id:
            q["portfolioId"] = portfolio_id
        return http.get("/api/v1/portfolio/margin/trades", q)

    def create_price_diff_watch(
        client_id: str,
        symbol: str,
        notional: float,
        min_profit: float,
        min_duration_sec: float = 0,
        min_net_diff_pct: float = 0,
        fee_binance_pct: float = 0,
        fee_coinbase_pct: float = 0,
        fee_bybit_pct: float = 0,
        exchanges: list[str] | None = None,
    ) -> str:
        body: dict[str, Any] = {
            "clientId": client_id,
            "symbol": symbol,
            "notional": notional,
            "minProfit": min_profit,
            "minDurationSec": min_duration_sec,
            "minNetDiffPct": min_net_diff_pct,
            "feeBinancePct": fee_binance_pct,
            "feeCoinbasePct": fee_coinbase_pct,
            "feeBybitPct": fee_bybit_pct,
        }
        if exchanges:
            body["exchanges"] = exchanges
        return http.post("/api/v1/price-diff/watches", body)

    def list_price_diff_watches(client_id: str) -> str:
        return http.get("/api/v1/price-diff/watches", {"clientId": client_id})

    def get_price_diff_watch(client_id: str, watch_id: str) -> str:
        return http.get(
            f"/api/v1/price-diff/watches/{watch_id}",
            {"clientId": client_id},
        )

    def update_price_diff_watch(
        client_id: str,
        watch_id: str,
        notional: float | None = None,
        min_profit: float | None = None,
        min_duration_sec: float | None = None,
        min_net_diff_pct: float | None = None,
        fee_binance_pct: float | None = None,
        fee_coinbase_pct: float | None = None,
        fee_bybit_pct: float | None = None,
        exchanges: list[str] | None = None,
    ) -> str:
        body: dict[str, Any] = {"clientId": client_id}
        if notional is not None:
            body["notional"] = notional
        if min_profit is not None:
            body["minProfit"] = min_profit
        if min_duration_sec is not None:
            body["minDurationSec"] = min_duration_sec
        if min_net_diff_pct is not None:
            body["minNetDiffPct"] = min_net_diff_pct
        if fee_binance_pct is not None:
            body["feeBinancePct"] = fee_binance_pct
        if fee_coinbase_pct is not None:
            body["feeCoinbasePct"] = fee_coinbase_pct
        if fee_bybit_pct is not None:
            body["feeBybitPct"] = fee_bybit_pct
        if exchanges is not None:
            body["exchanges"] = exchanges
        return http.patch(f"/api/v1/price-diff/watches/{watch_id}", body)

    def pause_price_diff_watch(client_id: str, watch_id: str) -> str:
        return http.post(
            f"/api/v1/price-diff/watches/{watch_id}/pause?clientId={client_id}",
            {},
        )

    def resume_price_diff_watch(client_id: str, watch_id: str) -> str:
        return http.post(
            f"/api/v1/price-diff/watches/{watch_id}/resume?clientId={client_id}",
            {},
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

    def quote_price_diff(
        symbol: str,
        buy_exchange: str,
        sell_exchange: str,
        notional: float | None = None,
        quantity: float | None = None,
        fee_buy_pct: float = 0,
        fee_sell_pct: float = 0,
        min_net_diff_pct: float = 0,
    ) -> str:
        params: dict[str, str] = {
            "symbol": symbol,
            "buyExchange": buy_exchange,
            "sellExchange": sell_exchange,
            "feeBuyPct": str(fee_buy_pct),
            "feeSellPct": str(fee_sell_pct),
        }
        if min_net_diff_pct:
            params["minNetDiffPct"] = str(min_net_diff_pct)
        if notional is not None:
            params["notional"] = str(notional)
        if quantity is not None:
            params["quantity"] = str(quantity)
        return http.get("/api/v1/price-diff/quote", params)

    def quote_price_diff_opportunity(
        client_id: str,
        opportunity_id: str,
        notional: float | None = None,
        quantity: float | None = None,
    ) -> str:
        params: dict[str, str] = {"clientId": client_id}
        if notional is not None:
            params["notional"] = str(notional)
        if quantity is not None:
            params["quantity"] = str(quantity)
        return http.get(
            f"/api/v1/price-diff/opportunities/{opportunity_id}/quote",
            params,
        )

    def scan_price_diff_quotes(
        symbol: str,
        notional: float | None = None,
        quantity: float | None = None,
        fee_binance_pct: float = 0,
        fee_coinbase_pct: float = 0,
        fee_bybit_pct: float = 0,
        min_net_diff_pct: float = 0,
        min_profit_pct: float = 0,
        min_profit_amount: float = 0,
    ) -> str:
        params: dict[str, str] = {
            "symbol": symbol,
            "feeBinancePct": str(fee_binance_pct),
            "feeCoinbasePct": str(fee_coinbase_pct),
            "feeBybitPct": str(fee_bybit_pct),
        }
        if min_net_diff_pct:
            params["minNetDiffPct"] = str(min_net_diff_pct)
        if min_profit_pct:
            params["minProfitPct"] = str(min_profit_pct)
        if min_profit_amount:
            params["minProfitAmount"] = str(min_profit_amount)
        if notional is not None:
            params["notional"] = str(notional)
        if quantity is not None:
            params["quantity"] = str(quantity)
        return http.get("/api/v1/price-diff/quote/scan", params)

    def quote_price_diff_watch(
        client_id: str,
        watch_id: str,
        notional: float | None = None,
        quantity: float | None = None,
        min_profit_pct: float = 0,
        min_profit_amount: float = 0,
    ) -> str:
        params: dict[str, str] = {"clientId": client_id}
        if notional is not None:
            params["notional"] = str(notional)
        if quantity is not None:
            params["quantity"] = str(quantity)
        if min_profit_pct:
            params["minProfitPct"] = str(min_profit_pct)
        if min_profit_amount:
            params["minProfitAmount"] = str(min_profit_amount)
        return http.get(f"/api/v1/price-diff/watches/{watch_id}/quote", params)

    def create_scanner_rule(
        client_id: str,
        rule_type: str = "",
        conditions: str = "",
        match_mode: str = "all",
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
        body: dict[str, object] = {
            "clientId": client_id,
            "interval": interval,
            "matchMode": match_mode,
            "rsiPeriod": rsi_period,
            "rsiCondition": rsi_condition,
            "rsiThreshold": rsi_threshold,
            "maFastPeriod": ma_fast_period,
            "maSlowPeriod": ma_slow_period,
            "maDirection": ma_direction,
            "volumeLookback": volume_lookback,
            "volumeMinRatio": volume_min_ratio,
        }
        if rule_type:
            body["type"] = rule_type
        conds = [c.strip() for c in conditions.split(",") if c.strip()]
        if conds:
            body["conditions"] = conds
        return http.post("/api/v1/scanner/rules", body)

    def list_scanner_rules(client_id: str) -> str:
        return http.get("/api/v1/scanner/rules", {"clientId": client_id})

    def update_scanner_rule(
        client_id: str,
        rule_id: str,
        enabled: bool | None = None,
        interval: str | None = None,
        conditions: str | None = None,
        match_mode: str | None = None,
        rsi_period: int | None = None,
        rsi_condition: str | None = None,
        rsi_threshold: float | None = None,
        ma_fast_period: int | None = None,
        ma_slow_period: int | None = None,
        ma_direction: str | None = None,
        volume_lookback: int | None = None,
        volume_min_ratio: float | None = None,
    ) -> str:
        body: dict[str, object] = {"clientId": client_id}
        if enabled is not None:
            body["enabled"] = enabled
        if interval:
            body["interval"] = interval
        if conditions:
            body["conditions"] = [c.strip() for c in conditions.split(",") if c.strip()]
        if match_mode:
            body["matchMode"] = match_mode
        if rsi_period is not None:
            body["rsiPeriod"] = rsi_period
        if rsi_condition:
            body["rsiCondition"] = rsi_condition
        if rsi_threshold is not None:
            body["rsiThreshold"] = rsi_threshold
        if ma_fast_period is not None:
            body["maFastPeriod"] = ma_fast_period
        if ma_slow_period is not None:
            body["maSlowPeriod"] = ma_slow_period
        if ma_direction:
            body["maDirection"] = ma_direction
        if volume_lookback is not None:
            body["volumeLookback"] = volume_lookback
        if volume_min_ratio is not None:
            body["volumeMinRatio"] = volume_min_ratio
        return http.patch(f"/api/v1/scanner/rules/{rule_id}", body)

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
            description="Spot delistings in the last 30 days or next ~31 days (symbol, halt delistTime, announcedAt when the notice was published). exchange=binance|bybit (Coinbase has no public calendar).",
            args_schema=DelistInput,
        ),
        StructuredTool.from_function(
            get_post_delist,
            name="get_post_delist",
            description=(
                "After a venue delisted a pair, off-venue last price and candles "
                "(another listed venue or CoinGecko USD). Informational only — not "
                "this exchange's book. Listed/upcoming pairs return available=false."
            ),
            args_schema=PostDelistInput,
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
            get_orderbook_icebergs,
            name="get_orderbook_icebergs",
            description=(
                "Detect iceberg-style refill: a visible buy or sell clip is "
                "eaten at the same price, then a similar size comes back, "
                "repeatedly. Both sides. Clip size, refill count, executed "
                "notional. Book pattern only, not proof of a hidden order."
            ),
            args_schema=IcebergsInput,
        ),
        StructuredTool.from_function(
            get_orderbook_liquidity_flow,
            name="get_orderbook_liquidity_flow",
            description=(
                "When resting size at a price shrinks (e.g. 8M bid at 65500 "
                "down to 2M), split how much was actually traded vs pulled "
                "from the book. Same for asks. series is added/pulled/traded "
                "over time. Stored spot books plus spot prints. window "
                "5m/15m/1h or start_time/end_time. Optional min_price/max_price."
            ),
            args_schema=OrderBookLiquidityFlowInput,
        ),
        StructuredTool.from_function(
            get_orderbook_history,
            name="get_orderbook_history",
            description=(
                "Stored spot order-book snapshots: bid/ask levels, spread, "
                "total liquidity, imbalance, and large walls. Pass at for the "
                "book nearest that time. Omit at to list recent samples."
            ),
            args_schema=OrderBookHistoryInput,
        ),
        StructuredTool.from_function(
            compare_orderbook_history,
            name="compare_orderbook_history",
            description=(
                "Compare two stored spot order books and show which price "
                "levels gained or lost liquidity, plus mid/spread/imbalance "
                "change and walls that appeared or were pulled."
            ),
            args_schema=OrderBookCompareInput,
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
            get_liquidation_overview,
            name="get_liquidation_overview",
            description=(
                "Market-wide futures liquidations: long vs short notional for the last "
                "1 hour, 4 hours, 12 hours, and 24 hours, plus coins ranked by total "
                "notional for a treemap. Binance USD-M + Bybit linear. exchange=all sums "
                "both. window (1h|4h|12h|24h, default 24h) ranks coins. Prefer this for "
                "the overall market liquidation situation."
            ),
            args_schema=LiquidationOverviewInput,
        ),
        StructuredTool.from_function(
            get_liquidation_levels,
            name="get_liquidation_levels",
            description=(
                "CoinGlass-style liquidation bar chart. symbol=BTCUSDT returns "
                "estimated long/short notional at each price plus the walk from "
                "last (cumLong/cumByLeverage). review scores later same-venue "
                "liquidations against those estimated bands (hit rate, fill, "
                "reliability by leverage). Combined lastPrice is OI-weighted "
                "of venues that have a last. symbol=all returns observed time "
                "bars of total liquidations across every coin. "
                "exchange=binance|bybit|okx|all. range=12h|24h|3d|7d."
            ),
            args_schema=LiquidationLevelsInput,
        ),
        StructuredTool.from_function(
            get_liquidation_cascade,
            name="get_liquidation_cascade",
            description=(
                "Detect a liquidation cascade: a short burst of long or short "
                "liquidations far above that stream's own typical rate "
                "(1m / 5m / 15m vs the prior 6 hours). episodes lists each "
                "wave in the last 24h (start, duration, long/short notional, "
                "price move). Combined waves include startedBy, "
                "venueStarts.lagSec, and venueStarts.sharePct / totalNotional "
                "(how much of the wave each venue produced). Binance, "
                "Bybit, and OKX are scored separately; a combined episode is "
                "the same-side wave on more than one venue at once. "
                "both.agree is true only when the same side is cascade or "
                "hotter on Binance, Bybit, and OKX. A quiet or opposite venue "
                "keeps it off. symbol=all is market-wide."
            ),
            args_schema=LiquidationCascadeInput,
        ),
        StructuredTool.from_function(
            scan_liquidation_cascades,
            name="scan_liquidation_cascades",
            description=(
                "Market-wide liquidation cascade scan: pooled market risk plus "
                "coins currently bursting (elevated / cascade / extreme). "
                "A hit's both flag means the same side is cascade or hotter on "
                "Binance, Bybit, and OKX."
            ),
            args_schema=LiquidationCascadeScanInput,
        ),
        StructuredTool.from_function(
            get_orderbook_heatmap,
            name="get_orderbook_heatmap",
            description=(
                "Resting bid/ask size over time (order heatmap). Each column is a live "
                "book snapshot, not executed volume. Use for where size has been sitting "
                "and whether a wall persisted. window is lookback seconds (default 600)."
            ),
            args_schema=OrderBookHeatmapInput,
        ),
        StructuredTool.from_function(
            get_options_overview,
            name="get_options_overview",
            description=(
                "BTC/ETH options desk from Deribit: listed open interest, 24h volume, "
                "put/call ratios, DVOL (30-day implied vol), 30-day realized vol from "
                "Binance daily closes, volCompare (dvol / realized / gap in vol points; "
                "DVOL 70 vs realized 45 is gap +25), ATM IV near 30 days, "
                "IV term structure, call/put skew (put minus call), expiry mix, "
                "1-SD expected move and max pain per expiry, dealer gamma "
                "(gamma.netUsd / callUsd / putUsd as USD delta per 1% spot move; "
                "regime long=stabler, short=faster; flip is where net GEX crosses zero). "
                "history is 1h/4h/24h/7d change plus a series for OI, put/call, DVOL, "
                "30-day realized, volGap, net dealer GEX, and flip vs index. "
                "moves lists only unusual short-window OI, put/call, or IV vs that "
                "series' typical, plus gamma / flip-side changes. "
                "vsMarket.findings lists only when implied vol or put/call lean "
                "disagrees with live spot range or futures funding / account long-short. "
                "asset is BTC or ETH. "
                "Not futures OI."
            ),
            args_schema=OptionsOverviewInput,
        ),
        StructuredTool.from_function(
            get_open_interest,
            name="get_open_interest",
            description=(
                "Futures open interest for a coin: current outstanding size plus "
                "how much OI opened or closed in the last 1 hour, 4 hours, and "
                "24 hours as USDT amount and percent, per venue and combined. "
                "flow=opened|closed; openedValue/closedValue is the USDT added "
                "or closed. contracts is base-asset size; value is USDT notional. "
                "Binance USD-M + Bybit linear + OKX. exchange=all sums venues."
            ),
            args_schema=OpenInterestInput,
        ),
        StructuredTool.from_function(
            compare_spot_perp_liquidity,
            name="compare_spot_perp_liquidity",
            description=(
                "Compare resting liquidity in the spot book and the perpetual "
                "book at the same distance from mid (0.1% / 0.5% / 1%). Says "
                "which book is thinner and which side (bid or ask) is thinner "
                "on each. Spot sums Binance, Coinbase, and Bybit. Perp sums "
                "Binance USD-M, Bybit linear, and OKX. Binance and Bybit rows "
                "compare that venue's own two books."
            ),
            args_schema=SpotPerpLiquidityInput,
        ),
        StructuredTool.from_function(
            get_perp_depth,
            name="get_perp_depth",
            description=(
                "How much perpetual liquidity is waiting below vs above the "
                "price. Example: 40M bid within 1% below mid vs 15M ask above. "
                "Binance USD-M, Bybit linear, and OKX USDT-swap separately plus combined. "
                "range_pct default 1. This is the futures book, not spot."
            ),
            args_schema=PerpDepthInput,
        ),
        StructuredTool.from_function(
            get_derivatives_board,
            name="get_derivatives_board",
            description=(
                "One sortable board for the liquid USDT perpetuals: mark price, "
                "predicted funding, open interest, 1h and 24h open-interest "
                "change, crowd long percent, top-minus-crowd gap, day CVD, "
                "15m print-size change, 15m order-count ratio, the farthest "
                "unusual reading, 5m liquidation heat, 1% bid share, "
                "1h taker buy share, and 24h long and short liquidations. "
                "exchange=all reads Binance USD-M, Bybit linear, and OKX. A "
                "combined number is present only when every expected venue is "
                "fresh for that column. gapPct is top-trader long percent minus "
                "crowd long percent; Bybit does not publish a top-trader ratio, "
                "so gapScope names who is in the average. cvdDay is buy minus "
                "sell since 00:00 UTC. cvdSplit is the current run where price "
                "and CVD disagree (price_up_cvd_down or price_down_cvd_up) and "
                "cvdSplitDuration says how long it has run, or agrees when they "
                "are in line. It stays blank until the tape reaches 00:00 UTC, "
                "the newest minute is fresh, and the newest bar has a price. "
                "Combined uses the summed series and stays blank unless every "
                "expected venue's 24h tape is complete. A later bar where price "
                "and CVD move the same way ends the run. Flat bars after it "
                "do not. A 4-hour window that "
                "disagrees without a current run is agrees, not a run. "
                "printChangePct compares the last 15 "
                "minutes of average fill size with the 15 minutes before. "
                "orderRatio is (buy - sell) / (buy + sell) over those same 15 "
                "minutes and stays blank unless every venue's tape covers the "
                "window. unusualMetric names the farthest high or low reading "
                "versus that coin's own samples, and unusualVenue says which "
                "venue it is. Venues are not averaged. A coin with no standout "
                "stays blank. "
                "liqHeat is the hottest 5-minute liquidation grade versus that "
                "coin's own typical rate and stays blank until the venue has "
                "been watched for 6 hours. bidPct is resting bid share within "
                "1% of mid. definite is false when any coin is incomplete on "
                "the original columns; missing and old venues are named and "
                "are not treated as zero. The 24h open-interest change uses "
                "the stored sample within 90 minutes of 24 hours ago when the "
                "live series does not reach that hour. change24hMissing names "
                "each venue with no sample, and that venue is not treated as "
                "zero. limit default 36, max 100."
            ),
            args_schema=DerivativesBoardInput,
        ),
        StructuredTool.from_function(
            get_perp_book_history,
            name="get_perp_book_history",
            description=(
                "Stored perpetual bid and ask within 0.1%, 0.5%, and 1% of mid "
                "for one coin. A minute is kept only when Binance, Bybit, and "
                "OKX are all fresh. Bands the snapshot does not reach are left "
                "out. window_sec is the lookback, default 24h, max 30 days. "
                "The liquid USDT list is sampled while the server runs."
            ),
            args_schema=PerpBookHistoryInput,
        ),
        StructuredTool.from_function(
            get_perp_depth_overview,
            name="get_perp_depth_overview",
            description=(
                "How much total perpetual bid vs ask liquidity is waiting "
                "across the liquid coins we track, plus which coins make up "
                "the biggest part of that book. Example: 2.4B bid vs 1.8B ask "
                "within 1% of mid; BTC 48%, ETH 17%. windows (15m/1h/4h) "
                "compare bid vs ask share of that book (bids 55% → 65%) and "
                "whether total bid+ask got thicker or thinner (at least 5%). "
                "drivers name the coins that moved the mix. sizeDrivers name "
                "the coins that added or removed the most size. history is "
                "total bid+ask each minute for the last 6 hours, with BTC's close "
                "on the same minutes. Sampled while "
                "the desk runs. exchange=all adds Binance, Bybit, and OKX per coin. "
                "range_pct default 1. limit default 20."
            ),
            args_schema=PerpDepthOverviewInput,
        ),
        StructuredTool.from_function(
            get_funding_rate,
            name="get_funding_rate",
            description=(
                "Perpetual futures funding rate: predicted next payment plus recent "
                "settled history. rate is decimal (0.0001 = 0.01%); ratePct is percent. "
                "payer=long means longs pay shorts. Binance USD-M + Bybit linear + OKX. "
                "Per-venue rows stay separate. pressure is OI-weighted combined predicted "
                "funding (USDT open-interest share) so a large book outweighs a small one; "
                "pressureWindows (1h/4h/24h) and pressureMoves show how that blend changed."
            ),
            args_schema=FundingRateInput,
        ),
        StructuredTool.from_function(
            get_funding_arb,
            name="get_funding_arb",
            description=(
                "Cross-exchange perpetual funding opportunity: which venue to "
                "long (cheaper funding) and which to short (richer funding) on "
                "Binance USD-M vs Bybit linear. Sizes the next settlement and a "
                "hold window on the notional you enter, subtracts taker fees, "
                "and shows each venue's spot vs perpetual gap. worthIt is true "
                "only when predicted funding over holdHours covers round-trip "
                "fees. Not an executable trade. Not financial advice."
            ),
            args_schema=FundingArbInput,
        ),
        StructuredTool.from_function(
            scan_funding_arb,
            name="scan_funding_arb",
            description=(
                "Rank liquid USDT coins by after-fee funding spread between "
                "Binance and Bybit. Long the cheaper-funding venue, short the "
                "richer one. Uses the notional you enter and holdHours "
                "(default 24). Not financial advice."
            ),
            args_schema=FundingArbScanInput,
        ),
        StructuredTool.from_function(
            get_funding_arb_history,
            name="get_funding_arb_history",
            description=(
                "Past Binance, Bybit, and OKX funding opportunities for one coin "
                "in a date range. Uses settled funding prints only. Long is the "
                "lowest known rate and short is the highest. OKX uses the Bybit "
                "fee. Each run is a "
                "stretch of the same long/short pair and is listed only when "
                "settled funding minus round-trip fees is positive. The first "
                "clock of a run is the entry signal and is not collected "
                "profit. A direction flip at settlement still pays the old "
                "sides, then starts a new run. start/end are RFC3339 or "
                "YYYY-MM-DD (UTC), max 30 days."
            ),
            args_schema=FundingArbHistoryInput,
        ),
        StructuredTool.from_function(
            create_funding_arb_watch,
            name="create_funding_arb_watch",
            description=(
                "Follow funding-arb opportunities. Omit symbol (or use scan/*) "
                "to scan liquid coins and notify when a new coin's after-fee "
                "net is at least min_profit. Same coin + same long/short while "
                "still above the floor does not re-notify; it must drop below "
                "and come back to notify again. duration_hours is how long "
                "the follow should run (omit for no end). Optional symbol "
                "follows one pair only. Not financial advice."
            ),
            args_schema=FundingArbWatchCreateInput,
        ),
        StructuredTool.from_function(
            list_funding_arb_watches,
            name="list_funding_arb_watches",
            description="List funding-arb follow watches for a clientId.",
            args_schema=ClientIdInput,
        ),
        StructuredTool.from_function(
            get_funding_arb_watch,
            name="get_funding_arb_watch",
            description="Get one funding-arb follow watch by id.",
            args_schema=FundingArbWatchIdInput,
        ),
        StructuredTool.from_function(
            update_funding_arb_watch,
            name="update_funding_arb_watch",
            description="Change minProfit and other settings on an existing funding-arb follow without deleting it.",
            args_schema=FundingArbWatchUpdateInput,
        ),
        StructuredTool.from_function(
            pause_funding_arb_watch,
            name="pause_funding_arb_watch",
            description="Pause a funding-arb follow without deleting it or its signals.",
            args_schema=FundingArbWatchIdInput,
        ),
        StructuredTool.from_function(
            resume_funding_arb_watch,
            name="resume_funding_arb_watch",
            description="Resume a paused funding-arb follow.",
            args_schema=FundingArbWatchIdInput,
        ),
        StructuredTool.from_function(
            delete_funding_arb_watch,
            name="delete_funding_arb_watch",
            description="Delete a funding-arb follow watch and its signals.",
            args_schema=FundingArbWatchIdInput,
        ),
        StructuredTool.from_function(
            list_funding_arb_signals,
            name="list_funding_arb_signals",
            description="List funding-arb crossings above minProfit (status open|closed|all). Includes predicted-vs-settled review when clocks have passed.",
            args_schema=FundingArbSignalListInput,
        ),
        StructuredTool.from_function(
            get_funding_arb_signal,
            name="get_funding_arb_signal",
            description="One funding-arb crossing plus predicted-vs-settled review after funding clocks pass.",
            args_schema=FundingArbSignalIdInput,
        ),
        StructuredTool.from_function(
            get_long_short_ratio,
            name="get_long_short_ratio",
            description=(
                "Futures long/short account ratio: share of accounts that are long "
                "vs short (not position size). On Binance, gap is top-trader long% "
                "minus crowd long% in percentage points (65% vs 55% → +10); "
                "gapWindows (1h/4h/24h) say if that spread widened, narrowed, or "
                "flipped. divergence is when top traders get more long while price "
                "falls (or more short while price rises), with since/duration. "
                "ratio is long/short (1 = even). bias is long if ratio≥1.05, "
                "short if ≤0.95. 5-minute samples. Binance USD-M, Bybit linear, "
                "and OKX USDT-swap. OKX top traders are the top 5% by position "
                "value. exchange=all is per venue, not averaged."
            ),
            args_schema=LongShortInput,
        ),
        StructuredTool.from_function(
            estimate_liquidation_hunt,
            name="estimate_liquidation_hunt",
            description=(
                "Hypothetical only: if Binance or Bybit walked their own spot "
                "book, or OKX was priced from its USDT-swap mark with an empty "
                "spot ladder, to push price and force futures liquidations, where is "
                "the main long/short pressure, how much spot buy/sell the "
                "visible book needs, and a rough desk result (book-only vs "
                "with cascade exit). Also returns upScore/downScore (0-100) "
                "and bias with reasons from zone distance, book cost, "
                "price+OI trend, crowding/funding, and recent flow. Scores "
                "do not change zone or P&L numbers. Venues are never "
                "averaged. Not evidence of exchange behavior. Not financial advice."
            ),
            args_schema=LiquidationHuntInput,
        ),
        StructuredTool.from_function(
            get_liquidation_heatmap,
            name="get_liquidation_heatmap",
            description=(
                "Price × time liquidation intensity map (CoinGlass-style) from "
                "historical open interest, price, and the hunt leverage mix. "
                "Cells are the leverage estimate only. Measured prints are "
                "get_liquidation_print_heatmap. range=12h|24h|3d|7d|14d|30d|90d. "
                "Windows past 24 hours read one stored open-interest, funding, "
                "and long/short sample per bucket from the archive. A column "
                "with no sample stays empty. The review's liquidation check "
                "still uses the newest 2,500 prints. "
                "90d is one column per day. Binance public open interest and "
                "long/short are 30 days; older Binance columns are only rows "
                "this process already stored. Bybit public open interest goes "
                "back to listing and this archive keeps 90 days. OKX hourly "
                "open interest is the latest 1,440 points. OKX prices are that "
                "venue's USDT-swap candles. Each venue uses its stored account "
                "ratio. A column with no stored ratio, including OKX before "
                "the first sample, is an even split. A column with no stored "
                "open interest uses that venue's public hourly history, "
                "clamped to the documented lookback (OKX: the latest 1,440 "
                "hours), and does not extend the archive. Combined sums a column "
                "only when Binance, Bybit, and OKX all have open interest. "
                "Returns Binance, Bybit, OKX, and combined grids plus the "
                "leverage mix. Not financial advice."
            ),
            args_schema=LiquidationHeatmapInput,
        ),
        StructuredTool.from_function(
            get_liquidation_print_heatmap,
            name="get_liquidation_print_heatmap",
            description=(
                "Measured liquidation prints by price and time. Each cell is "
                "stored force-order notional, not an open-interest leverage "
                "estimate. Longs are positions closed by a sell. Shorts are "
                "positions closed by a buy. range=12h|24h|3d|7d|14d|30d|90d. "
                "Public REST refill is about 7 days; older cells are only "
                "prints this process already stored. A column is recorded "
                "when a print landed, or when that venue was watched and the "
                "column is outside a known feed gap. A quiet recorded column "
                "is a real zero. A column before the watch, or inside a gap "
                "with no print, stays empty. Binance, Bybit, and OKX never "
                "borrow each other's prints. Combined sums a column only when "
                "all three were recorded. Not financial advice."
            ),
            args_schema=LiquidationPrintHeatmapInput,
        ),
        StructuredTool.from_function(
            get_liquidation_open_levels,
            name="get_liquidation_open_levels",
            description=(
                "Estimated liquidation levels that stay at the price where "
                "that slice of open interest was placed, until a later candle "
                "trades through them. range=12h|24h|3d|7d|14d|30d|90d. Windows "
                "past 24 hours read one stored open-interest sample per bucket "
                "from the archive. A column with no sample stays empty. The "
                "first column places the open interest already on the books. "
                "Later columns only add an increase. A traded-through level "
                "is not put back. An open-interest drop closes the oldest "
                "slices and is not a liquidation. Longs are hit by a later "
                "low; shorts by a later high. Binance, Bybit, and OKX each "
                "use only their own price and open interest. Combined is "
                "withheld (definite=false) when any venue has none. The "
                "leverage mix is the labeled hunt model with 0.40% "
                "maintenance. Not exchange liquidation-engine output. Not "
                "financial advice."
            ),
            args_schema=LiquidationOpenLevelsInput,
        ),
        StructuredTool.from_function(
            get_squeeze_risk,
            name="get_squeeze_risk",
            description=(
                "Long-squeeze and short-squeeze risk scores (0–100) for a coin "
                "on Binance and Bybit, plus an OI-weighted combined view. "
                "Explains which side is crowded and why risk is high or low. "
                "Uses OI change, funding, long/short accounts, liquidations. "
                "Not a prediction or financial advice."
            ),
            args_schema=SqueezeRiskInput,
        ),
        StructuredTool.from_function(
            get_positioning,
            name="get_positioning",
            description=(
                "Price + open interest positioning: long_buildup, short_buildup, "
                "long_unwinding, or short_covering for Binance and Bybit, plus "
                "a general market (combined) direction. Short summary and why. "
                "Not a prediction or financial advice."
            ),
            args_schema=PositioningInput,
        ),
        StructuredTool.from_function(
            get_venue_divergence,
            name="get_venue_divergence",
            description=(
                "Compare Binance, Bybit, and OKX for one coin. groups lists "
                "venues moving the same way. outliers names the venue moving "
                "differently when the other two agree. Each of OI, funding, "
                "crowding, and positioning has a reading per venue. OKX has "
                "no account long/short."
            ),
            args_schema=VenueDivergenceInput,
        ),
        StructuredTool.from_function(
            get_volume_share,
            name="get_volume_share",
            description=(
                "How BTC (or another coin) futures volume splits across "
                "Binance, Bybit, and OKX. Example: Binance 55%, Bybit 30%, "
                "OKX 15%. 15m/1h/4h also show who gained or lost share vs "
                "the prior window."
            ),
            args_schema=TakerFlowInput,
        ),
        StructuredTool.from_function(
            get_participation,
            name="get_participation",
            description=(
                "Share of futures volume that stuck as new (or closed) open "
                "interest: OI USDT delta divided by taker volume. 2B volume "
                "and OI +300M is ratio 0.15 (15% new positions). Windows "
                "15m, 1h, and 4h. Binance + Bybit + combined."
            ),
            args_schema=TakerFlowInput,
        ),
        StructuredTool.from_function(
            get_print_size,
            name="get_print_size",
            description=(
                "Average USDT size per aggressive futures fill, buy vs sell, "
                "for 5m and 15m versus the window before. Use when volume is "
                "high and you need to know if it was a few large market orders "
                "(avg jumped, e.g. 20K to 180K) or many small prints. "
                "read=few_large or many_small. Binance + Bybit."
            ),
            args_schema=TakerFlowInput,
        ),
        StructuredTool.from_function(
            get_print_size_overview,
            name="get_print_size_overview",
            description=(
                "Whole-market average print size. Sums aggressive futures "
                "notional and fill counts across the liquid USDT pairs, then "
                "divides. Compares 5m and 15m with the window just before so "
                "you can see if prints got larger or smaller. A busier coin "
                "weighs more. Use this for the whole market, not one coin."
            ),
            args_schema=PrintSizeOverviewInput,
        ),
        StructuredTool.from_function(
            get_bot_tracker,
            name="get_bot_tracker",
            description=(
                "Repeated same-size aggressive futures prints. A size is "
                "flagged when it hits at least 4 times inside any 60 seconds "
                "during the last 5 minutes. Buy and sell are separate. Use "
                "this when the same market-order size keeps coming back, not "
                "for the average print size. Not proof of a named bot."
            ),
            args_schema=TakerFlowInput,
        ),
        StructuredTool.from_function(
            get_order_count_ratio,
            name="get_order_count_ratio",
            description=(
                "Market order count ratio from -1 to +1: (buy print count - "
                "sell print count) / (buy + sell) over 1m, 5m, and 15m. Counts "
                "aggressive fills, not USDT size. Use for whether more market "
                "orders are buying or selling. Repeated clip size is get_bot_tracker."
            ),
            args_schema=TakerFlowInput,
        ),
        StructuredTool.from_function(
            get_order_count_overview,
            name="get_order_count_overview",
            description=(
                "Whole-market order count ratio. Sums aggressive buy and sell "
                "print counts across the most liquid USDT pairs, then divides. "
                "Ratio is -1 to +1. A busier coin counts for more of the ratio. "
                "buyCoins and sellCoins are one vote per pair. Use this for the "
                "whole market, not one coin."
            ),
            args_schema=OrderCountOverviewInput,
        ),
        StructuredTool.from_function(
            get_net_position,
            name="get_net_position",
            description=(
                "Estimate net long and net short USDT over 15m, 1h, and 4h "
                "from open-interest change plus taker buy and sell. Positive "
                "means that side grew, negative means it shrank. The two nets "
                "add up to the open-interest change. Binance and Bybit. Not "
                "the account long/short ratio."
            ),
            args_schema=TakerFlowInput,
        ),
        StructuredTool.from_function(
            get_taker_flow,
            name="get_taker_flow",
            description=(
                "Aggressive futures buy vs sell volume (who is hitting the "
                "book) for 5m, 1h, and 4h on Binance and Bybit, plus combined."
            ),
            args_schema=TakerFlowInput,
        ),
        StructuredTool.from_function(
            get_market_pressure,
            name="get_market_pressure",
            description=(
                "One -1 to +1 score for which side has more pressure when the "
                "resting order book, CVD, and absorption disagree. +1 is buy, "
                "-1 is sell. When the book looks bid-heavy but CVD is selling, "
                "absorption is weighted more. change1h is this score minus ~1h ago."
            ),
            args_schema=TakerFlowInput,
        ),
        StructuredTool.from_function(
            get_cvd,
            name="get_cvd",
            description=(
                "Cumulative Volume Delta: running sum of market-buy minus "
                "market-sell notional over time, with price. Futures and spot "
                "are separate; spotFutures flags when spot buying and futures "
                "selling (or the reverse). Windows show 15m / 1h / 4h / 24h "
                "change on the returned bars. Anchors reset the sum at 00:00 UTC (day) and at the "
                "current 4-hour open. anchor is an optional RFC3339 candle time: a click "
                "anchor sums bars from that candle and is incomplete when the tape starts "
                "later. Choose a range that reaches it. range is 24h (5-minute bars), 7d (30-minute), "
                "30d (2-hour), or 90d (4-hour) from stored minutes. "
                "Binance minutes for the liquid list are filled from public "
                "aggTrades, about 7 days. Bybit and OKX stay on minutes this "
                "process recorded. Futures combined is complete only when Binance, "
                "Bybit, and OKX each cover the range. Spot combined "
                "(spotVenues / spotCombined) needs Binance and Bybit. OKX has "
                "no spot CVD. A missing venue keeps combined incomplete. "
                "whaleDelta and retailDelta split a futures bar when print size "
                "was stored: a whale is one aggressive print of at least "
                "100,000 USD, and retail is every smaller print. A minute "
                "stored before size was known omits both and is not zero. "
                "Combined size is set only when every venue's bar at that step "
                "was sized. Spot points do not carry the split. "
                "Price vs CVD divergence is the current consecutive "
                "run only. Combined shows Binance vs Bybit when they disagree."
            ),
            args_schema=CvdInput,
        ),
        StructuredTool.from_function(
            get_positioning_anchor,
            name="get_positioning_anchor",
            description=(
                "Restart stored open interest, funding, and liquidation notional "
                "at one candle for Binance, Bybit, and OKX. Open interest and "
                "funding are the change from the last sample at or before that "
                "candle. A venue with no sample by then is named and left off. "
                "Liquidation notional sums from that candle, short positive and "
                "long negative. A first print after the candle is late and the "
                "line starts there. anchor is RFC3339. bucket is 5m, 30m, 2h, "
                "or 4h."
            ),
            args_schema=PositioningAnchorInput,
        ),
        StructuredTool.from_function(
            get_tape_backtest,
            name="get_tape_backtest",
            description=(
                "Replay stored tape for one coin and score the spot close "
                "1 hour, 4 hours, and 24 hours later. cvd is a finished UTC "
                "hour whose buy or sell share crosses the threshold. "
                "liquidation is a 5-minute force-order burst versus that "
                "side's own median (not estimated leverage levels). book is "
                "a 15-minute spot bid or ask share cross. funding is a settled "
                "funding print in percent. oi is the open-interest percent "
                "change from about 24 hours earlier. A missing sample is left "
                "out. gap is top-trader long share minus the crowd, in "
                "percentage points, on Binance and OKX. size is the share of "
                "a finished hour from prints of at least $100,000. Minutes "
                "stored before print size was known are left out. Each horizon "
                "includes average, median, win rate, and worst drop. Worst "
                "drop is the lowest close versus the entry. Zero means price "
                "did not close below the entry. exchange=all withholds "
                "combined unless every expected venue has the bar. "
                "Informational only."
            ),
            args_schema=TapeBacktestInput,
        ),
        StructuredTool.from_function(
            get_volume_surge,
            name="get_volume_surge",
            description=(
                "How much higher this coin's volume is than its own typical "
                "over 5 minutes, 15 minutes, and 1 hour. Buy and sell are "
                "separate when available so you can see if the spike is "
                "one-sided."
            ),
            args_schema=TakerFlowInput,
        ),
        StructuredTool.from_function(
            get_rsi_heatmap,
            name="get_rsi_heatmap",
            description=(
                "Ranked Wilder RSI scatter for top listed pairs. Each "
                "item is one coin (stables omitted) with rank and RSI "
                "on a single interval (default 1h). Informational only."
            ),
            args_schema=RSIHeatmapInput,
        ),
        StructuredTool.from_function(
            scan_volume_surges,
            name="scan_volume_surges",
            description=(
                "Which coins have much more volume than their own typical "
                "right now. Scans top 24h-volume pairs and ranks by "
                "5m/15m/1h current-vs-median. min_ratio default 2."
            ),
            args_schema=VolumeSurgeScanInput,
        ),
        StructuredTool.from_function(
            get_liquidity_sweeps,
            name="get_liquidity_sweeps",
            description=(
                "Liquidity sweep: price pokes a little through a prior high "
                "or low that had already turned it back, then comes back. "
                "Returns the level, how far through, how long to reclaim, "
                "and volume during that poke."
            ),
            args_schema=TakerFlowInput,
        ),
        StructuredTool.from_function(
            get_sweep_recovery,
            name="get_sweep_recovery",
            description=(
                "How long spot depth took to refill after a liquidity sweep. "
                "Clock starts when price comes back and stops when the swept "
                "side is at least 80% of its pre-poke size, or after two hours."
            ),
            args_schema=TakerFlowInput,
        ),
        StructuredTool.from_function(
            get_absorption,
            name="get_absorption",
            description=(
                "When big market buys or sells hit but price barely moves: "
                "how much buy/sell volume came in, how far price moved, "
                "which side is absorbing (bids absorbing sells, or asks "
                "absorbing buys), and how strong (score 0–100). Windows "
                "15m / 1h / 4h / 24h plus the current run."
            ),
            args_schema=TakerFlowInput,
        ),
        StructuredTool.from_function(
            get_around,
            name="get_around",
            description=(
                "What happened around a chosen time for a coin: before, "
                "during, and after that move. Price, volume, buy/sell, VWAP, "
                "volume vs typical, volume-profile POC, liquidity sweeps, "
                "stored book/futures history, and listed BTC/ETH options "
                "(OI, put/call, IV, unusual prints) when sampled. `at` is "
                "required. window is the lookback/lookforward (default 1h). "
                "during is the move starting at at (default 15m)."
            ),
            args_schema=AroundInput,
        ),
        StructuredTool.from_function(
            compare_around,
            name="compare_around",
            description=(
                "Compare two different times for the same coin: how the "
                "two moves differed in price, volume, vs typical, order "
                "book, open interest, funding, liquidations, and sweeps. "
                "from_time and to_time are the two event times."
            ),
            args_schema=AroundCompareInput,
        ),
        StructuredTool.from_function(
            find_around_moves,
            name="find_around_moves",
            description=(
                "Find the important up and down moves in a coin's recent "
                "history, then show what happened during each one (price, "
                "volume, VWAP, vs typical, POC, sweeps, stored book/futures). "
                "Ranked by |return|. lookback default 24h."
            ),
            args_schema=AroundMovesInput,
        ),
        StructuredTool.from_function(
            find_around_precursors,
            name="find_around_precursors",
            description=(
                "Scan a coin's candles for important up and down moves, "
                "compare the tape before those moves, and list conditions "
                "that show up often — singles and combos that fire together "
                "(volume + book + OI). Combos say whether the group is more "
                "common before increases or drops. Default lookback 7d."
            ),
            args_schema=AroundMovesInput,
        ),
        StructuredTool.from_function(
            find_around_similar,
            name="find_around_similar",
            description=(
                "Compare the current market to the setup before past "
                "important moves. fields selects price,volume,takers,oi,book,options "
                "(e.g. volume,book,oi). weights sets importance "
                "(default book and oi 3, options 2, takers 1.5, volume and price 1). "
                "min_coverage (default 60) sends thin cases to skipped. "
                "Uncompared fields have no score. Each past match "
                "uses only tape available before that move "
                "(dataFrom-dataTo) and lists whySimilar plus per-field "
                "compared scores. Overlapping or chained same-move "
                "matches count as one event. afterHorizons is how "
                "price usually moved after those unique events. "
                "horizons is a CSV of after-windows (default "
                "15m,1h,4h; e.g. 5m,30m,2h) on matching candle "
                "size, with avg, median, up, down, sample, and "
                "similarity bands 40-60/60-80/80-100. "
                "Optional at compares the tape before that time "
                "instead of the current last window."
            ),
            args_schema=AroundSimilarInput,
        ),
        StructuredTool.from_function(
            get_market_forensics,
            name="get_market_forensics",
            description=(
                "Market forensics for a coin and a chosen time: what "
                "happened before, during, and after that move (price, "
                "volume, takers, VWAP, vs typical, POC, sweeps, stored "
                "book/futures, and listed BTC/ETH options when sampled), plus past important-move setups that "
                "looked like the before-window and how price moved after "
                "those. why is a plain-language read of why the move "
                "may have happened (not a metric dump). timeline is one "
                "clock of OI, takers, venue price, follow-through, "
                "liquidations, options, dealer gamma / flip vs spot, the futures curve (contango / backwardation and near-vs-far gap), tapeRead (POC buy vs sell, dominance start, delta vs price), and sweeps. at is required. "
                "Prefer this when the user asks what happened around a "
                "move and what similar past cases did next. If "
                "similar-move search fails, tape is still returned."
            ),
            args_schema=ForensicsInput,
        ),
        StructuredTool.from_function(
            scan_market_forensics,
            name="scan_market_forensics",
            description=(
                "Scan top-volume coins and find which ones currently "
                "look like the setup before their own important past "
                "moves. Returns ranked coins, closeness, whySimilar "
                "(which of price/volume/takers/oi/book/options matched), and "
                "what usually happened after those similar cases."
            ),
            args_schema=ForensicsScanInput,
        ),
        StructuredTool.from_function(
            review_setup_radar,
            name="review_setup_radar",
            description=(
                "Review saved Setup Radar hits and what price did after "
                "each one (15m / 1h / 4h). Usefulness is how often an "
                "up/down lean matched the later move."
            ),
            args_schema=SetupRadarReviewInput,
        ),
        StructuredTool.from_function(
            get_vwap,
            name="get_vwap",
            description=(
                "Volume-weighted average price from a start time (or window) "
                "until now. Prices with more trading pull the average more. "
                "Returns VWAP, how far last is above or below it, and total "
                "volume. Binance and Bybit separately plus combined."
            ),
            args_schema=VWAPInput,
        ),
        StructuredTool.from_function(
            get_volume_profile,
            name="get_volume_profile",
            description=(
                "Where aggressive futures prints happened by price: buy and "
                "sell USDT on the printed price, plus open interest opened "
                "or closed there and long/short liquidations on the same "
                "rows. Point of control is the busiest price; the value area "
                "holds about 70%. Stored minutes, about 30 days. Binance "
                "liquid names are filled from public trades for about 7 days. "
                "Bybit and OKX start when this process recorded them. "
                "combined is present only when every venue reaches the "
                "window open; missing names a later venue and is not zero. "
                "Use window 1h/4h/24h/7d/30d or start_time/end_time."
            ),
            args_schema=VolumeProfileInput,
        ),
        StructuredTool.from_function(
            get_flow_at_prices,
            name="get_flow_at_prices",
            description=(
                "What printed at each price: market buy vs sell USDT, print "
                "count, and delta (buy − sell). Use when a candle's total "
                "buy/sell is not enough — e.g. how much bought and sold at "
                "65520 while BTC traded 65400–65600. Built from aggressive "
                "futures prints, not candle smear. window 5m/15m/1h/4h/1d/3d/7d "
                "or start_time/end_time (max 7d). Live tape is about 1h on "
                "Binance and the newest ~1000 Bybit trades; stored minutes "
                "(about 7 days, while the process runs) cover longer lookbacks. "
                "Optional min_price/max_price and tick_size. Binance + Bybit."
            ),
            args_schema=FlowAtPricesInput,
        ),
        StructuredTool.from_function(
            get_candle_footprint,
            name="get_candle_footprint",
            description=(
                "Buy versus sell inside each candle, at the prices that actually "
                "printed. Each candle has price rows with buy and sell USDT, print "
                "count, delta, the busiest row (isPoc), and imbalance buy or sell "
                "when one side is at least 3× the other. interval matches the chart "
                "(1m, 5m, 15m, 1h, 4h, 1d, 1w, 1M). Optional start_time and end_time "
                "together; otherwise the newest 120 candles. At most 400 candles and "
                "30 days; truncated when the range is shortened. Closed minutes are "
                "the stored ladder (about 30 days). The current minute uses the live "
                "tape when that tape already covers it. This request does not walk "
                "public trade history. Binance minutes for the liquid list are "
                "filled in the background from public aggTrades, about 7 days. "
                "Bybit and OKX stay on minutes this process recorded. "
                "exchange=all returns each venue plus combined. "
                "combined is present only when every venue was recorded for the range. "
                "missing names a venue with no stored minutes and that venue is not "
                "treated as zero. Not financial advice."
            ),
            args_schema=CandleFootprintInput,
        ),
        StructuredTool.from_function(
            get_basis,
            name="get_basis",
            description=(
                "How far the perpetual is from spot/index: premium or discount, "
                "dollar and percent, expanding or shrinking, with funding and "
                "OI context. curve compares listed dated futures (near vs far, "
                "contango / backwardation). curve.history / windows / moves "
                "show when that spread flipped or got steeper (1h/4h/24h/7d). "
                "Binance, Bybit, and OKX. Agreement needs all three usable "
                "quotes. A missing quote leaves it unknown."
            ),
            args_schema=BasisInput,
        ),
        StructuredTool.from_function(
            get_price_correlation,
            name="get_price_correlation",
            description=(
                "How similarly a coin has been moving with BTC and ETH over "
                "the last 1 hour, 4 hours, and 24 hours: correlation, beta, "
                "same-direction share, and whether it follows with a delay."
            ),
            args_schema=CorrelationInput,
        ),
        StructuredTool.from_function(
            get_market_breadth,
            name="get_market_breadth",
            description=(
                "How many of the liquid coins we follow are up vs down over "
                "1h, 4h, and 24h (count and percent). Says whether BTC and "
                "ETH are moving with the rest of the market or a few large "
                "coins are carrying it."
            ),
            args_schema=BreadthInput,
        ),
        StructuredTool.from_function(
            get_fear_greed,
            name="get_fear_greed",
            description=(
                "Swyngora fear and greed index from 0 to 100, calculated from "
                "our own tape: 24h breadth, BTC move, volatility versus usual, "
                "OI-weighted funding, account long/short, 24h liquidations, "
                "perp bid share, and taker flow. Not the Binance or CoinGlass "
                "index. top and factors say which input moved the score the "
                "most. history and windows (1h/4h/24h/7d) are past scores "
                "stored while the backend is running."
            ),
            args_schema=FearGreedInput,
        ),
        StructuredTool.from_function(
            get_price_volatility,
            name="get_price_volatility",
            description=(
                "How volatile a coin has been over 1h, 4h, and 24h: net move, "
                "high-low range, whether the range is bigger than normal or "
                "expanding, and whether it is jumpy or calm versus BTC and ETH."
            ),
            args_schema=VolatilityInput,
        ),
        StructuredTool.from_function(
            get_market_snapshot,
            name="get_market_snapshot",
            description=(
                "Price, volume, market cap, open interest, funding, "
                "long/short, and taker buy/sell together for one coin, "
                "with current values and 1h / 4h / 24h changes. Use when "
                "volume or OI may be building before price moves."
            ),
            args_schema=SnapshotInput,
        ),
        StructuredTool.from_function(
            get_support_resistance,
            name="get_support_resistance",
            description=(
                "Support and resistance areas from price history, volume, "
                "and the live order book. Distance from last, test count, "
                "nearby bid/ask liquidity, and a breakout score when price "
                "is close to or through a level (volume, book, taker flow)."
            ),
            args_schema=LevelsInput,
        ),
        StructuredTool.from_function(
            get_whale_trades,
            name="get_whale_trades",
            description=(
                "Largest recent aggressive futures buys/sells (taker long/"
                "short) and large liquidations, sorted biggest first. Each "
                "row has average price, first/last time, total size, and "
                "whether the print is large versus that coin's market cap. "
                "Omit symbol to scan the top liquid USDT coins."
            ),
            args_schema=WhalesInput,
        ),
        StructuredTool.from_function(
            get_futures_history,
            name="get_futures_history",
            description=(
                "Durable stored history of futures open interest, funding, "
                "long/short ratio, or liquidations for Binance, Bybit, and OKX. "
                "Survives restarts. metric=open_interest|funding|long_short|"
                "liquidations. bucket=5m|30m|2h|4h returns one sample per step "
                "across the window, capped at 90 days. Open interest, funding, "
                "and long/short keep the last sample in the step. Funding "
                "prefers a settled print. Liquidations sum notional. A step "
                "with no sample is omitted and is not zero. Bybit long/short "
                "has no top-trader share. Omit bucket for the newest raw rows."
            ),
            args_schema=FuturesHistoryInput,
        ),
        StructuredTool.from_function(
            get_history_span,
            name="get_history_span",
            description=(
                "How long this coin has actually been recorded. Returns the "
                "oldest stored futures open interest, spot book, and "
                "flow-at-prices sample per venue. A venue with no rows is "
                "omitted. spanSec is from that sample to now. The startup "
                "archive backfills the liquid USDT perpetuals, up to 100 names "
                "with the seed pairs first, only as far as each public endpoint "
                "documents. Binance tape for that list is walked back about "
                "7 days. Informational only."
            ),
            args_schema=HistorySpanInput,
        ),
        StructuredTool.from_function(
            get_unusual_context,
            name="get_unusual_context",
            description=(
                "Whether funding, 24h open-interest change, the last "
                "completed liquidation hour, the last completed CVD day, "
                "and the latest hour of average spread are high or low "
                "versus this coin's own stored history on that venue. "
                "Print size stays unknown: only the previous 15 minutes "
                "are kept. exchange=all returns Binance, Bybit, and OKX "
                "separately. The span is the stored sample span. Too few "
                "samples stay unknown. Informational only, not advice."
            ),
            args_schema=UnusualInput,
        ),
        StructuredTool.from_function(
            scan_unusual,
            name="scan_unusual",
            description=(
                "Ranks the liquid USDT perpetuals, up to 36 with the seed "
                "pairs first, whose funding, 24h open-interest change, last "
                "completed liquidation hour, last completed CVD day, or "
                "latest hour of average spread is high or low versus that "
                "coin's own stored samples. Typical readings and readings "
                "with too few samples are left off. Venues stay separate. "
                "Print size is not scanned. Informational only, not advice."
            ),
            args_schema=UnusualScanInput,
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
            scan_market_impact,
            name="scan_market_impact",
            description=(
                "Rank liquid coins by slippage for the same quote-sized market "
                "order. Default 100000 quote. Wider slippage is first. "
                "exhausted means the visible book could not fill the whole order. "
                "Use this to see which coins slip more for the same order."
            ),
            args_schema=MarketImpactScanInput,
        ),
        StructuredTool.from_function(
            scan_suspicious_walls,
            name="scan_suspicious_walls",
            description=(
                "List suspicious and flickering spot walls across the liquid USDT "
                "pairs so you do not open coins one by one. kind=suspicious means "
                "the wall was added and pulled at least 4 times inside 2 minutes. "
                "kind=flicker means it rested under about 20 seconds and then left. "
                "Empty until the desk has been sampling. Not financial advice."
            ),
            args_schema=WallScanInput,
        ),
        StructuredTool.from_function(
            get_book_gaps,
            name="get_book_gaps",
            description=(
                "Find empty areas in the live spot book. easier=up means asks "
                "near the price are much thinner than bids, so a move up meets "
                "less resting size. easier=down is the mirror. Lists the empty "
                "price stretches. One snapshot, not a forecast."
            ),
            args_schema=BookGapsInput,
        ),
        StructuredTool.from_function(
            scan_book_gaps,
            name="scan_book_gaps",
            description=(
                "Rank liquid coins by which side of the spot book is thinner. "
                "Use this to see where price can move more easily up or down "
                "without opening each coin."
            ),
            args_schema=BookGapScanInput,
        ),
        StructuredTool.from_function(
            get_wide_spread,
            name="get_wide_spread",
            description=(
                "Compare a coin's current bid-ask spread with the median of "
                "earlier samples of that same book. opened is true at 1.5× "
                "its own typical."
            ),
            args_schema=WideSpreadInput,
        ),
        StructuredTool.from_function(
            scan_wide_spreads,
            name="scan_wide_spreads",
            description=(
                "Rank coins whose bid-ask spread is at least 1.5× that coin's "
                "own typical spread. Widest first. Empty until books have been sampled."
            ),
            args_schema=WideSpreadScanInput,
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
            get_holders,
            name="get_holders",
            description=(
                "On-chain holder count, top 10/50/100 concentration, and top wallets "
                "for a crypto asset (BTC or BTCUSDT). Tries CoinMarketCap, GeckoTerminal, "
                "then Ethplorer. 404 if every source misses. Informational only."
            ),
            args_schema=SupplyInput,
        ),
        StructuredTool.from_function(
            get_asset_profile,
            name="get_asset_profile",
            description=(
                "Asset identity: name, logo, listing date, and contract addresses "
                "for a crypto or equity ticker. Informational only."
            ),
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
            list_custom_signal_factors,
            name="list_custom_signal_factors",
            description="List mixable custom-signal factors (swing patterns + hunt_up) and built-in percents.",
        ),
        StructuredTool.from_function(
            list_custom_signal_profiles,
            name="list_custom_signal_profiles",
            description="List saved custom-signal mixes for a clientId.",
            args_schema=ClientIdInput,
        ),
        StructuredTool.from_function(
            create_custom_signal_profile,
            name="create_custom_signal_profile",
            description="Save a named custom-signal mix. weights_json must sum to 100.",
            args_schema=CustomSignalProfileInput,
        ),
        StructuredTool.from_function(
            update_custom_signal_profile,
            name="update_custom_signal_profile",
            description="Update a saved custom-signal mix.",
            args_schema=CustomSignalProfileUpdateInput,
        ),
        StructuredTool.from_function(
            duplicate_custom_signal_profile,
            name="duplicate_custom_signal_profile",
            description="Copy a saved custom-signal mix.",
            args_schema=CustomSignalProfileDeleteInput,
        ),
        StructuredTool.from_function(
            set_default_custom_signal_profile,
            name="set_default_custom_signal_profile",
            description="Open the Custom tab on this saved mix.",
            args_schema=CustomSignalProfileDeleteInput,
        ),
        StructuredTool.from_function(
            delete_custom_signal_profile,
            name="delete_custom_signal_profile",
            description="Delete a saved custom-signal mix.",
            args_schema=CustomSignalProfileDeleteInput,
        ),
        StructuredTool.from_function(
            scan_custom_signals,
            name="scan_custom_signals",
            description=(
                "Scan the watchlist with a custom factor mix. Optional weight_* must sum to 100. "
                "Missing hunt adds 0. scoreCompare is default vs applied. Informational only."
            ),
            args_schema=CustomSignalScanInput,
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
                "backtests, paper portfolios, and/or custom signal profiles as json or csv. One active export per client. "
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
            create_liquidation_feed_alert,
            name="create_liquidation_feed_alert",
            description=(
                "Alert when Binance or Bybit stop sending liquidation data longer than "
                "min_down_seconds (default 300). Fires once, stays quiet while still down, "
                "re-arms when the feed is live again."
            ),
            args_schema=LiquidationFeedAlertInput,
        ),
        StructuredTool.from_function(
            create_liquidation_cascade_alert,
            name="create_liquidation_cascade_alert",
            description=(
                "Alert when a coin has a big liquidation cascade. Payload names the exchange "
                "and coin. symbol=BTCUSDT or all. min_grade=cascade (default) or extreme. "
                "Fires once per wave and does not re-fire while the same cascade stays on."
            ),
            args_schema=LiquidationCascadeAlertInput,
        ),
        StructuredTool.from_function(
            create_liquidation_notional_alert,
            name="create_liquidation_notional_alert",
            description=(
                "Alert when a coin's liquidations in a window exceed a USDT amount. "
                "side=long|short|both, exchange=binance|bybit|all, window=1m|5m|15m|1h. "
                "Fires once when the wave crosses the line and does not re-fire until "
                "the window drops and a new wave starts."
            ),
            args_schema=LiquidationNotionalAlertInput,
        ),
        StructuredTool.from_function(
            create_setup_radar_alert,
            name="create_setup_radar_alert",
            description=(
                "Alert when a coin's current tape looks similar enough to the setup "
                "before that coin's own important past moves (Setup Radar). "
                "min_similarity 40–100 (default 60). lean=any|up|down. "
                "symbol=BTCUSDT or all. Fires once when the match appears and does "
                "not re-fire until the tape no longer looks like those past setups."
            ),
            args_schema=SetupRadarAlertInput,
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
            args_schema=ClientIdInput,
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
            args_schema=ClientIdInput,
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
            args_schema=ClientIdInput,
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
            description="Move virtual cash between your own paper portfolios. Owner only. Pass idempotency_key so a dropped retry does not move cash twice.",
            args_schema=PortfolioTransferInput,
        ),
        StructuredTool.from_function(
            list_transfer_attempts,
            name="list_transfer_attempts",
            description="List transfer requests: completed (cash moved), replayed (same key came again), or conflict (same key, different request).",
            args_schema=TransferAttemptsInput,
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
            description="Set optional risk limits (daily loss %, max one-coin weight %, and/or max total exposure %). Does not close positions.",
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
                "Paper pending order: limit_buy, limit_sell, stop_loss, or trailing_stop. "
                "trailing_stop uses trail_type (percent|offset) and trail_value. "
                "time_in_force gtc|ioc|fok; optional expires_at for GTC. "
                "Pass portfolio_id when the client has more than one book. "
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
                "intervalHours (e.g. 12) for interval. time_zone (Europe/Istanbul) + hour/minute "
                "for a DST-aware local clock (Monday 09:00). max_price skips a run if last or "
                "fee+slippage unit would exceed it. budget is a total cash cap; end_date "
                "(YYYY-MM-DD) or ends_at is the last inclusive run. Simulated only."
            ),
            args_schema=RecurringBuyCreateInput,
        ),
        StructuredTool.from_function(
            update_recurring_buy,
            name="update_recurring_buy",
            description="Update a paper recurring buy name, amount, schedule, time_zone/hour/minute, max_price, budget, or end_date/ends_at.",
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
            description="List execution history for a paper recurring buy plan (succeeded, failed, missed). Succeeded rows include timing on_time or late.",
            args_schema=RecurringBuyRunsInput,
        ),
        StructuredTool.from_function(
            complete_recurring_buy_run,
            name="complete_recurring_buy_run",
            description="Buy a missed paper recurring slot now at the current last price. Marks the fill as late. Does not move the plan's next run.",
            args_schema=RecurringBuyCompleteInput,
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
                "Track a coin across Binance/Coinbase/Bybit. Opens an opportunity only when "
                "the notional can be bought and sold on fresh live books and after-fee profit "
                "(including slippage) is at least min_profit. min_duration_sec keeps the fill "
                "qualifying that long before opening; a break resets the timer."
            ),
            args_schema=PriceDiffWatchCreateInput,
        ),
        StructuredTool.from_function(
            list_price_diff_watches,
            name="list_price_diff_watches",
            description="List cross-exchange price difference watches for a clientId.",
            args_schema=ClientIdInput,
        ),
        StructuredTool.from_function(
            get_price_diff_watch,
            name="get_price_diff_watch",
            description="Get one price-diff watch by id.",
            args_schema=PriceDiffWatchIdInput,
        ),
        StructuredTool.from_function(
            update_price_diff_watch,
            name="update_price_diff_watch",
            description=(
                "Change notional, min_profit, min_duration_sec, or fees on an existing "
                "price-diff watch. Resets the duration timer. Does not change pause/active."
            ),
            args_schema=PriceDiffWatchUpdateInput,
        ),
        StructuredTool.from_function(
            pause_price_diff_watch,
            name="pause_price_diff_watch",
            description=(
                "Pause a price-diff watch. Stops searching and closes any open opportunity. "
                "Resume starts the duration timer from zero."
            ),
            args_schema=PriceDiffWatchIdInput,
        ),
        StructuredTool.from_function(
            resume_price_diff_watch,
            name="resume_price_diff_watch",
            description="Resume a paused price-diff watch. Does not continue a duration wait from before pause.",
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
            quote_price_diff,
            name="quote_price_diff",
            description=(
                "Walk live spot books on two venues for a size (notional USDT or quantity). "
                "Returns average buy/sell, slippage, profit after fees, and max still-profitable size. "
                "Provide notional or quantity, not both. Informational only."
            ),
            args_schema=PriceDiffQuoteInput,
        ),
        StructuredTool.from_function(
            quote_price_diff_opportunity,
            name="quote_price_diff_opportunity",
            description=(
                "Quote a stored price-diff opportunity at a size using that watch's fees "
                "and the live books. Provide notional (USDT on the buy book) or quantity. "
                "Informational only."
            ),
            args_schema=PriceDiffOppQuoteInput,
        ),
        StructuredTool.from_function(
            scan_price_diff_quotes,
            name="scan_price_diff_quotes",
            description=(
                "Compare every Binance/Coinbase/Bybit buy-sell route at one size. "
                "Walks both live books with per-venue fees, ranks by after-fee profit, "
                "and shows how much of the entered money can actually be used. "
                "Provide notional or quantity, not both. Informational only."
            ),
            args_schema=PriceDiffScanInput,
        ),
        StructuredTool.from_function(
            quote_price_diff_watch,
            name="quote_price_diff_watch",
            description=(
                "Scan all venue pairs for a stored watch at a size using that watch's fees. "
                "Ranked by after-fee profit and usable money. Informational only."
            ),
            args_schema=PriceDiffWatchQuoteInput,
        ),
        StructuredTool.from_function(
            create_scanner_rule,
            name="create_scanner_rule",
            description=(
                "Create a technical scanner rule for the client's watchlist. "
                "Pick one or more of rsi, ma_crossover, volume_increase via conditions. "
                "match_mode=all requires every selected condition; match_mode=any needs only one. "
                "Informational only."
            ),
            args_schema=ScannerRuleCreateInput,
        ),
        StructuredTool.from_function(
            list_scanner_rules,
            name="list_scanner_rules",
            description="List technical scanner rules for a clientId.",
            args_schema=ClientIdInput,
        ),
        StructuredTool.from_function(
            update_scanner_rule,
            name="update_scanner_rule",
            description=(
                "Enable or disable a scanner rule, or edit its conditions, "
                "match mode, interval, periods, and thresholds without deleting it."
            ),
            args_schema=ScannerRuleUpdateInput,
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
            args_schema=ClientIdInput,
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
