"""P3: 30-question eval set — routing + grounding fixtures (no live LLM)."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from swyngora_ai.graph.facts import extract_market_facts
from swyngora_ai.graph.route import classify_route
from swyngora_ai.grounding import apply_grounding, strip_unallowed_urls

_EVALS = Path(__file__).resolve().parents[1] / "evals" / "questions.json"


def _cases() -> list[dict[str, Any]]:
    data = json.loads(_EVALS.read_text(encoding="utf-8"))
    return list(data["cases"])


def test_eval_set_has_at_least_30():
    cases = _cases()
    assert len(cases) >= 30
    ids = [c["id"] for c in cases]
    assert ids == list(range(1, len(cases) + 1))


def test_eval_routing_matches_expected_flags():
    failures: list[str] = []
    for case in _cases():
        route = classify_route(case["q"]).as_dict()
        for flag, want in (case.get("expect") or {}).items():
            if bool(route.get(flag)) is not bool(want):
                failures.append(
                    f"#{case['id']} {case['q']!r}: {flag} want {want} got {route.get(flag)}"
                )
    assert not failures, "\n".join(failures)


def test_eval_citation_and_grounding_fixtures():
    cleaned = strip_unallowed_urls(
        "Read [ok](https://www.coindesk.com/x) and [bad](https://spam.example/p)"
    )
    assert "coindesk.com" in cleaned
    assert "spam.example" not in cleaned

    facts = extract_market_facts(json.dumps({"lastPrice": "2500", "symbol": "ETHUSDT"}))
    flagged = apply_grounding("ETH last is 99999.0", facts, [])
    assert "Unverified" in flagged
    ok = apply_grounding("ETH last is 2500", facts, [])
    assert "Unverified" not in ok
