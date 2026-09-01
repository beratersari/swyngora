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
| `HolderPanel` | CoinDetailPage |
| `PostDelistPanel` | CoinDetailPage |
| `DetailChartToolbar` | CoinDetailPage |
| `IndicatorPanel` | CoinDetailPage |
| `OrderBookPanel` | CoinDetailPage |
| `OrderDepthChart` | CoinDetailPage |
| `OrderHeatmap` | CoinDetailPage |
| `LiquidationHeatmap` | LiquidationsPage (`?view=heatmap`) |
| `LiquidationBarChart` | LiquidationsPage (`?view=chart`) |
| `LiquidationCascade` | LiquidationsPage (`?view=cascade`) |
| `LiquidationTreemap` / `LiquidationWindowCards` / `LiquidationFeedHealth` | LiquidationsPage overview + feed health |
| `PriceChangeHeatmap` | HeatmapPage |
| `RSIHeatmap` | HeatmapPage (`?view=rsi`) — ranked scatter |
| `AlertsTable` / `CreateAlertForm` | AlertsPage |
| `SignalsSetupGrid` / `SignalsHitsTable` / `SignalsRuleForm` / `SignalsRulesTable` / `SignalsBacktestPanel` | SignalsPage |

Motion molecules used by organisms: `WatchStar`, `FlashValue` (live ticks). Page enter lives in `App` (`PageEnter`).

Import example:

```ts
import { MarketsTable } from '@/components/organisms/MarketsTable';
```
