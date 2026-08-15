# aiagent service

Thin HTTP client to the Python multi-agent service (`swyngora_ai.serve`).

Used by Telegram `/ask` and `POST /api/v1/ai/chat` (JSON) plus `POST /api/v1/ai/chat/stream` (NDJSON process events).

When `AI_SERVICE_TOKEN` is set on both sides, the client sends `Authorization: Bearer <token>`. Empty token keeps open localhost (dev). Health probes stay unauthenticated.
