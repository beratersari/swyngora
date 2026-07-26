# features/markets

Multi-exchange spot markets **UI feature** (Epic B).

| Path                        | Role                            |
| --------------------------- | ------------------------------- |
| `components/ExchangeTabs`   | Exchange switcher               |
| `components/MarketsToolbar` | Search, quote, tags             |
| `components/MarketsTable`   | Sortable Ant Table + pagination |

**Data:** `@/libs/api` (`listExchanges`, `listProductTags`, `listSpotMarkets`).  
**Page wiring:** `components/pages/MarketsPage`.
