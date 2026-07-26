# GitLab project management: Mobile epics & issues

**Project:** `trace-analysis/swyngora`  
**Host:** https://nova.teachx.ai  
**Labels (create if missing):** `mobile`, `type::epic`, `type::feature`, `type::chore`, `priority::p0`, `priority::p1`

**Local board (current):** [`project-management/`](../../project-management/) — day-to-day status until GitLab issues exist.

---

## Epic C — Mobile project initialization (FIRST for mobile)

| Field | Value |
|---|---|
| **Title** | `[mobile] Project initialization` |
| **Type** | Epic |
| **Labels** | `mobile`, `priority::p0` |
| **Design plan** | `docs/design/mobile-project-initialization.md` |
| **System design** | `docs/design/mobile-system-design.md` |
| **Decision** | `project-management/decisions/002-react-native-cli-modules-viewmodel.md` |

### Epic description (paste into GitLab)

```markdown
## Summary

Initialize the production React Native app under `mobile/` (no Expo) so feature work can start.

This is **P0 for the mobile track** and **blocks** multi-exchange spot markets UI on mobile.

## Goals

- React Native CLI + TypeScript scaffold (**no Expo**)
- Lint / test tooling + path aliases
- Atomic Design under components (no pages) + modules that own pages
- View + ViewModel page contract
- Redux Toolkit + RTK Query baseApi
- OpenAPI codegen from backend/api/openapi/openapi.yaml
- React Navigation shell + AppState polling hygiene
- **Same brand color tokens as frontend**
- Package README + AGENTS.md

## Out of scope

Full markets table, coin detail, charts, auth, Expo.

## Acceptance

- npm run android launches shell (documented host)
- npm run codegen:api works
- colors.ts brand hex matches frontend
- Home + MarketsList stubs with ViewModels
- Structure matches mobile/AGENTS.md

## Child issues

See MINIT-1 … MINIT-9.
```

### Child issues

| ID | Title |
|---|---|
| MINIT-1 | Scaffold React Native CLI TypeScript app (no Expo) |
| MINIT-2 | Lint, format, Jest, path aliases |
| MINIT-3 | libs + Atomic + modules skeleton + boundary ESLint |
| MINIT-4 | libs/api store + RTK baseApi + env |
| MINIT-5 | OpenAPI codegen → libs/api/generated |
| MINIT-6 | Navigation shell + AppState + providers |
| MINIT-7 | Color tokens (match frontend) + core atoms + ScreenTemplate |
| MINIT-8 | Home + Markets stub pages with ViewModels |
| MINIT-9 | Package docs + root links + changelog |

Full task text: `project-management/tasks/MINIT-*.md`.
