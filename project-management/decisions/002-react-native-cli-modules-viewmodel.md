# Decision 002: React Native CLI, modules + ViewModel, shared brand tokens

**Date:** 2026-07-26  
**Status:** Accepted  
**Scope:** Product mobile (`mobile/`)

## Decision

1. **Runtime:** React Native TypeScript app with **no Expo**. **Primary dev target is Chrome** via **react-native-web** + Vite (`npm run web` → http://localhost:5180). Native `android/` / `ios/` projects are deferred; source still uses RN primitives for later native targets.
2. **UI structure:** Atomic Design under `src/components/` for **atoms → molecules → organisms → templates only**. **Pages must not live under `components/`.**
3. **Features:** Product capabilities live under **`src/modules/<name>/`**. **Each module owns its pages** (and optional flat module-local components).
4. **Page pattern:** Every page is **View + ViewModel**:
   - `Name.tsx` — render from VM only
   - `Name.viewModel.ts` — RTK Query, AppState polling options, navigation actions, mapping
   - `Name.types.ts` — normative VM shape; optional injectable `viewModel` prop for tests
5. **Data:** RTK Query + OpenAPI codegen under `src/libs/api` (same contract as web: `backend/api/openapi/openapi.yaml`).
6. **Brand colors:** Mobile **uses the same color tokens as frontend** — copy `colors` and `semanticColors` from `frontend/src/styles/tokens/colors.ts` into `mobile/src/styles/tokens/colors.ts`. Keep spacing scale aligned with web `spacing.ts` where practical.
7. **Styling:** React Native `StyleSheet` + token files; custom Atomic atoms first (no Ant Design on mobile).
8. **Navigation:** React Navigation in `src/app/navigation`; modules export screen names only.
9. **Focus/poll hygiene:** Primary control is `pollingInterval` from AppState (`useAppStateActive`); `refetchOnFocus: false` in init (browser focus listeners are incomplete on RN).

## Rationale

| Choice | Why |
|---|---|
| No Expo | Product/team requirement; avoid Expo lock-in for native market tooling |
| Modules own pages | Clear feature boundaries; avoids web’s `components/pages` mixing design system with routes |
| ViewModel | Prevents fat screens (web MarketsPage anti-pattern); testable without full tree |
| Same color tokens | One brand across web and mobile; agents must not invent a second palette |
| RTK + OpenAPI | Same data contract as product web; regenerate from one OpenAPI source |

## Alternatives considered

| Alternative | Outcome |
|---|---|
| Expo managed | Rejected (hard requirement) |
| Pages under `components/pages` | Rejected for mobile; modules own pages |
| `features/` naming | Prefer **`modules/`** for clearer page ownership |
| Class / MobX ViewModels | Rejected; hooks + RTK fit better |
| React Native Paper from day one | Deferred; custom atoms until a separate decision |
| Shared `packages/api-types` in init | Deferred; duplicate codegen in mobile for speed |

## Rules

1. Install **no** `expo` / `expo-*` packages during mobile init without a new decision.
2. Never add `src/components/pages/`.
3. Never hand-edit `src/libs/api/generated/`.
4. When changing brand colors, update **frontend and mobile** token files (or open a same-sprint mobile follow-up).
5. Views must not import RTK Query hooks; ViewModels must not return React elements.
6. Enforce import boundaries with ESLint `no-restricted-imports` (see system design).

## References

- `docs/design/mobile-project-initialization.md`
- `docs/design/mobile-system-design.md`
- `frontend/src/styles/tokens/colors.ts`
- root `AGENTS.md` §6.6 / §6.8 / §6.9
