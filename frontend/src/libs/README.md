# libs/

Shared non-UI layers for the product frontend. **Import from here** instead of scattering API clients, hooks, and helpers across features.

| Package   | Path          | Responsibility                                                                                 |
| --------- | ------------- | ---------------------------------------------------------------------------------------------- |
| **api**      | `libs/api/`      | RTK Query `baseApi`, OpenAPI-generated clients, endpoint injectors, Redux store wiring for API |
| **realtime** | `libs/realtime/` | WebSocket client: subscribe selected coins + paper portfolio; patches RTK caches             |
| **hooks**    | `libs/hooks/`    | Shared React hooks (visibility, debounce, media, etc.) — not feature page containers           |
| **utils** | `libs/utils/` | Pure helpers (formatters, guards, query builders) — no React components                        |
| **types** | `libs/types/` | Shared app/view types re-exports (not hand-written OpenAPI DTOs)                               |

## Rules

1. **UI stays out of libs** — React components live under `components/` or `features/*/components/`.
2. **Backend HTTP only via `libs/api`** (RTK Query). No ad-hoc `fetch` in features for standard REST.
3. **`libs/api/generated/` is read-only** — regenerate with `npm run codegen:api`.
4. Feature modules **import** libs; libs must **not** import `features/*` or Atomic pages.
5. Prefer `@/libs/api`, `@/libs/hooks`, `@/libs/utils` path aliases (set in init epic).

## Dependency direction

```text
app / features / components/pages
        ↓
   libs/hooks  →  libs/api  →  backend
        ↓
   libs/utils
```

`libs/api` may use `libs/utils`. `libs/hooks` may use `libs/api` and `libs/utils`. Never the reverse into features.
