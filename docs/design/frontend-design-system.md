# Swyngora frontend design system

**Status:** Active  
**Package:** `frontend/`  
**Stack:** React · Ant Design · **styled-components** · design tokens · Atomic Design  
**Branch context:** product web UI  

---

## 1. Purpose

A single source of truth for **color**, **typography**, **loading**, and **core atoms** so features (markets, detail, watchlist) stay visually consistent.

| Concern | Location |
|---|---|
| Brand + semantic colors | `frontend/src/styles/tokens/colors.ts` |
| Type scale + fonts | `frontend/src/styles/tokens/typography.ts` |
| Spacing / radius | `frontend/src/styles/tokens/spacing.ts` |
| Motion | `frontend/src/styles/tokens/motion.ts` |
| styled-components theme | `frontend/src/styles/theme.ts` + `GlobalStyle.ts` |
| Ant Design theme map | `frontend/src/styles/antdTheme.ts` |
| Text atom | `frontend/src/components/atoms/Text/` |
| Skeleton atom | `frontend/src/components/atoms/Skeleton/` |
| Loading prop contract | `frontend/src/components/types/loading.types.ts` |

---

## 2. Color palette (brand)

### Primary

| Token | Hex | Role |
|---|---|---|
| **richBlack** | `#000F0F` | Deepest canvas / app chrome |
| **darkGreen** | `#032221` | Page / muted card background |
| **bangladeshGreen** | `#03624C` | Primary actions, table headers, strong borders |
| **mountainMeadow** | `#4FD4A5` | Accents, links, success / chart up secondary |
| **caribbeanGreen** | `#00FF81` | High-emphasis accent, focus, chart up, tab ink |
| **antiFlashWhite** | `#F1F7F6` | Primary text on dark |

### Secondary

| Token | Hex | Role |
|---|---|---|
| **pine** | `#063028` | Deep elevated / tag bg |
| **basil** | `#0B453A` | Elevated surfaces, Ant elevated |
| **forest** | `#095544` | Mid elevation |
| **frog** | `#17876D` | Hover primary, EMA slow |
| **mint** | `#74F9BC` | Soft accent, link hover |

### Neutral

| Token | Hex | Role |
|---|---|---|
| **stone** | `#707D7D` | Secondary text, muted icons |
| **pistachio** | `#AACBC4` | Soft borders / info / secondary labels |

### Semantic mapping

| Semantic | Maps to |
|---|---|
| `bg.canvas` | richBlack (deepest shell) |
| `bg.page` | darkGreen (page field) |
| `bg.chrome` | pine (header / toolbars) |
| `bg.muted` | darkGreen (cards / tables) |
| `bg.elevated` | basil (popovers) |
| `bg.hover` | frog @ ~22% |
| `bg.tableHeader` | bangladeshGreen @ ~42% |
| `text.primary` | antiFlashWhite |
| `text.secondary` | pistachio (readable secondary) |
| `text.tertiary` | stone (muted meta) |
| `text.link` / `accent` | mountainMeadow |
| `action.primary` | bangladeshGreen |
| `action.primaryHover` | frog |
| `status.success` | mountainMeadow |
| `chart.up` | caribbeanGreen (**charts + focus only**) |
| `chart.down` | `#E07A7A` (readable red; not in brand set) |

## 2b. Motion

Desk UI uses **short, functional** motion — never cinematic. Tokens: `theme.motion.duration` (`instant` / `fast` / `base` / `slow`) and `theme.motion.ease`.

| Pattern | Where |
|---|---|
| Page enter (fade + 10px rise) | `PageEnter` on route change |
| Star pop | `WatchStar` when a pair is added |
| Price flash (green/red wash) | `FlashValue` on live ticks |
| Live pulse | `ConnectionStatus` while API is live |
| Hover lift | buttons, nav pills, setup cards |
| Tape pause | ticker marquee pauses when the tab is hidden |

Always wrap new animations in `@media (prefers-reduced-motion: reduce)`. `GlobalStyle` disables long transitions globally for that query.

**Rules**

1. Import from `@/styles/tokens` (`palette`, `semanticColors`, `withAlpha`) — do **not** hardcode hex/`rgba` in features.
2. **anti-flash white** on dark greens for body; **pistachio** secondary; **stone** tertiary.
3. **caribbeanGreen** is for chart up + `:focus-visible` only — never for tabs, badges, or pagination.
4. UI accents / active states use **mountainMeadow**.
5. Legacy aliases `navy` / `indigo` / `steel` / `cream` remain on `colors` for old call sites; prefer `semantic.*`.

```ts
import { palette, semanticColors } from '@/styles/tokens';
```

styled-components:

```ts
import styled from 'styled-components';

export const Panel = styled.div`
  background: ${({ theme }) => theme.semantic.bg.canvas};
  color: ${({ theme }) => theme.semantic.text.primary};
`;
```

Colocate styles in **`ComponentName.styles.ts`** next to the component (no CSS/CSS-modules files).

---

## 3. Typography

### Font families

| Role | Stack |
|---|---|
| **Sans (UI)** | DM Sans, Segoe UI, system-ui |
| **Mono (prices / code)** | JetBrains Mono, SF Mono, ui-monospace |

Loaded via Google Fonts in `global.css` with system fallbacks.

### Type scale (`typeScale`)

| Variant | Size | Weight | Use |
|---|---|---|---|
| `display` | 36 | Bold | Marketing / empty hero |
| `h1` | 30 | Bold | Page title (rare) |
| `h2` | 24 | Semibold | Page section (Markets) |
| `h3` | 20 | Semibold | Subsection |
| `h4` | 16 | Semibold | Card titles |
| `bodyLg` | 16 | Regular | Lead paragraphs |
| `body` | 14 | Regular | Default copy |
| `bodySm` | 13 | Regular | Dense UI |
| `caption` | 12 | Regular | Footnotes, timestamps |
| `overline` | 11 | Semibold · uppercase | Column group labels |
| `label` | 13 | Medium | Form labels, chips text |
| `code` | 13 | Regular · mono | IDs, API bases |
| `numeric` | 14 | Medium · mono · tabular | Prices, volumes, mcap |

### Text atom

```tsx
import { Text } from '@/components/atoms/Text';

<Text variant="h2" color="primary">Markets</Text>
<Text variant="body" color="secondary">Secondary description</Text>
<Text variant="numeric" color="success">+2.4%</Text>
<Text variant="body" isLoading skeletonWidth={120} />
```

| Prop | Description |
|---|---|
| `variant` | Type scale key |
| `color` | `primary` \| `secondary` \| `inverse` \| `cream`* \| `steel`* \| `success` \| `warning` \| `error` (*legacy aliases → primary/secondary) |
| `as` | Polymorphic element |
| `truncate` | Ellipsis |
| `mono` | Force mono stack |
| `isLoading` | Renders skeleton instead of text |

---

## 4. Skeleton & `isLoading` contract

Every design-system / Atomic component that shows remote or deferred content **must** support:

```ts
type WithLoadingProps = {
  isLoading?: boolean;
};
```

Defined in `frontend/src/components/types/loading.types.ts`.

### Behavior

| `isLoading` | Expected UI |
|---|---|
| `true` | Skeleton (or equivalent), **no** flash of empty wrong content |
| `false` / omitted | Real content |

### Skeleton atom

```tsx
import { Skeleton } from '@/components/atoms/Skeleton';

<Skeleton variant="text" rows={3} />
<Skeleton variant="chart" height={280} />
<Skeleton variant="button" />

{/* Wrapper mode */}
<Skeleton isLoading={loading} variant="card">
  <MarketsTable ... />
</Skeleton>
```

| Variant | Typical use |
|---|---|
| `text` / `title` | Copy blocks |
| `button` | Actions |
| `avatar` | Icons / avatars |
| `input` | Filters |
| `chart` | Candle / series charts |
| `card` | Panel placeholders |
| `image` | Media |

### Components with `isLoading` (current)

| Component | Loading UI |
|---|---|
| `Text` | text / title skeleton |
| `Button` | button skeleton (`pending` = in-button spinner) |
| `Skeleton` | self / wrapper |
| `CandleChartHost` | chart skeleton |

**New components must:**

1. Extend `WithLoadingProps`.
2. Prefer `<Skeleton variant="…" />` when `isLoading`.
3. Document the prop in `*.types.ts`.

```tsx
export function MarketsTable({ isLoading, rows }: MarketsTableProps) {
  if (isLoading) {
    return <Skeleton variant="card" height={320} />;
  }
  return <table>…</table>;
}
```

---

## 5. Spacing & radius

4px grid (`spacing[1] = 4` … `spacing[16] = 64`).  
Radii: `sm` 4 · `md` 8 · `lg` 12 · `pill` 999.

---

## 6. Ant Design integration

`ConfigProvider` uses `antdTheme` derived from tokens (`app/providers.tsx`).

- Primary = bangladeshGreen  
- Text base = antiFlashWhite  
- Layout / cards = richBlack / darkGreen  
- Skeleton gradients = brand skeleton tokens  

Prefer **Atomic wrappers** (`Text`, `Button`) over raw `Typography` in new UI.

---

## 7. Atomic Design map (Option A — no `features/`)

```text
atoms/      Text, Button, Skeleton
molecules/  CandleChartHost, IndicatorChartHost (+ isLoading)
organisms/  MarketsTable, MarketsToolbar, ExchangeTabs,
            DetailHeader, DetailStats, DetailChartToolbar, IndicatorPanel
templates/  page chrome (when needed)
pages/      MarketsPage, CoinDetailPage  ← RTK only here
```

---

## 8. Do / Don’t

| Do | Don’t |
|---|---|
| Import `@/styles/tokens` / `palette` | Scatter raw hex in features |
| Use `<Text variant="numeric">` for prices | Use default browser fonts for tables |
| Pass `isLoading` from RTK `isLoading` / `isFetching` | Leave empty boxes while data loads |
| Keep anti-flash white on rich black / dark green | Low-contrast gray-on-green body text |

---

## 9. Styling rules (styled-components only)

1. **No CSS / CSS Modules / SCSS** in `frontend/` for product UI.
2. Every component with styles gets a colocated **`Name.styles.ts`** file.
2b. Constants / helpers use **`Name.constants.ts`** / **`Name.helpers.ts`** (searchable; never bare `constants.ts`).
3. Use `ThemeProvider` theme (`@/styles/theme`) — brand colors and type live on `theme`.
4. Transient props use `$` prefix (`$truncate`, `$height`) so they are not forwarded to the DOM.
5. Global resets live in `GlobalStyle.ts` only.

## 10. File checklist for new UI

1. Tokens from `@/styles/tokens` / `theme`.  
2. Styles in `*.styles.ts` with styled-components.  
3. Copy via `<Text />`.  
4. Loading via `isLoading` + `<Skeleton />`.  
5. Colocate **name-prefixed** files: `Name.types.ts`, `Name.constants.ts`, `Name.helpers.ts`, `Name.styles.ts`.  
6. Unit test loading state when logic is non-trivial.

---

## 11. Changelog note

- Introduced brand palette (styled-components). Type scale (DM Sans + JetBrains Mono), Text + Skeleton, `isLoading`.
- **2026-07-26:** Palette replaced navy/indigo blues with green system — rich black, dark/bangladesh greens, mountain meadow, caribbean green, anti-flash white, secondary pine–mint, neutrals stone/pistachio.

**Last updated:** 2026-07-26
