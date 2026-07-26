# Frontend chart libraries comparison

**Decision:** TradingView **Lightweight Charts** (with Ant Design for UI chrome).  
See `project-management/decisions/001-antd-and-lightweight-charts.md`.

## Comparison (reference)

| Library | Candlesticks | React fit | Bundle / perf | License | Notes |
|---|---|---|---|---|---|
| **Lightweight Charts** ★ | Excellent | Imperative API + thin React wrapper | Small, fast | Apache-2.0 | Best for trading OHLCV |
| Apache ECharts | Good | `echarts-for-react` | Larger | Apache-2.0 | Strong multi-chart generalist |
| Recharts | Weak | Excellent | Medium | MIT | Dashboards, not primary market candles |
| Chart.js | Plugin-based | `react-chartjs-2` | Medium | MIT | Extra plugins for finance |
| Visx | DIY | Excellent | Depends | MIT | Low-level; more engineering |
| uPlot | Possible | Thin wrapper | Very small | MIT | Fast; less docs/ecosystem |
| Highcharts Stock | Excellent | Official React | — | **Commercial** | Avoid without ADR |

★ = selected for Swyngora product frontend.
