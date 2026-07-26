# components/

Shared Atomic Design UI (atoms → molecules → organisms → templates → pages).

- No RTK Query / backend fetch inside **atoms** or **molecules**
- Pages and feature containers import `@/libs/api` hooks and pass props down

See `frontend/AGENTS.md` and `docs/design/frontend-system-design.md`.
