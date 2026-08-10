# components/organisms

Domain UI sections that **know product concepts** (markets, symbols, RSI) but:

- receive data via **props** (no RTK in organisms)
- compose atoms / molecules / antd
- are used by **pages**

## Layout rule (Option A — no `features/`)

| Level | Path | Knows domain? | Fetches API? |
|---|---|---|---|
| atoms | `components/atoms/` | no | no |
| molecules | `components/molecules/` | no (generic chart hosts OK) | no |
| **organisms** | `components/organisms/` | **yes** | **no** |
| templates | `components/templates/` | layout only | no |
| pages | `components/pages/` | yes | **yes** (RTK only here) |

Do **not** reintroduce `src/features/` for product UI until multiple product areas need isolation (watchlist / paper / AI). Prefer organisms first.

## Current organisms

| Folder | Used by |
|---|---|
| `ExchangeTabs` | MarketsPage |
| `MarketsToolbar` | MarketsPage |
| `MarketsTable` | MarketsPage |
| `DetailHeader` | CoinDetailPage |
| `DetailStats` | CoinDetailPage |
| `DetailChartToolbar` | CoinDetailPage |
| `IndicatorPanel` | CoinDetailPage |
| `OrderBookPanel` | CoinDetailPage |

Import example:

```ts
import { MarketsTable } from '@/components/organisms/MarketsTable';
```
