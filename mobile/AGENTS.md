# Mobile package — agent guide

Closest `AGENTS.md` wins under `mobile/`. Root monorepo rules still apply.

## Hard rules

1. **No Expo** — do not add `expo` / `expo-*` packages.
2. **Pages live under `src/modules/<name>/pages/`** — never `src/components/pages/`.
3. **All UI uses Atomic Design under `src/components/`** — atoms → molecules → organisms → templates.
4. **No feature/module component folders** — do not create `modules/*/components/`.
5. **Folder names are kebab-case** — e.g. `star-button/`, `coin-detail-page/`. File names stay PascalCase (`StarButton.tsx`).
6. **View + ViewModel** for every page:
   - `Name.tsx` — compose Atomic organisms/molecules; optional injectable `viewModel` prop for tests
   - `Name.viewModel.ts` — RTK Query, AppState polling, mapping (no JSX)
7. **Brand colors** must match `frontend/src/styles/tokens/colors.ts` (mobile tokens copy).
8. **RTK Query** only under `src/libs/api` — not in atoms/molecules/organisms.
9. **Primary run target:** Chrome via `npm run web` (react-native-web).

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
modules/<m>/pages/foo-page/
  FooPage.tsx              # composes @/components/* only
  FooPage.viewModel.ts
  FooPage.types.ts
  FooPage.styles.ts
  FooPage.test.tsx
  index.ts
```

Import example: `@/components/molecules/star-button` → `export { StarButton } from './StarButton'`.

## Markets module

- Endpoints: `libs/api/endpoints/marketApi.ts`
- Pages: `modules/markets/pages/*` (View + ViewModel + context) — kebab folders
- UI: `components/organisms/{exchange-chips,markets-toolbar,market-row,markets-list,markets-filter-form}`
- Molecules: `components/molecules/{chip,search-field,chip-group}`
- State: `MarketsProvider` in `modules/markets/context/`
- Infinite scroll + filter screen; search debounce 300ms

## Data / freshness

- Use `useAppStateActive()` and set `pollingInterval` to `0` when inactive.
- Prefer `refetchOnFocus: false`.

## Related docs

- `docs/design/mobile-project-initialization.md`
- `docs/design/mobile-system-design.md`
- `project-management/decisions/002-react-native-cli-modules-viewmodel.md`

## Coin detail

- Page: `modules/markets/pages/coin-detail-page/`
- Navigate: `MarketsScreens.Detail` `{ exchange, symbol }`
- Organisms: `coin-detail-header`, `coin-detail-stats`, `interval-toolbar`, `candle-chart`, `indicator-rsi-pane`
- RTK: intervals, ticker, supply, candles, indicators in `marketApi.ts`
- Chart: `lightweight-charts` on react-native-web only

## Watchlist / favorites

- Module: `modules/watchlist/` (context + `pages/watchlist-page`)
- RTK: `libs/api/endpoints/watchlistApi.ts`
- Helpers: `libs/utils/clientId.ts`, `watchlistKey.ts`, `watchlistMerge.ts`
- UI: `molecules/star-button`; stars on `market-row` / `coin-detail-header`; Favorites tab (hidden when empty)
- clientId: `mobile-<uuid>` in localStorage; header `X-Client-Id`
- Max items: 200 (backend cap)

