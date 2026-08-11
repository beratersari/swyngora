from swyngora_ai.agents.prompts import (
    ANALYST_SYSTEM,
    DISCLAIMER,
    HORIZON,
    MARKET_SYSTEM,
    ORCHESTRATOR_SYSTEM,
    WEB_SYSTEM,
    X_SYSTEM,
)


def test_prompts_mention_not_advice():
    for p in (ORCHESTRATOR_SYSTEM, MARKET_SYSTEM, WEB_SYSTEM, X_SYSTEM, ANALYST_SYSTEM, DISCLAIMER):
        assert (
            "financial advice" in p.lower()
            or "not financial" in p.lower()
            or "informational" in p.lower()
        )


def test_prompts_default_1_2_day_horizon():
    for p in (ORCHESTRATOR_SYSTEM, MARKET_SYSTEM, WEB_SYSTEM, X_SYSTEM, ANALYST_SYSTEM, HORIZON):
        assert "1–2" in p or "1-2" in p or "1–2 day" in p or "1-2 day" in HORIZON


def test_orchestrator_routes_specialists():
    o = ORCHESTRATOR_SYSTEM.lower()
    for name in ("market_agent", "web_agent", "x_agent", "analyst_agent"):
        assert name in o
    assert "web_research" in WEB_SYSTEM.lower()
    assert "do **not** write" in WEB_SYSTEM.lower()


def test_analyst_requires_structure():
    a = ANALYST_SYSTEM.lower()
    assert "bottom line" in a
    assert "invalidation" in a
    assert "1–2" in ANALYST_SYSTEM or "1-2" in ANALYST_SYSTEM
