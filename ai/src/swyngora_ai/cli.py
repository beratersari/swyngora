"""CLI entrypoint: swyngora-ai (Typer)."""

from __future__ import annotations

import os
from enum import StrEnum
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
    _console.print(Align.center(cmds))
    _console.print()


def _repl(orch: Orchestrator, session_id: str) -> int:
    _print_banner()
    you = typer.style("you", fg=typer.colors.CYAN, bold=True)
    suffix = typer.style(" › ", fg=typer.colors.CYAN)
    while True:
        try:
            line = typer.prompt(you, prompt_suffix=suffix).strip()
        except (typer.Abort, EOFError, KeyboardInterrupt):
            _console.print()
            return 0
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
    if provider is not None:
        os.environ["AI_LLM_PROVIDER"] = provider.value
    orch = _build_orchestrator()
    code = _one_shot(orch, message, session) if message else _repl(orch, session)
    raise typer.Exit(code)


def main(argv: list[str] | None = None) -> None:
    app(args=argv, prog_name="swyngora-ai")


if __name__ == "__main__":
    main()
