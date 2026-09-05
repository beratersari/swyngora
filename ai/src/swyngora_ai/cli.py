"""CLI entrypoint: swyngora-ai (Typer)."""

from __future__ import annotations

import os
import sys
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


def attach_repl_readline(path: Path | None = None) -> Path | None:
    """Hook GNU readline for Up/Down when stdin is a TTY. Returns the history path."""
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
    n = readline.get_current_history_length()
    lines = [readline.get_history_item(i) or "" for i in range(1, n + 1)]
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
