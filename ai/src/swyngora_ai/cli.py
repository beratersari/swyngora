"""CLI entrypoint: swyngora-ai (Typer)."""

from __future__ import annotations

import os
import sys
import time
from collections.abc import Sequence
from dataclasses import dataclass, field
from enum import StrEnum
from pathlib import Path
from typing import TYPE_CHECKING, Annotated, Any

import typer
from rich.align import Align
from rich.console import Console
from rich.markdown import Markdown
from rich.markup import escape
from rich.progress import Progress, SpinnerColumn, TextColumn, TimeElapsedColumn
from rich.table import Table

from swyngora_ai.llm.errors import parse_usage_emit

if TYPE_CHECKING:
    from swyngora_ai.graph.orchestrator import ChatResult, Orchestrator

app = typer.Typer(
    add_completion=False,
    pretty_exceptions_enable=False,
    context_settings={"help_option_names": ["-h", "--help"]},
)

_console = Console()
_err = Console(stderr=True)

CLI_HISTORY_MAX = 500
REPL_COMMANDS: tuple[str, ...] = ("exit", "quit", "q", "reset")


def default_history_path() -> Path:
    """Per-user prompt history. Outside the repo; created on first REPL submit."""
    return Path.home() / ".swyngora" / "cli_history"


def load_history_lines(path: Path) -> list[str]:
    if not path.is_file():
        return []
    text = path.read_text(encoding="utf-8", errors="replace")
    return [line.strip() for line in text.splitlines() if line.strip()]


def write_history_lines(path: Path, lines: list[str], *, max_lines: int = CLI_HISTORY_MAX) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    kept = [line.strip() for line in lines if line.strip()][-max_lines:]
    path.write_text("\n".join(kept) + ("\n" if kept else ""), encoding="utf-8")


def _command_matches(buffer: str) -> list[str]:
    if " " in buffer:
        return []
    lower = buffer.lower()
    hits = [cmd for cmd in REPL_COMMANDS if cmd.startswith(lower)]
    if buffer and buffer in hits:
        longer = [cmd for cmd in hits if cmd != buffer]
        if longer:
            return longer
    return hits


@dataclass
class ReplCompleteState:
    """Frozen prefix + newest-first unique history hits for one prefix cycle."""

    prefix: str | None = None
    matches: list[str] = field(default_factory=list)
    index: int = 0
    full_walk: bool = False

    def reset(self) -> None:
        self.prefix = None
        self.matches = []
        self.index = 0
        self.full_walk = False


_REPL_CYCLE = ReplCompleteState()
_CTRL_F = 6  # same complete path as Tab
_CTRL_N = 14  # get_completion_type() when Ctrl+N invoked complete
_CTRL_P = 16


def reset_repl_cycle() -> None:
    """Clear cycle state (call at the start of each REPL prompt)."""
    _REPL_CYCLE.reset()


def _without_prefix_stubs(lines: Sequence[str]) -> list[str]:
    """Drop fragments that are only a prefix of a longer saved prompt.

    libedit's default delimiters include ``/``, so a URL line can be saved as
    ``... (https:/`` plus the real prompt. Those two then prefix-cycle forever.
    """
    items = list(lines)
    return [
        line
        for line in items
        if not any(other != line and other.startswith(line) for other in items)
    ]


def _starts_with(line: str, prefix: str) -> bool:
    """Prefix match for history. Case-insensitive so ``c`` hits ``Can you…``."""
    return line.lower().startswith(prefix.lower())


def _line_key(line: str) -> str:
    """Collapse tabs/spaces so ``re\\tq`` and ``re   q`` are the same prompt."""
    return " ".join(line.split()).lower()


def _same_line(left: str, right: str) -> bool:
    return left == right or _line_key(left) == _line_key(right)


def _match_index(buffer: str, matches: Sequence[str]) -> int | None:
    for i, line in enumerate(matches):
        if _same_line(buffer, line):
            return i
    return None


def history_cycle_matches(buffer: str, history: Sequence[str]) -> list[str]:
    """Newest-first unique history lines that start with ``buffer`` and are longer."""
    if not buffer:
        return []
    seen: set[str] = set()
    out: list[str] = []
    lower = buffer.lower()
    for line in reversed(history):
        if not line or line in seen:
            continue
        if _starts_with(line, buffer) and line.lower() != lower:
            out.append(line)
            seen.add(line)
    return _without_prefix_stubs(out)


def _prefix_cycle_matches(buffer: str, history: Sequence[str]) -> list[str]:
    """History prefix hits, then matching session commands not already listed.

    ``re`` must still reach ``reset`` when history also has a leftover ``re\\tq``
    stub (old Tab-inserted spaces). Commands stay after history so ``q`` still
    opens on ``Question…`` rather than jumping to ``quit``.
    """
    hits = history_cycle_matches(buffer, history)
    for cmd in _command_matches(buffer):
        if cmd not in hits:
            hits.append(cmd)
    return hits


def _inferred_cycle_prefix(buffer: str, history: Sequence[str]) -> str | None:
    """Recover the typed token after libedit turns a history tab into spaces."""
    if not buffer or (" " not in buffer and "\t" not in buffer):
        return None
    key = _line_key(buffer)
    if not any(_line_key(line) == key for line in history if line):
        return None
    parts = buffer.split()
    if len(parts) < 2:
        return None
    return parts[0]


def full_history_matches(history: Sequence[str]) -> list[str]:
    """Newest-first unique history lines (empty-line Ctrl+P / Ctrl+N)."""
    seen: set[str] = set()
    out: list[str] = []
    for line in reversed(history):
        if not line or line in seen:
            continue
        out.append(line)
        seen.add(line)
    return _without_prefix_stubs(out)


def _nav_line(buffer: str) -> str:
    """Drop the trailing space readline appends after a unique complete."""
    return buffer.rstrip()


def ring_step_history(buffer: str, history: Sequence[str], *, reverse: bool = False) -> str | None:
    """Next distinct history line. Never empty — libedit ignores ``""`` completions.

    Newest-first ring: older wraps oldest → newest; newer wraps newest → oldest.
    """
    hits = full_history_matches(history)
    if not hits:
        return None
    buf = _nav_line(buffer)
    if not buf:
        return hits[-1] if reverse else hits[0]
    i = _match_index(buf, hits)
    if i is None:
        return None
    if len(hits) == 1:
        return hits[0]
    return hits[(i + (-1 if reverse else 1)) % len(hits)]


def advance_repl_cycle(
    buffer: str,
    history: Sequence[str],
    state: ReplCompleteState,
    *,
    reverse: bool = False,
    apply_commands: bool = True,
) -> str | None:
    """Prefix-cycle fill. One candidate so libedit does not dump a match list.

    ``reverse`` (Ctrl+N) walks newer matches; Tab / Ctrl+P walk older.
    Session commands apply only when ``apply_commands`` (Tab).
    """
    buffer = _nav_line(buffer)
    hist_hits = _prefix_cycle_matches(buffer, history) if buffer else []
    # Unique command with no history prefix (Tab / Ctrl+F on ``re`` → reset).
    # Ctrl+P on libedit is often reported as Tab; history hits already include
    # matching commands *after* saved prompts, so ``q`` still opens on
    # ``Question…`` rather than jumping to ``quit``.
    if apply_commands and not hist_hits:
        commands = _command_matches(buffer)
        if len(commands) == 1:
            state.reset()
            return commands[0]

    if state.matches and state.prefix is not None and _starts_with(buffer, state.prefix):
        idx = _match_index(buffer, state.matches)
        if idx is not None:
            if len(state.matches) == 1:
                return state.matches[0]
            step = -1 if reverse else 1
            state.index = (idx + step) % len(state.matches)
            return state.matches[state.index]

    if not hist_hits:
        # Empty-line Ctrl+P is a full-history ring — do not treat the filled
        # prompt as a new first-word prefix (``oldest unique prompt``).
        if not state.full_walk:
            inferred = _inferred_cycle_prefix(buffer, history)
            if inferred:
                hist_hits = _prefix_cycle_matches(inferred, history)
                idx = _match_index(buffer, hist_hits) if hist_hits else None
                if hist_hits and idx is not None:
                    state.prefix = inferred
                    state.matches = hist_hits
                    if len(hist_hits) == 1:
                        state.index = 0
                        return hist_hits[0]
                    step = -1 if reverse else 1
                    state.index = (idx + step) % len(hist_hits)
                    return hist_hits[state.index]
            state.reset()
        return None
    state.full_walk = False
    state.prefix = buffer
    state.matches = hist_hits
    state.index = 0
    return state.matches[0]


def advance_repl_nav(
    buffer: str,
    history: Sequence[str],
    state: ReplCompleteState,
    *,
    source: str,
) -> str | None:
    """Tab / Ctrl+P / Ctrl+N. ``source`` is ``tab``, ``older``, or ``newer``."""
    buffer = _nav_line(buffer)
    # libedit reports Ctrl+P as Tab. An empty complete must still walk history.
    if buffer == "" and source == "tab":
        source = "older"
    reverse = source == "newer"

    if buffer == "":
        state.full_walk = True
        state.prefix = None
        state.matches = []
        return ring_step_history("", history, reverse=reverse)

    found = advance_repl_cycle(
        buffer,
        history,
        state,
        reverse=reverse,
        apply_commands=source == "tab",
    )
    if found is not None and found != buffer:
        state.full_walk = False
        return found
    if found in REPL_COMMANDS:
        state.full_walk = False
        return found
    # Empty-line walk may continue from the filled prompt. A typed prefix
    # (``q``, ``C``) never jumps to unrelated history like ``Hi``.
    if state.full_walk:
        return ring_step_history(buffer, history, reverse=reverse)
    return found


def complete_repl(buffer: str, history: Sequence[str]) -> list[str]:
    """First-Tab candidates (fresh cycle). Used by tests and as the public matcher."""
    found = advance_repl_nav(buffer, history, ReplCompleteState(), source="tab")
    return [found] if found else []


def _readline_history_lines(readline: Any) -> list[str]:
    n = int(readline.get_current_history_length())
    return [readline.get_history_item(i) or "" for i in range(1, n + 1)]


def _readline_uses_libedit(readline: Any) -> bool:
    """CPython on this machine is often built against libedit, not GNU readline."""
    backend = str(getattr(readline, "backend", "") or "").lower()
    doc = str(getattr(readline, "__doc__", "") or "").lower()
    return backend in {"editline", "libedit"} or "libedit" in doc


def _disable_completion_append(readline: Any) -> None:
    """Stop readline from adding a space after a unique match (breaks the next Ctrl+P)."""
    setter = getattr(readline, "set_completion_append_character", None)
    if callable(setter):
        setter("")
        return
    try:
        import ctypes

        char = ctypes.c_char.in_dll(ctypes.CDLL(None), "rl_completion_append_character")
        char.value = b"\0"
    except (OSError, ValueError, AttributeError, TypeError):
        return


def _bind_tab_complete(readline: Any) -> None:
    """GNU ``tab: complete`` is a no-op on libedit — Tab then inserts spaces.

    Ctrl+F uses the same completer (user habit). That replaces forward-char.
    """
    if _readline_uses_libedit(readline):
        readline.parse_and_bind("bind ^I rl_complete")
        readline.parse_and_bind("bind ^F rl_complete")
    else:
        readline.parse_and_bind("tab: complete")
        readline.parse_and_bind(r'"\C-f": complete')


def _bind_cycle_nav_keys(readline: Any) -> None:
    """Ctrl+P / Ctrl+N always call our completer (empty = history, else prefix)."""
    if _readline_uses_libedit(readline):
        readline.parse_and_bind("bind ^P rl_complete")
        readline.parse_and_bind("bind ^N rl_complete")
    else:
        readline.parse_and_bind(r'"\C-p": complete')
        readline.parse_and_bind(r'"\C-n": complete')


def _bind_clear_line(readline: Any) -> None:
    """Ctrl+U wipes the whole prompt. Native kill — not an empty completer.

    libedit's default ^U is kill-to-end (a no-op at the usual caret). GNU's
    default is kill-to-start (mid-line leftovers). ``em-kill-line`` /
    ``kill-whole-line`` clear everything; Ctrl+Y yanks it back.
    """
    if _readline_uses_libedit(readline):
        readline.parse_and_bind("bind ^U em-kill-line")
    else:
        readline.parse_and_bind(r'"\C-u": kill-whole-line')


def _completion_source(readline: Any) -> str:
    """Map the invoking key to ``tab``, ``older`` (Ctrl+P), or ``newer`` (Ctrl+N)."""
    getter = getattr(readline, "get_completion_type", None)
    if not callable(getter):
        return "tab"
    try:
        kind = int(getter())
    except (TypeError, ValueError):
        return "tab"
    if kind == _CTRL_N:
        return "newer"
    if kind == _CTRL_P:
        return "older"
    # Tab (9) and Ctrl+F (6) share the command-first complete path.
    return "tab"


def _completion_wants_reverse(readline: Any) -> bool:
    return _completion_source(readline) == "newer"


def _bind_repl_completer(readline: Any) -> None:
    """Fill a unique match. Do not print the stock match list (it wrecks the prompt)."""
    matches: list[str] = []

    def completer(text: str, state: int) -> str | None:
        nonlocal matches
        if state == 0:
            try:
                buf = str(readline.get_line_buffer())
            except (AttributeError, TypeError):
                buf = text
            source = _completion_source(readline)
            hist = _readline_history_lines(readline)
            line = buf if buf != "" else text
            found = advance_repl_nav(line, hist, _REPL_CYCLE, source=source)
            matches = [found] if found is not None else []
        if 0 <= state < len(matches):
            return matches[state]
        return None

    def _hide_match_list(_substitution: str, _found: list[str], _longest: int) -> None:
        return None

    readline.set_completer(completer)
    # libedit treats set_completer_delims("") as "reset to default", and the
    # default includes ``/``, so URL prompts complete as ``https:/`` stubs.
    readline.set_completer_delims("\n")
    _disable_completion_append(readline)
    hook = getattr(readline, "set_completion_display_matches_hook", None)
    if callable(hook):
        hook(_hide_match_list)
    _bind_tab_complete(readline)
    _bind_cycle_nav_keys(readline)
    _bind_clear_line(readline)


def attach_repl_readline(path: Path | None = None) -> Path | None:
    """Hook GNU readline for Up/Down and Tab when stdin is a TTY."""
    if not sys.stdin.isatty():
        return None
    try:
        import readline
    except ImportError:
        return None
    hist = path if path is not None else default_history_path()
    for line in load_history_lines(hist):
        readline.add_history(line)
    readline.set_history_length(CLI_HISTORY_MAX)
    reset_repl_cycle()
    _bind_repl_completer(readline)
    return hist


def readline_prompt() -> str:
    """Same cyan 'you › ' as Typer, with ANSI wrapped so readline ignores their width.

    Without \\001/\\002, Up-arrow overwrites the prompt (you see leftover 'm' + history).
    """
    # Last counted character must come after reset. If the trailing space stays
    # inside the cyan region, readline paints the typed line cyan too.
    return "\001\033[1;36m\002you\001\033[0m\033[36m\002 ›\001\033[0m\002 "


def persist_repl_readline(path: Path | None) -> None:
    if path is None:
        return
    try:
        import readline
    except ImportError:
        return
    lines = _readline_history_lines(readline)
    try:
        write_history_lines(path, lines)
    except OSError:
        pass


class Provider(StrEnum):
    ollama = "ollama"
    grok = "grok"


def _build_orchestrator() -> Orchestrator:
    from swyngora_ai.graph.orchestrator import build_orchestrator

    return build_orchestrator()


def _terminal_columns() -> int:
    """Live TTY columns via ioctl. Ignore COLUMNS — shells often leave it stale on zoom."""
    fds: list[int] = []
    for stream in (sys.__stdout__, sys.stdout, sys.__stderr__, sys.stderr):
        fileno = getattr(stream, "fileno", None)
        if fileno is None:
            continue
        try:
            fds.append(int(fileno()))
        except (OSError, ValueError):
            continue
    tty_fd = -1
    try:
        tty_fd = os.open("/dev/tty", os.O_RDONLY)
        fds.append(tty_fd)
    except OSError:
        pass
    try:
        for fd in fds:
            try:
                return max(2, os.get_terminal_size(fd).columns)
            except OSError:
                continue
    finally:
        if tty_fd >= 0:
            os.close(tty_fd)
    return 80


def _rule_bar(width: int, title: str = "") -> str:
    """Exactly ``width`` cells: dashes, or a centered heading like ``── details ──``."""
    if not title:
        return "─" * width
    label = f" {title} "
    if len(label) >= width:
        return label[:width]
    fill = width - len(label)
    left = fill // 2
    return f"{'─' * left}{label}{'─' * (fill - left)}"


def _full_width_rule(title: str = "") -> None:
    """One left-to-right line written to the TTY, not through Rich wrap/crop."""
    width = _terminal_columns()
    bar = _rule_bar(width, title)
    out = getattr(_console, "file", None) or sys.stdout
    if getattr(out, "isatty", lambda: False)():
        out.write(f"\033[2m{bar}\033[0m\n")
    else:
        out.write(f"{bar}\n")
    flush = getattr(out, "flush", None)
    if flush is not None:
        flush()


def _print_result(result: ChatResult, activity: str | Sequence[str] = "") -> None:
    _console.print()
    _console.print(Markdown(result.reply))
    _console.print()
    _print_details(result, activity)


_DETAILS_LABEL_WIDTH = 8  # "Activity" and "thinking" — values share one left column


def _activity_detail_rows(activity: str | Sequence[str]) -> list[str]:
    if isinstance(activity, str):
        return [activity] if activity else []
    return [row for row in activity if row]


def _print_details(result: ChatResult, activity: str | Sequence[str] = "") -> None:
    extras = Table.grid(padding=(0, 3))
    extras.add_column(
        style="bold dim",
        no_wrap=True,
        min_width=_DETAILS_LABEL_WIDTH,
        max_width=_DETAILS_LABEL_WIDTH,
    )
    extras.add_column(overflow="fold")

    def _add(label: str, value: str) -> None:
        extras.add_row(label, value)

    rows = _activity_detail_rows(activity)
    if rows:
        _add("Activity", escape(rows[0]))
        for row in rows[1:]:
            _add("", escape(row))
        _add("", "")
    first = True
    for step in result.thinking:
        _add("thinking" if first else "", escape(step))
        first = False
    first = True
    for tool in result.tools:
        _add("tool" if first else "", escape(tool))
        first = False
    first = True
    for ref in result.references:
        title = escape(ref.get("title") or ref.get("url") or "source")
        url = ref.get("url") or ""
        cell = f"[link={url}]{title}[/link]\n[dim]{escape(url)}[/]" if url else title
        _add("source" if first else "", cell)
        first = False
    _add("session", escape(result.session_id))

    _full_width_rule("details")
    _console.print(extras)
    _console.print()


def _print_error(exc: BaseException) -> None:
    _err.print(f"[red]error:[/] {exc}")


_STATUS_TYPES = frozenset({"status", "tool", "thinking", "tool_error"})
# Count the start event. Prefetch only emits "tool" (no tool_result on success);
# orchestrator also emits tool_result later — do not count both.
_TOOL_TICK_TYPES = frozenset({"tool", "tool_retry"})
_LLM_TICK_TYPES = frozenset({"llm"})
_USAGE_TICK_TYPES = frozenset({"usage"})
_STATUS_MAX = 88


def _status_text(ev: dict[str, Any]) -> str | None:
    if (ev.get("type") or "") not in _STATUS_TYPES:
        return None
    text = (ev.get("text") or "").strip()
    if not text:
        return None
    if len(text) > _STATUS_MAX:
        return text[: _STATUS_MAX - 1] + "…"
    return text


def _tools_label(count: int) -> str:
    return "1 tool call" if count == 1 else f"{count} tool calls"


def tool_activity_name(text: str) -> str:
    """Stable tool name from a type=tool emit (`prefetch → get_ticker(...)` → get_ticker)."""
    base = (text or "").strip().split("(", 1)[0].strip()
    if "→" in base:
        base = base.rsplit("→", 1)[-1].strip()
    return base or "unknown"


def _llm_label(count: int) -> str:
    return "1 llm call" if count == 1 else f"{count} llm calls"


def _activity_model_id(settings: Any) -> str:
    if settings is None:
        return ""
    if getattr(settings, "llm_provider", "") == "grok":
        return str(getattr(settings, "grok_model", "") or "")
    return str(getattr(settings, "ollama_model", "") or "")


def _activity_temperature(settings: Any) -> float | None:
    if settings is None:
        return None
    raw = getattr(settings, "temperature", None)
    if raw is None:
        return None
    return float(raw)


def _format_temp(value: float) -> str:
    return f"{value:g}"


def _format_tokens(n: int) -> str:
    return f"{n:,}"


def _format_elapsed(seconds: float) -> str:
    total = max(0, int(seconds))
    return f"{total // 60}:{total % 60:02d}"


def activity_summary(state: ActivityBarState) -> str:
    """One-line live bar: total tool starts plus per-model llm counts."""
    parts = [_tools_label(sum(state.tools_by_name.values()))]
    if state.llm_by_model:
        parts.extend(f"{_llm_label(n)} {name}" for name, n in state.llm_by_model.items())
    elif state.fallback_model:
        parts.append(f"{_llm_label(0)} {state.fallback_model}")
    else:
        parts.append(_llm_label(0))
    return " · ".join(parts)


def activity_headline(state: ActivityBarState, elapsed: float) -> str:
    """First details Activity row: llm (and 0 tool calls) plus elapsed."""
    parts: list[str] = []
    if not state.tools_by_name:
        parts.append(_tools_label(0))
    if state.llm_by_model:
        parts.extend(f"{_llm_label(n)} {name}" for name, n in state.llm_by_model.items())
    elif state.fallback_model:
        parts.append(f"{_llm_label(0)} {state.fallback_model}")
    else:
        parts.append(_llm_label(0))
    if state.temperature is not None:
        parts.append(f"temperature {_format_temp(state.temperature)}")
    parts.append(f"{_format_elapsed(elapsed)} elapsed")
    return " · ".join(parts)


def _tool_row(state: ActivityBarState, name: str, count: int) -> str:
    line = f"{_tools_label(count)} {name}"
    if state.tool_retries_by_name.get(name, 0):
        return f"{line} (includes retries)"
    return line


def activity_token_rows(state: ActivityBarState) -> list[str]:
    items = list(state.tokens_by_model.items())
    if not items:
        return ["0 in · 0 out · 0 tokens"]
    total_in = sum(pair[0] for pair in state.tokens_by_model.values())
    total_out = sum(pair[1] for pair in state.tokens_by_model.values())
    total = total_in + total_out
    if len(items) == 1:
        inn, out = items[0][1]
        return [
            f"{_format_tokens(inn)} in · {_format_tokens(out)} out · {_format_tokens(inn + out)} tokens"
        ]
    rows = [
        f"{name}  {_format_tokens(inn)} in · {_format_tokens(out)} out"
        for name, (inn, out) in items
    ]
    rows.append(f"{_format_tokens(total)} tokens")
    return rows


def activity_rows(state: ActivityBarState, elapsed: float) -> list[str]:
    rows = [activity_headline(state, elapsed)]
    rows.extend(activity_token_rows(state))
    rows.extend(_tool_row(state, name, n) for name, n in state.tools_by_name.items())
    return rows


@dataclass
class ActivityBarState:
    """Live CLI activity line: status text plus per-name tool, llm, and token counts."""

    description: str = "Planning…"
    tools_by_name: dict[str, int] = field(default_factory=dict)
    tool_retries_by_name: dict[str, int] = field(default_factory=dict)
    llm_by_model: dict[str, int] = field(default_factory=dict)
    tokens_by_model: dict[str, list[int]] = field(default_factory=dict)
    fallback_model: str = ""
    temperature: float | None = None


def apply_activity_event(state: ActivityBarState, ev: dict[str, Any]) -> bool:
    """Update description and counters. True if the bar should refresh."""
    changed = False
    kind = ev.get("type") or ""
    if kind in _TOOL_TICK_TYPES:
        name = tool_activity_name(str(ev.get("text") or ""))
        state.tools_by_name[name] = state.tools_by_name.get(name, 0) + 1
        if kind == "tool_retry":
            state.tool_retries_by_name[name] = state.tool_retries_by_name.get(name, 0) + 1
        changed = True
    if kind in _LLM_TICK_TYPES:
        label = (ev.get("text") or "").strip() or state.fallback_model or "unknown"
        state.llm_by_model[label] = state.llm_by_model.get(label, 0) + 1
        changed = True
    if kind in _USAGE_TICK_TYPES:
        parsed = parse_usage_emit(str(ev.get("text") or ""))
        if parsed is not None:
            label, inn, out = parsed
            cur = state.tokens_by_model.get(label)
            if cur is None:
                state.tokens_by_model[label] = [inn, out]
            else:
                cur[0] += inn
                cur[1] += out
            changed = True
    text = _status_text(ev)
    if text is not None:
        state.description = text
        changed = True
    return changed


def activity_line(state: ActivityBarState, elapsed: float) -> str:
    return activity_headline(state, elapsed)


def _chat(orch: Any, message: str, session_id: str) -> tuple[ChatResult, list[str]]:
    if not _console.is_terminal:
        return orch.chat(message, session_id=session_id), []

    started = time.monotonic()
    settings = getattr(orch, "settings", None)
    state = ActivityBarState(
        fallback_model=_activity_model_id(settings),
        temperature=_activity_temperature(settings),
    )
    try:
        with Progress(
            SpinnerColumn(),
            TextColumn("[cyan]{task.description}[/]"),
            TextColumn("[dim]{task.fields[summary]}[/]"),
            TimeElapsedColumn(),
            console=_console,
            transient=True,
        ) as progress:
            task_id = progress.add_task(
                escape(state.description), total=None, summary=activity_summary(state)
            )

            def on_event(ev: dict[str, Any]) -> None:
                if not apply_activity_event(state, ev):
                    return
                progress.update(
                    task_id,
                    description=escape(state.description),
                    summary=activity_summary(state),
                )

            result = orch.chat(message, session_id=session_id, on_event=on_event)
    except Exception:
        from swyngora_ai.graph.orchestrator import ChatResult as _ChatResult

        _print_details(
            _ChatResult(reply="", session_id=session_id),
            activity_rows(state, time.monotonic() - started),
        )
        raise
    return result, activity_rows(state, time.monotonic() - started)


def _one_shot(orch: Orchestrator, message: str, session_id: str) -> int:
    try:
        result, activity = _chat(orch, message, session_id)
        _print_result(result, activity)
    except Exception as e:  # noqa: BLE001
        _print_error(e)
        return 1
    return 0


def _print_banner() -> None:
    _console.print()
    _full_width_rule()
    _console.print(Align.center("[bold green]Swyngora AI[/]"))
    _console.print()
    _console.print(
        Align.center("[bold yellow]Informational analysis only — not financial advice.[/]")
    )
    _console.print()
    cmds = Table.grid(padding=(0, 3))
    cmds.add_column(no_wrap=True)
    cmds.add_column()
    cmds.add_row("[bold]exit[/] / [bold]quit[/] / [bold]q[/]", "leave the session")
    cmds.add_row("[bold]reset[/]", "clear conversation memory")
    cmds.add_row("[bold]↑[/] / [bold]↓[/]", "previous prompts")
    cmds.add_row("[bold]Tab[/] / [bold]Ctrl+F[/]", "commands; cycle previous prompts")
    cmds.add_row("[bold]Ctrl+P[/]", "older prompt matching typed text; all history if empty")
    cmds.add_row("[bold]Ctrl+N[/]", "newer prompt matching typed text; all history if empty")
    cmds.add_row("[bold]Ctrl+U[/]", "clear the line (Ctrl+Y restores it)")
    _console.print(Align.center(cmds))
    _console.print()


def _repl(orch: Orchestrator, session_id: str) -> int:
    _print_banner()
    hist = attach_repl_readline()
    you = typer.style("you", fg=typer.colors.CYAN, bold=True)
    suffix = typer.style(" › ", fg=typer.colors.CYAN)
    try:
        while True:
            try:
                if hist is not None:
                    reset_repl_cycle()
                    line = input(readline_prompt()).strip()
                else:
                    line = typer.prompt(you, prompt_suffix=suffix).strip()
            except (typer.Abort, EOFError, KeyboardInterrupt):
                _console.print()
                return 0
            if not line:
                continue
            if line.lower() in {"exit", "quit", "q"}:
                return 0
            if line.lower() == "reset":
                from swyngora_ai.config import get_settings

                orch.reset(session_id, client_id=get_settings().default_client_id)
                _console.print("[dim]session cleared[/]")
                continue
            try:
                result, activity = _chat(orch, line, session_id)
                _print_result(result, activity)
            except Exception as e:  # noqa: BLE001
                _print_error(e)
    finally:
        persist_repl_readline(hist)


@app.command()
def run(
    message: Annotated[
        str | None,
        typer.Argument(help="One-shot question (omit for an interactive REPL)"),
    ] = None,
    session: Annotated[
        str,
        typer.Option("--session", help="Session id for in-process memory"),
    ] = "cli",
    provider: Annotated[
        Provider | None,
        typer.Option("--provider", help="Override AI_LLM_PROVIDER for this process"),
    ] = None,
) -> None:
    """Swyngora multi-agent AI assistant."""
    from swyngora_ai.config import load_env_files

    load_env_files()
    if provider is not None:
        os.environ["AI_LLM_PROVIDER"] = provider.value
    orch = _build_orchestrator()
    code = _one_shot(orch, message, session) if message else _repl(orch, session)
    raise typer.Exit(code)


def main(argv: list[str] | None = None) -> None:
    app(args=argv, prog_name="swyngora-ai")


if __name__ == "__main__":
    main()
