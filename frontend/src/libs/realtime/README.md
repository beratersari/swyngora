# libs/realtime

WebSocket client for live **price ticks** and **paper portfolio** order/position updates.

- Protocol: `GET /api/v1/ws` (see `GET /api/v1/realtime` and `docs/features/realtime.md`)
- Locked APIs: mint `POST /api/v1/realtime/ticket` then connect with `?ticket=` (never `?token=`)
- One shared connection; pages subscribe/unsubscribe selected coins and one book
- After disconnect, the client reconnects and resends subscriptions
- Incoming events patch RTK Query caches (`getTicker24h`, `listSpotMarkets`, `getPortfolio`)
- **Venue isolation:** `listSpotMarkets` patches only caches whose query `exchange` matches the tick (list items have no `exchange` field)
- **Mcap while live:** price ticks scale circulating/total/max mcap by last-price ratio; markets/watchlist keep a slower REST poll (`SPOT_LIST_WS_REST_POLL_MS`) so derived fields do not freeze forever
- Desk chrome **Live** requires the WebSocket connected; API-up / stream-down shows **Delayed**

REST polling remains a fallback while the socket is down (fast interval); while WS is up, list endpoints use the slower REST interval above.
