# libs/utils

Pure TypeScript helpers (no React components, no network I/O).

Examples:

| Module           | Purpose                                    |
| ---------------- | ------------------------------------------ |
| `formatPrice.ts` | Price / volume / mcap display              |
| `exchange.ts`    | Symbol format quirks (Binance vs Coinbase) |
| `spotQuery.ts`   | Build/normalize spot list query args       |
| `spotMetrics.ts` | Shared metric column catalog + prefs       |
| `rtkQuery.ts`    | Read RTK Query current-arg data only       |
| `displayCurrency.ts` | Convert BIST/Nasdaq/crypto quotes for display |

Colocate unit tests as `*.test.ts` next to the module.
