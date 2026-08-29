# libs/utils

Pure TypeScript helpers (no React components, no network I/O).

Domain rules live in the Go API. This folder is formatters, URL/session, and API→chart adapters. Venue quotes and RSI zones come from the API; the SPA does not keep a second copy.

Examples:

| Module           | Purpose                                    |
| ---------------- | ------------------------------------------ |
| `formatPrice.ts` | Price / volume / mcap display              |
| `spotQuery.ts`   | Build/normalize spot list query args       |
| `spotMetrics.ts` | Shared metric column catalog + prefs       |
| `rtkQuery.ts`    | Read RTK Query current-arg data only       |
| `displayCurrency.ts` | Apply published FX rates + venue quote maps from the API |
| `indicators.ts`  | Map API indicator points and RSI `zone` onto chart/UI |

Colocate unit tests as `*.test.ts` next to the module.
