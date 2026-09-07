"""CLI entrypoint: swyngora-ai (Typer)."""

from __future__ import annotations

import os
import sys
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
from rich.table import Table

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

    def reset(self) -> None:
        self.prefix = None
        self.matches = []
        self.index = 0


_REPL_CYCLE = ReplCompleteState()
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


def history_cycle_matches(buffer: str, history: Sequence[str]) -> list[str]:
    """Newest-first unique history lines that start with ``buffer`` and are longer."""
    if not buffer:
        return []
    seen: set[str] = set()
    out: list[str] = []
    for line in reversed(history):
        if not line or line in seen:
            continue
        if line.startswith(buffer) and line != buffer:
            out.append(line)
            seen.add(line)
    return _without_prefix_stubs(out)


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
    if buf not in hits:
        return None
    if len(hits) == 1:
        return hits[0]
    i = hits.index(buf)
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
    if apply_commands:
        commands = _command_matches(buffer)
        if len(commands) == 1:
            state.reset()
            return commands[0]

    if (
        state.matches
        and 0 <= state.index < len(state.matches)
        and state.prefix is not None
        and buffer == state.matches[state.index]
        and buffer.startswith(state.prefix)
    ):
        if len(state.matches) == 1:
            return buffer
        step = -1 if reverse else 1
        state.index = (state.index + step) % len(state.matches)
        return state.matches[state.index]

    state.prefix = buffer
    state.matches = history_cycle_matches(buffer, history)
    state.index = 0
    if not state.matches:
        state.reset()
        return None
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
        return ring_step_history("", history, reverse=reverse)

    found = advance_repl_cycle(
        buffer,
        history,
        state,
        reverse=reverse,
        apply_commands=source == "tab",
    )
    if found is not None and found != buffer:
        return found
    if found in REPL_COMMANDS:
        return found
    # Unique / already-filled history line: ring to a *different* prompt.
    # libedit ignores "" and ignores a complete equal to the current line.
    return ring_step_history(buffer, history, reverse=reverse)


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
    """GNU ``tab: complete`` is a no-op on libedit — Tab then inserts spaces."""
    if _readline_uses_libedit(readline):
        readline.parse_and_bind("bind ^I rl_complete")
    else:
        readline.parse_and_bind("tab: complete")


def _bind_cycle_nav_keys(readline: Any) -> None:
    """Ctrl+P / Ctrl+N always call our completer (empty = history, else prefix)."""
    if _readline_uses_libedit(readline):
        readline.parse_and_bind("bind ^P rl_complete")
        readline.parse_and_bind("bind ^N rl_complete")
    else:
        readline.parse_and_bind(r'"\C-p": complete')
        readline.parse_and_bind(r'"\C-n": complete')


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
            cur = _nav_line(line)
            if found is not None and found == cur:
                found = ring_step_history(cur, hist, reverse=source == "newer")
                if found == cur:
                    found = None
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


def _print_result(result: ChatResult) -> None:
    _console.print()
    _console.print(Markdown(result.reply))
    _console.print()
    _print_details(result)


def _print_details(result: ChatResult) -> None:
    extras = Table.grid(padding=(0, 3))
    extras.add_column(style="bold dim", no_wrap=True)
    extras.add_column(overflow="fold")

    def _add(label: str, value: str) -> None:
        extras.add_row(label, value)

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

    _console.rule("[dim]details[/]", style="dim")
    _console.print(extras)
    _console.print()


def _print_error(exc: BaseException) -> None:
    _err.print(f"[red]error:[/] {exc}")


_STATUS_TYPES = frozenset({"status", "tool", "thinking", "tool_error"})
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


def _chat(orch: Orchestrator, message: str, session_id: str) -> ChatResult:
    if not _console.is_terminal:
        return orch.chat(message, session_id=session_id)

    with _console.status("[dim]Planning…[/]", spinner="dots") as status:

        def on_event(ev: dict[str, Any]) -> None:
            text = _status_text(ev)
            if text:
                status.update(f"[cyan]{escape(text)}[/]")

        return orch.chat(message, session_id=session_id, on_event=on_event)


def _one_shot(orch: Orchestrator, message: str, session_id: str) -> int:
    try:
        _print_result(_chat(orch, message, session_id))
    except Exception as e:  # noqa: BLE001
        _print_error(e)
        return 1
    return 0


def _print_banner() -> None:
    _console.print()
    _console.rule("[bold green]Swyngora AI[/]")
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
    cmds.add_row("[bold]Tab[/]", "commands; cycle previous prompts")
    cmds.add_row("[bold]Ctrl+P[/]", "older prompt matching typed text; all history if empty")
    cmds.add_row("[bold]Ctrl+N[/]", "newer prompt matching typed text; all history if empty")
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
                _print_result(_chat(orch, line, session_id))
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
