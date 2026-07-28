# MAI-1: OpenAPI for AI chat + client codegen

| Field | Value |
|---|---|
| **ID** | MAI-1 |
| **Epic** | mobile-ai-chat |
| **Status** | done |
| **Area** | backend + mobile/frontend codegen |
| **Depends on** | MAI-A |
| **Path** | `project-management/tasks/mobile/ai-chat/MAI-1.md` |

## Summary

Document `POST /api/v1/ai/chat` in `backend/api/openapi/openapi.yaml` so clients follow §6.9:

- operationId e.g. `postAiChat`
- Request: `{ message: string, sessionId?: string }`
- Response: `{ reply, sessionId, tools?, thinking?, note? }`
- Errors: 400 invalid body; 502/503 upstream / not configured
- Tag: `AI` (or existing convention)

Then regenerate:

```bash
cd mobile && npm run codegen:api
# also frontend if monorepo script expects both
```

**Do not hand-edit** `libs/api/generated/schema.d.ts`.

No mobile UI in this task.

## Acceptance

- [x] OpenAPI path matches handler  
- [x] Codegen updates mobile generated schema  
- [x] Spec validates / examples reasonable  
- [x] Status updated  

## Status

`done`
