# config

Env and app-level constants.

| Env var | Purpose |
| --- | --- |
| `VITE_API_BASE_URL` | Optional absolute API origin. Empty = same-origin + Vite proxy. |
| `VITE_CLIENT_ID` | Optional watchlist client id → `X-Client-Id` header via `baseApi`. |

See `frontend/AGENTS.md` and `docs/design/frontend-system-design.md`.
