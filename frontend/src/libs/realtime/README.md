# libs/realtime

WebSocket client for live **price ticks** and **paper portfolio** order/position updates.

- Protocol: `GET /api/v1/ws` (see `GET /api/v1/realtime` and `docs/features/realtime.md`)
- One shared connection; pages subscribe/unsubscribe selected coins and one book
- After disconnect, the client reconnects and resends subscriptions
- Incoming events patch RTK Query caches (`getTicker24h`, `listSpotMarkets`, `getPortfolio`)

REST polling remains a fallback while the socket is down.
