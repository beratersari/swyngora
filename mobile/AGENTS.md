# Mobile package — agent guide

Closest `AGENTS.md` wins under `mobile/`. Root monorepo rules still apply.

## Hard rules

1. **No Expo** — do not add `expo` / `expo-*` packages.
2. **Pages live under `src/modules/<name>/pages/`** — never `src/components/pages/`.
3. **All UI uses Atomic Design under `src/components/`** — atoms → molecules → organisms → templates.
4. **No feature/module component folders** — do not create `modules/*/components/`.
5. **View + ViewModel** for every page:
   - `Name.tsx` — compose Atomic organisms/molecules; optional injectable `viewModel` prop for tests
   - `Name.viewModel.ts` — RTK Query, AppState polling, mapping (no JSX)
6. **Brand colors** must match `frontend/src/styles/tokens/colors.ts` (mobile tokens copy).
7. **RTK Query** only under `src/libs/api` — not in atoms/molecules/organisms.
8. **Primary run target:** Chrome via `npm run web` (react-native-web).

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
  FooPage.tsx              # composes @/components/* only
  FooPage.viewModel.ts
  FooPage.types.ts
  FooPage.styles.ts
  FooPage.test.tsx
  index.ts
```

## Markets module

- Endpoints: `libs/api/endpoints/marketApi.ts`
- Pages: `modules/markets/pages/*` (View + ViewModel + context)
- UI: `components/organisms/{ExchangeChips,MarketsToolbar,MarketRow,MarketsList,MarketsFilterForm}`
- Molecules: `components/molecules/{Chip,SearchField,ChipGroup}`
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

- Page: `modules/markets/pages/CoinDetailPage/`
- Navigate: `MarketsScreens.Detail` `{ exchange, symbol }`
- Organisms: CoinDetailHeader, CoinDetailStats, IntervalToolbar, CandleChart, IndicatorRsiPane
- RTK: intervals, ticker, supply, candles, indicators in `marketApi.ts`
- Chart: `lightweight-charts` on react-native-web only

## Watchlist

- Module: `modules/watchlist/` (context + WatchlistPage)
- RTK: `libs/api/endpoints/watchlistApi.ts`
- Helpers: `libs/utils/clientId.ts`, `watchlistKey.ts`, `watchlistMerge.ts`
- UI: `StarButton` molecule; stars on `MarketRow` / `CoinDetailHeader`; Watchlist tab
- clientId: `mobile-<uuid>` in localStorage; header `X-Client-Id`
- Max items: 200 (backend cap)

