# AI desk evals

Thirty fixture questions that score the **deterministic router** (and, in unit tests, grounding / allowlist). No live LLM.

## Run

```bash
cd ai
source .venv/bin/activate
pytest tests/test_evals.py -q
```

`questions.json` is the source of truth. Add a case when you add a routing keyword.
