# AI conversation memory (FinMem)

Working chat history plus optional durable layers.

## Layout

| Layer | Store | Purpose |
|-------|--------|---------|
| Working | in-process, thread-safe | Last ~40 Human/AI turns, keyed by `clientId` + `sessionId` |
| Daily | SQLite `daily_notes` | Truncated Q→A bullets per UTC day |
| Long-term | SQLite `facts` | `last_symbol`, `last_exchange`, `last_tape_at` |

## Config

| Variable | Default | Meaning |
|----------|---------|---------|
| `AI_MEMORY_PATH` | empty (RAM only) | SQLite file, or `:memory:` |

Tape older than `TAPE_TTL_SECONDS` (300) is treated as stale; the desk re-fetches numbers.

## Tests

`cd ai && pytest tests/test_finmem.py`
