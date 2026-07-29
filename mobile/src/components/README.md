# components (Atomic Design)

**Atoms → molecules → organisms → templates only.**

**Do not put pages here.** Pages live under `src/modules/<name>/pages/`.

**Do not put feature UI under modules.** All UI composition belongs here.

| Level | Folder | Examples (kebab-case dirs) |
|-------|--------|----------|
| Atoms | `atoms/` | `text/`, `button/`, `skeleton/`, `icon/` (Lucide) |
| Molecules | `molecules/` | `chip/`, `search-field/`, `chip-group/`, `star-button/`, `rsi-badge/`, `language-switcher/`, `chat-bubble/`, `chat-composer/`, `chat-disclaimer/`, `chat-tools-chips/`, `section-header/`, `quick-action-chips/` |
| Organisms | `organisms/` | `exchange-chips/`, `markets-toolbar/`, `market-row/`, `markets-list/`, `watchlist-row/`, `chat-message-list/`, `dashboard-market-row/`, `dashboard-section-list/`, `pump-teaser-card/`, `category-chip-grid/`, `category-section/` |
| Templates | `templates/` | `screen-template/` |

**Folders:** kebab-case. **Files:** PascalCase (`StarButton.tsx` inside `star-button/`).

## Dependency rules

- Lower levels must not import higher levels.
- Components must **not** import `@/modules` or `@/app`.
- No RTK Query / navigation in atoms or molecules.
- Organisms receive data via props only (pages / ViewModels own data).

## Modules

`src/modules/` owns **pages**, **context**, **navigation**, and **ViewModels** only — not UI component trees.
