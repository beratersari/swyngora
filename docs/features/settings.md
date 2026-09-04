# Feature: Settings desk

## Problem / goal

API keys, user-data export/import, watchlist/portfolio sharing, and paper recurring
buys existed as HTTP + MCP only. Users need a desk page.

## Behavior

- Route `/settings` (nav item).
- Tabs: API keys, export, import, sharing, recurring buys.
- Import preview lists custom signal mixes that will be **added** vs **updated**
  before confirm (`addNames` / `updateNames` from `POST /api/v1/import/preview`).
- Uses existing OpenAPI endpoints and the current `X-Client-Id`.
- Creating a **read** key shows the secret once and does **not** write it
  to `swyngora.apiAuthToken` (read keys 403 paper mutations). A **trade**
  key may become the browser session token.
- Paper copy stays labeled as not real money.

## Where the code lives

| Layer | Path |
|---|---|
| Web | `frontend/src/components/pages/SettingsPage/` |
| RTK | `accountApi`, `exportApi`, `importApi`, `recurringApi`, watchlist/portfolio share hooks |

## Verify

Open `/settings`. Create a key, start an export, preview an import (new vs updated profiles), share a watchlist, add a daily paper buy.

## Limits

- Native iOS/Android settings are not in this change (mobile is still `npm run web`).
