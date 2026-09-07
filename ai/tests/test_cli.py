"""CLI tests (Typer; no live LLM)."""

from __future__ import annotations

import os
import sys

from typer.testing import CliRunner

from swyngora_ai.cli import (
    _CTRL_N,
    _CTRL_P,
    ReplCompleteState,
    _bind_cycle_nav_keys,
    _bind_tab_complete,
    _completion_source,
    _completion_wants_reverse,
    _readline_uses_libedit,
    _status_text,
    advance_repl_cycle,
    advance_repl_nav,
    app,
    attach_repl_readline,
    complete_repl,
    full_history_matches,
    history_cycle_matches,
    load_history_lines,
    persist_repl_readline,
    readline_prompt,
    ring_step_history,
    write_history_lines,
)
from swyngora_ai.graph.orchestrator import ChatResult


class _FakeOrch:
    def __init__(self) -> None:
        self.chats: list[tuple[str, str]] = []
        self.resets: list[str] = []

    def chat(self, message: str, session_id: str = "default", on_event=None) -> ChatResult:
        self.chats.append((message, session_id))
        return ChatResult(
            reply=f"echo:{message}",
            session_id=session_id,
            tools=["market_agent(BTC)"],
            thinking=["checking RSI"],
            references=[{"title": "Binance", "url": "https://example.com/btc", "source": "web"}],
        )

    def reset(self, session_id: str, client_id: str = "") -> None:
        self.resets.append(session_id)


class _BoomOrch(_FakeOrch):
    def chat(self, message: str, session_id: str = "default", on_event=None) -> ChatResult:
        raise RuntimeError("backend down")


runner = CliRunner()


def test_status_text_filters_noise_and_truncates() -> None:
    assert _status_text({"type": "final", "text": "done"}) is None
    assert _status_text({"type": "ping", "text": "x"}) is None
    assert _status_text({"type": "status", "text": "Planning…"}) == "Planning…"
    long = "x" * 120
    out = _status_text({"type": "tool", "text": long})
    assert out is not None
    assert out.endswith("…")
    assert len(out) == 88


def test_help() -> None:
    result = runner.invoke(app, ["--help"])
    assert result.exit_code == 0
    assert "One-shot question" in result.stdout
    assert "--session" in result.stdout
    assert "--provider" in result.stdout


def test_one_shot_prints_reply_not_dataclass(monkeypatch) -> None:
    fake = _FakeOrch()
    monkeypatch.setattr("swyngora_ai.cli._build_orchestrator", lambda: fake)
    result = runner.invoke(app, ["What is BTC RSI?", "--session", "desk"])
    assert result.exit_code == 0
    assert "echo:What is BTC RSI?" in result.stdout
    assert "ChatResult" not in result.stdout
    assert "details" in result.stdout
    assert "checking RSI" in result.stdout
    assert "market_agent(BTC)" in result.stdout
    assert "Binance" in result.stdout
    assert "https://example.com/btc" in result.stdout
    assert "desk" in result.stdout
    assert fake.chats == [("What is BTC RSI?", "desk")]


def test_cli_loads_env_files(monkeypatch) -> None:
    called: list[int] = []
    fake = _FakeOrch()
    monkeypatch.setattr("swyngora_ai.cli._build_orchestrator", lambda: fake)

    def _mark() -> None:
        called.append(1)

    monkeypatch.setattr("swyngora_ai.config.load_env_files", _mark)
    result = runner.invoke(app, ["hi"])
    assert result.exit_code == 0
    assert called == [1]


def test_provider_sets_env(monkeypatch) -> None:
    fake = _FakeOrch()
    monkeypatch.setattr("swyngora_ai.cli._build_orchestrator", lambda: fake)
    monkeypatch.delenv("AI_LLM_PROVIDER", raising=False)
    result = runner.invoke(app, ["--provider", "grok", "hi"])
    assert result.exit_code == 0
    assert os.environ["AI_LLM_PROVIDER"] == "grok"
    assert fake.chats == [("hi", "cli")]


def test_one_shot_error_is_caught(monkeypatch) -> None:
    monkeypatch.setattr("swyngora_ai.cli._build_orchestrator", _BoomOrch)
    result = runner.invoke(app, ["ping"])
    assert result.exit_code == 1
    assert "error: backend down" in result.stderr
    assert "Traceback" not in result.stderr


def test_repl_reset_and_exit(monkeypatch) -> None:
    fake = _FakeOrch()
    monkeypatch.setattr("swyngora_ai.cli._build_orchestrator", lambda: fake)
    result = runner.invoke(app, ["--session", "s1"], input="reset\nhello\nexit\n")
    assert result.exit_code == 0
    assert fake.resets == ["s1"]
    assert fake.chats == [("hello", "s1")]
    assert "echo:hello" in result.stdout
    assert "session cleared" in result.stdout
    assert "not financial advice" in result.stdout.lower()
    assert "leave the session" in result.stdout
    assert "clear conversation memory" in result.stdout
    assert "previous prompts" in result.stdout
    assert "cycle previous prompts" in result.stdout
    assert "older prompt matching typed text" in result.stdout
    assert "newer prompt matching typed text" in result.stdout
    assert "all history if empty" in result.stdout


def test_load_history_lines_missing(tmp_path) -> None:
    assert load_history_lines(tmp_path / "missing") == []


def test_write_history_lines_creates_caps_and_skips_empty(tmp_path) -> None:
    path = tmp_path / "nested" / "cli_history"
    write_history_lines(
        path, ["", "  first  ", "second"] + [f"n{i}" for i in range(10)], max_lines=3
    )
    assert path.is_file()
    assert load_history_lines(path) == ["n7", "n8", "n9"]


def test_attach_repl_readline_skips_when_not_a_tty(monkeypatch, tmp_path) -> None:
    monkeypatch.setattr("swyngora_ai.cli.sys.stdin.isatty", lambda: False)
    assert attach_repl_readline(tmp_path / "cli_history") is None
    assert not (tmp_path / "cli_history").exists()


def test_persist_repl_readline_none_is_noop() -> None:
    persist_repl_readline(None)


def test_readline_prompt_hides_ansi_from_width() -> None:
    prompt = readline_prompt()
    assert "you" in prompt
    assert "›" in prompt
    assert "\001" in prompt
    assert "\002" in prompt
    assert prompt.startswith("\001")
    assert prompt.endswith(" ")
    assert prompt.rstrip().endswith("\002")


def test_complete_repl_commands_case_insensitive() -> None:
    assert complete_repl("re", []) == ["reset"]
    assert complete_repl("Re", []) == ["reset"]
    assert complete_repl("q", []) == ["quit"]
    assert complete_repl("quit", []) == ["quit"]
    assert complete_repl("", ["What is BTC RSI"]) == ["What is BTC RSI"]


def test_complete_repl_history_prefix_after_typed_text() -> None:
    hist = ["What is BTC RSI", "What is ETH funding", "reset", "What is BTC RSI"]
    assert complete_repl("What is BTC", hist) == ["What is BTC RSI"]
    assert complete_repl("What is", hist) == ["What is BTC RSI"]
    assert complete_repl("re", hist) == ["reset"]
    assert complete_repl("re", ["review the book", "reset later"]) == ["reset"]


def test_complete_repl_ambiguous_history_uses_most_recent() -> None:
    hist = [
        "What is BTC price and RSI on binance?",
        "What is BTC price in India?",
        "How is ETH",
        "Hi",
    ]
    assert complete_repl("What is BTC price", hist) == ["What is BTC price in India?"]
    assert complete_repl("H", hist) == ["Hi"]
    assert complete_repl("H", ["Hi", "How is ETH"]) == ["How is ETH"]


def test_history_cycle_matches_newest_first_and_dedup() -> None:
    hist = ["Hi", "How is ETH", "Hi"]
    assert history_cycle_matches("H", hist) == ["Hi", "How is ETH"]
    assert history_cycle_matches("", hist) == []


def test_advance_repl_cycle_walks_newest_to_oldest_and_wraps() -> None:
    hist = ["How is ETH", "Hello", "Hi"]
    state = ReplCompleteState()
    assert advance_repl_cycle("H", hist, state) == "Hi"
    assert advance_repl_cycle("Hi", hist, state) == "Hello"
    assert advance_repl_cycle("Hello", hist, state) == "How is ETH"
    assert advance_repl_cycle("How is ETH", hist, state) == "Hi"


def test_advance_repl_cycle_reverse_walks_newer() -> None:
    hist = ["How is ETH", "Hello", "Hi"]
    state = ReplCompleteState()
    assert advance_repl_cycle("H", hist, state) == "Hi"
    assert advance_repl_cycle("Hi", hist, state) == "Hello"
    assert advance_repl_cycle("Hello", hist, state, reverse=True) == "Hi"
    assert advance_repl_cycle("Hi", hist, state, reverse=True) == "How is ETH"


def test_advance_repl_cycle_resets_when_the_line_is_edited() -> None:
    hist = ["How is ETH", "Hi"]
    state = ReplCompleteState()
    assert advance_repl_cycle("H", hist, state) == "Hi"
    assert advance_repl_cycle("Ho", hist, state) == "How is ETH"


def test_advance_repl_cycle_command_wins_and_clears_history_cycle() -> None:
    hist = ["review the book"]
    state = ReplCompleteState()
    assert advance_repl_cycle("H", ["Hi"], state) == "Hi"
    assert advance_repl_cycle("re", hist, state) == "reset"
    assert state.matches == []


def test_ctrl_p_on_command_prefix_uses_history_not_reset() -> None:
    hist = ["review the book"]
    state = ReplCompleteState()
    assert advance_repl_nav("re", hist, state, source="older") == "review the book"
    assert advance_repl_nav("re", hist, ReplCompleteState(), source="tab") == "reset"


def test_empty_ctrl_p_walks_full_history_as_a_ring() -> None:
    hist = ["oldest", "middle", "newest"]
    state = ReplCompleteState()
    assert advance_repl_nav("", hist, state, source="tab") == "newest"
    assert advance_repl_nav("newest", hist, state, source="older") == "middle"
    assert advance_repl_nav("middle", hist, state, source="older") == "oldest"
    assert advance_repl_nav("oldest", hist, state, source="older") == "newest"
    assert advance_repl_nav("newest", hist, state, source="newer") == "oldest"
    assert ring_step_history("", hist, reverse=True) == "oldest"


def test_full_history_matches_newest_first() -> None:
    assert full_history_matches(["a", "b", "a"]) == ["a", "b"]


def test_url_prefix_stub_is_not_a_cycle_stop() -> None:
    stub = "Can you explain these docs (https:/"
    full = (
        "Can you explain these docs "
        "(https://docs.langchain.com/oss/python/langchain/middleware/built-in) "
        "to me clearly"
    )
    hist = [full, "Hi", stub]
    assert full_history_matches(hist) == ["Hi", full]
    assert history_cycle_matches("Can you explain these docs (https:/", hist) == [full]
    state = ReplCompleteState()
    assert advance_repl_nav(stub, hist, state, source="older") == full
    assert advance_repl_nav(full, hist, state, source="older") == "Hi"


def test_trailing_space_still_walks_full_history() -> None:
    hist = ["oldest", "newest"]
    state = ReplCompleteState()
    assert advance_repl_nav("", hist, state, source="older") == "newest"
    assert advance_repl_nav("newest ", hist, state, source="older") == "oldest"
    assert advance_repl_nav("oldest ", hist, state, source="older") == "newest"
    assert advance_repl_nav("oldest ", hist, state, source="newer") == "newest"


def test_exact_history_line_ctrl_p_leaves_the_oldest_item() -> None:
    hist = ["oldest unique prompt", "newest"]
    state = ReplCompleteState()
    assert advance_repl_nav("oldest unique prompt", hist, state, source="older") == "newest"


def test_trailing_space_still_cycles_prefix() -> None:
    hist = ["How is ETH", "Hi"]
    state = ReplCompleteState()
    assert advance_repl_nav("H", hist, state, source="older") == "Hi"
    assert advance_repl_nav("Hi ", hist, state, source="older") == "How is ETH"


def test_complete_repl_no_files_after_a_space() -> None:
    hist = ["What is BTC RSI on binance"]
    assert complete_repl("What is BTC RSI on ", hist) == ["What is BTC RSI on binance"]
    assert complete_repl("What is BTC RSI on coinbase", hist) == []
    assert complete_repl("reset ", hist) == ["reset"]
    assert complete_repl("foo bar", hist) == []


def test_attach_repl_readline_binds_tab_completer(monkeypatch, tmp_path) -> None:
    class FakeReadline:
        def __init__(self) -> None:
            self.history: list[str] = []
            self.completer = None
            self.delims = " \t\n"
            self.binds: list[str] = []
            self.buffer = ""
            self.display_hook = None
            self.completion_type = 9

        def add_history(self, line: str) -> None:
            self.history.append(line)

        def set_history_length(self, n: int) -> None:
            return None

        def get_current_history_length(self) -> int:
            return len(self.history)

        def get_history_item(self, i: int) -> str:
            return self.history[i - 1]

        def get_line_buffer(self) -> str:
            return self.buffer

        def set_completer(self, fn) -> None:
            self.completer = fn

        def set_completer_delims(self, d: str) -> None:
            self.delims = d

        def parse_and_bind(self, cmd: str) -> None:
            self.binds.append(cmd)

        def set_completion_display_matches_hook(self, fn) -> None:
            self.display_hook = fn

        def get_completion_type(self) -> int:
            return self.completion_type

    fake = FakeReadline()
    path = tmp_path / "cli_history"
    path.write_text("How is ETH\nWhat is BTC RSI\nHi\n", encoding="utf-8")
    monkeypatch.setattr("swyngora_ai.cli.sys.stdin.isatty", lambda: True)
    monkeypatch.setitem(sys.modules, "readline", fake)

    assert attach_repl_readline(path) == path
    assert fake.delims == "\n"
    assert fake.binds[:3] == [
        "tab: complete",
        r'"\C-p": complete',
        r'"\C-n": complete',
    ]
    assert fake.completer is not None
    assert fake.display_hook is not None
    fake.buffer = ""
    fake.completion_type = 9  # libedit reports Ctrl+P as Tab
    assert fake.completer("", 0) == "Hi"
    fake.buffer = "q"
    assert fake.completer("q", 0) == "quit"
    fake.buffer = "re"
    assert fake.completer("re", 0) == "reset"
    fake.buffer = "H"
    fake.completion_type = _CTRL_P
    assert fake.completer("H", 0) == "Hi"
    fake.buffer = "Hi"
    assert fake.completer("Hi", 0) == "How is ETH"
    fake.completion_type = _CTRL_N
    fake.buffer = "How is ETH"
    assert fake.completer("How is ETH", 0) == "Hi"
    assert fake.completer("re", 1) is None
    fake.buffer = "What is BTC"
    assert fake.completer("What is BTC", 0) == "What is BTC RSI"
    fake.buffer = "What is BTC RSI on "
    assert fake.completer("What is BTC RSI on ", 0) is None


def test_bind_tab_complete_uses_libedit_syntax() -> None:
    class _Gnu:
        __doc__ = "Importing this module enables command line editing using GNU readline."
        binds: list[str] = []

        def parse_and_bind(self, cmd: str) -> None:
            self.binds.append(cmd)

    class _Edit:
        __doc__ = "Importing this module enables command line editing using libedit readline."
        binds: list[str] = []

        def parse_and_bind(self, cmd: str) -> None:
            self.binds.append(cmd)

    gnu, edit = _Gnu(), _Edit()
    assert _readline_uses_libedit(gnu) is False
    assert _readline_uses_libedit(edit) is True
    _bind_tab_complete(gnu)
    _bind_tab_complete(edit)
    assert gnu.binds == ["tab: complete"]
    assert edit.binds == ["bind ^I rl_complete"]


def test_bind_cycle_nav_keys_always_complete() -> None:
    class _Gnu:
        __doc__ = "GNU readline."
        binds: list[str] = []

        def parse_and_bind(self, cmd: str) -> None:
            self.binds.append(cmd)

    class _Edit:
        __doc__ = "libedit readline."
        binds: list[str] = []

        def parse_and_bind(self, cmd: str) -> None:
            self.binds.append(cmd)

    gnu, edit = _Gnu(), _Edit()
    _bind_cycle_nav_keys(gnu)
    _bind_cycle_nav_keys(edit)
    assert gnu.binds == [r'"\C-p": complete', r'"\C-n": complete']
    assert edit.binds == ["bind ^P rl_complete", "bind ^N rl_complete"]


def test_completion_source_maps_invoking_key() -> None:
    class _Rl:
        def __init__(self, kind: int) -> None:
            self.kind = kind

        def get_completion_type(self) -> int:
            return self.kind

    assert _completion_source(_Rl(9)) == "tab"
    assert _completion_source(_Rl(_CTRL_P)) == "older"
    assert _completion_source(_Rl(_CTRL_N)) == "newer"
    assert _completion_wants_reverse(_Rl(_CTRL_N)) is True
    assert _completion_wants_reverse(_Rl(9)) is False
    assert _completion_source(object()) == "tab"
