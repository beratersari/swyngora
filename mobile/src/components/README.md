# components (Atomic Design)

**Atoms → molecules → organisms → templates only.**

**Do not put pages here.** Pages live under `src/modules/<name>/pages/`.

**Do not put feature UI under modules.** All UI composition belongs here.

| Level | Folder | Examples |
|-------|--------|----------|
| Atoms | `atoms/` | `Text`, `Button`, `Skeleton` |
| Molecules | `molecules/` | `Chip`, `SearchField`, `ChipGroup` |
| Organisms | `organisms/` | `ExchangeChips`, `MarketsToolbar`, `MarketRow`, `MarketsList`, `MarketsFilterForm` |
| Templates | `templates/` | `ScreenTemplate` |

## Dependency rules

- Lower levels must not import higher levels.
- Components must **not** import `@/modules` or `@/app`.
- No RTK Query / navigation in atoms or molecules.
- Organisms receive data via props only (pages / ViewModels own data).

## Modules

`src/modules/` owns **pages**, **context**, **navigation**, and **ViewModels** only — not UI component trees.
