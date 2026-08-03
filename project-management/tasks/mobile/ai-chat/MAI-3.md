# MAI-3: sessionId + message model helpers

| Field | Value |
|---|---|
| **ID** | MAI-3 |
| **Epic** | mobile-ai-chat |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MAI-2 (or parallel after MAI-A) |
| **Path** | `project-management/tasks/mobile/ai-chat/MAI-3.md` |

## Summary

Pure helpers + light persistence (no UI):

- `getOrCreateAiSessionId()` → stable `mobile-ai-<uuid>` (localStorage on web; never empty / never shared `"default"` / `"http-default"` alone if avoidable)
- Message view types: `role: 'user' | 'assistant' | 'system'`, `id`, `text`, optional `tools`, `thinking`, `createdAt`
- `buildContextPrompt({ exchange, symbol, interval? })` for prefill from detail
- Constants in `config/aiChatConstants.ts` (storage keys, max history display length)

Reuse patterns from watchlist `clientId` storage adapter if present.

## Acceptance

- [x] Unit tests for session id + context prompt builder  
- [x] No React components in helpers  
- [x] Status updated  

## Status

`done`
