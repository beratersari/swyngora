# Decision 001: Ant Design + TradingView Lightweight Charts

**Date:** 2026-07-26  
**Status:** Accepted  
**Scope:** Product frontend (`frontend/`)

## Decision

1. **UI component library:** [Ant Design](https://ant.design/) (`antd` + `@ant-design/icons` as needed).
2. **Chart library:** [TradingView Lightweight Charts](https://tradingview.github.io/lightweight-charts/) (`lightweight-charts`).

## Rationale

### Ant Design

- Mature React UI kit: Table, Form, Select, Tabs, Layout, Typography, theme tokens.
- Fits multi-exchange **markets table**, filters, pagination without inventing primitives.
- Dark/light themes via ConfigProvider — good for trading dashboards.
- Use **Atomic Design wrappers**: prefer `@/components/atoms/Button` wrapping `antd` rather than scattering raw `antd` imports in every feature (allows restyling and consistent API).

### Lightweight Charts

- Purpose-built for **financial OHLCV** candlesticks and volume histograms.
- Small, fast, free (Apache-2.0).
- Natural fit for coin detail (candles from `GET /api/v1/market/candles`).
- Indicator overlays (EMA lines) supported; RSI often as a separate pane/series pattern.

## Alternatives considered (charts)

| Library | Why not primary |
|---|---|
| Apache ECharts | Excellent general charts; heavier; candlesticks good but not trading-native UX |
| Recharts | Idiomatic React; poor candlestick story |
| Chart.js | Needs plugins for candles; less ideal for live trading |
| Highcharts/Highstock | License friction for product |
| Both ECharts + Lightweight | Deferred; start with one chart lib |

## Rules

1. Install `antd` during frontend **project initialization**.
2. Install `lightweight-charts` during init (wrapper shell OK); full candle UI is detail epic.
3. Do not add a second chart library without a new decision doc.
4. Map API candle strings → chart numeric types in `libs/utils` (not inside atoms).

## Doc updates

- Root `AGENTS.md`, `frontend/AGENTS.md`, `frontend/README.md`
- `docs/design/frontend-system-design.md`
- `docs/design/frontend-project-initialization.md`
- `docs/features/multi-exchange-spot-markets.md` (table = Ant Design; charts later note)

