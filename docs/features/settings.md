# Feature: Settings desk

## Problem / goal

API keys, user-data export, watchlist/portfolio sharing, and paper recurring
buys existed as HTTP + MCP only. Users need a desk page.

## Behavior

- Route `/settings` (nav item).
- Tabs: API keys, export, sharing, recurring buys.
- Uses existing OpenAPI endpoints and the current `X-Client-Id`.
- Paper copy stays labeled as not real money.

## Where the code lives

| Layer | Path |
|---|---|
| Web | `frontend/src/components/pages/SettingsPage/` |
| RTK | `accountApi`, `exportApi`, `recurringApi`, watchlist/portfolio share hooks |

## Verify

Open `/settings`. Create a key, start an export, share a watchlist, add a daily paper buy.

## Limits

- Native iOS/Android settings are not in this change (mobile is still `npm run web`).
- Import of user data is still API-only.
