# features/markets

Multi-exchange spot markets **UI feature** (Epic B).

| Path          | Role                                                           |
| ------------- | -------------------------------------------------------------- |
| `components/` | Feature organisms/molecules (Atomic file split)                |
| (page wiring) | Prefer `components/pages/MarketsPage` or export page from here |

**Data layer:** import from `@/libs/api` (spot/exchanges/tags endpoints).  
**Shared hooks/utils:** `@/libs/hooks`, `@/libs/utils`.

Do not reintroduce `features/markets/api` for backend REST — extend `libs/api/endpoints/` instead.
