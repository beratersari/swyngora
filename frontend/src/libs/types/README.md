# libs/types

Shared frontend types that are **not** OpenAPI DTOs.

- Re-export selected generated types if the app needs a stable import path
- UI view-models that map API → screen (prefer feature `*.types.ts` when local)

Do **not** hand-copy backend response shapes here long-term — use OpenAPI codegen under `libs/api/generated/`.
