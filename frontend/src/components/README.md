# components/

Atomic Design UI only (no `src/features/` tree — **Option A**).

```text
atoms → molecules → organisms → templates → pages
```

| Level | Role |
|---|---|
| **atoms** | Design-system primitives (`Text`, `Button`, `Skeleton`) |
| **molecules** | Small compositions / chart hosts |
| **organisms** | Domain sections (tables, detail panels) — props only |
| **templates** | Layout shells without data |
| **pages** | Route screens — **only place that uses RTK Query** |

- No RTK / `fetch` inside **atoms**, **molecules**, or **organisms**
- Pages import `@/libs/api` and pass props down

See `organisms/README.md`, `frontend/AGENTS.md`, `docs/design/frontend-system-design.md`.
