# MAI-2: RTK aiApi chat mutation

| Field | Value |
|---|---|
| **ID** | MAI-2 |
| **Epic** | mobile-ai-chat |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MAI-1 |
| **Path** | `project-management/tasks/mobile/ai-chat/MAI-2.md` |

## Summary

Add `libs/api/endpoints/aiApi.ts` (or inject into baseApi):

- `postAiChat` **mutation** → `POST /api/v1/ai/chat`
- Types from OpenAPI-generated schema
- `rtkErrorMessage` friendly label (e.g. "assistant")
- Export hooks from `libs/api/index.ts`
- Optional tag `AiChat` on baseApi if cache invalidation later needed

No page UI.

## Acceptance

- [x] Typed mutation + exports  
- [x] Body omits empty sessionId only if API allows (prefer always send device session from MAI-3)  
- [x] Unit/smoke test optional for arg builder  
- [x] Status updated  

## Status

`done`
