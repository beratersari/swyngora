"""CLI entrypoint: swyngora-ai"""

from __future__ import annotations

import argparse
import sys


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Swyngora multi-agent AI assistant")
    parser.add_argument("message", nargs="?", help="One-shot question (omit for REPL)")
    parser.add_argument("--session", default="cli", help="Session id for memory")
    parser.add_argument("--provider", choices=["ollama", "grok"], default=None)
    args = parser.parse_args(argv)

    # Optional provider override
    if args.provider:
        import os

        os.environ["AI_LLM_PROVIDER"] = args.provider

    from swyngora_ai.graph.orchestrator import build_orchestrator

    orch = build_orchestrator()
    if args.message:
        print(orch.chat(args.message, session_id=args.session))
        return 0

    print("Swyngora AI (orchestrator). Type 'exit' to quit. Not financial advice.")
    while True:
        try:
            line = input("you> ").strip()
        except (EOFError, KeyboardInterrupt):
            print()
            return 0
        if not line:
            continue
        if line.lower() in {"exit", "quit"}:
            return 0
        if line.lower() == "reset":
            orch.reset(args.session)
            print("(session cleared)")
            continue
        try:
            print(orch.chat(line, session_id=args.session))
        except Exception as e:  # noqa: BLE001
            print(f"error: {e}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
