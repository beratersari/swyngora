# Mobile package — agent guide

Closest `AGENTS.md` wins under `mobile/`. Root monorepo rules still apply.

## Hard rules

1. **No Expo** — do not add `expo` / `expo-*` packages.
2. **Pages live under `src/modules/<name>/pages/`** — never `src/components/pages/`.
3. **Atomic only in `src/components/`** — atoms → molecules → organisms → templates.
4. **View + ViewModel** for every page:
   - `Name.tsx` — render from VM; optional injectable `viewModel` prop for tests
   - `Name.viewModel.ts` — RTK Query, AppState polling, mapping (no JSX)
5. **Brand colors** must match `frontend/src/styles/tokens/colors.ts`.
6. **RTK Query** only under `src/libs/api` — not in atoms/molecules.
7. **Primary run target:** Chrome via `npm run web` (react-native-web). Do not require a simulator for day-to-day work.

## Import boundaries (ESLint)

| From | Must not import |
|------|-----------------|
| `src/components/**` | `@/modules`, `@/app` |
| `src/libs/**` | `@/modules`, `@/app`, `@/components` |

```bash
npm run lint
```

## Commands

```bash
npm run web          # Chrome — http://localhost:5180
npm test
npm run lint
npm run typecheck
npm run codegen:api  # also regenerate frontend when OpenAPI changes
```

## Page template

```text
modules/<m>/pages/FooPage/
  FooPage.tsx
  FooPage.viewModel.ts
  FooPage.types.ts
  FooPage.styles.ts
  FooPage.test.tsx
  index.ts
```

## Data / freshness

- Use `useAppStateActive()` and set `pollingInterval` to `0` when inactive.
- Prefer `refetchOnFocus: false` (do not rely on browser focus alone for RN parity).

## Related docs

- `docs/design/mobile-project-initialization.md`
- `docs/design/mobile-system-design.md`
- `project-management/decisions/002-react-native-cli-modules-viewmodel.md`
