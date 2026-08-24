"""create_agent + ModelRetryMiddleware retries a transient model error."""

from __future__ import annotations

from typing import Any

from langchain.agents import create_agent
from langchain.agents.middleware import ModelRetryMiddleware
from langchain_core.language_models.chat_models import BaseChatModel
from langchain_core.messages import AIMessage, HumanMessage
from langchain_core.outputs import ChatGeneration, ChatResult

from swyngora_ai.llm.retry import MODEL_RETRY


class _FlakyModel(BaseChatModel):
    fails_left: int = 0
    error: Exception = TimeoutError("timed out")
    reply: str = "ok"

    @property
    def _llm_type(self) -> str:
        return "flaky"

    def _generate(
        self,
        messages: list[Any],
        stop: list[str] | None = None,
        run_manager: Any = None,
        **kwargs: Any,
    ) -> ChatResult:
        if self.fails_left > 0:
            self.fails_left -= 1
            raise self.error
        return ChatResult(generations=[ChatGeneration(message=AIMessage(content=self.reply))])

    def bind_tools(self, tools: Any, **kwargs: Any) -> _FlakyModel:
        return self


def test_model_retry_constant_is_middleware() -> None:
    assert isinstance(MODEL_RETRY, ModelRetryMiddleware)
    assert MODEL_RETRY.max_retries == 3


def test_create_agent_retries_then_answers() -> None:
    model = _FlakyModel(fails_left=2, error=TimeoutError("blip"), reply="recovered")
    graph = create_agent(
        model,
        [],
        system_prompt="t",
        middleware=[
            ModelRetryMiddleware(max_retries=3, backoff_factor=2.0, initial_delay=0.0),
        ],
    )
    result = graph.invoke({"messages": [HumanMessage(content="q")]}, {"recursion_limit": 4})
    msgs = list((result or {}).get("messages") or [])
    texts = [str(getattr(m, "content", "")) for m in msgs]
    assert any("recovered" in t for t in texts)
    assert model.fails_left == 0
