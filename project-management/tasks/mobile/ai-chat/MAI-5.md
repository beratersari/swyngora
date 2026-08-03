# MAI-5: Ask tab + AiChatPage ViewModel

| Field | Value |
|---|---|
| **ID** | MAI-5 |
| **Epic** | mobile-ai-chat |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MAI-2, MAI-3, MAI-4 |
| **Path** | `project-management/tasks/mobile/ai-chat/MAI-5.md` |

## Summary

Wire product chat:

- Module: `modules/ai/` (or `modules/assistant/`) with `pages/ai-chat-page/`
- View + ViewModel: local message list, pending state, `postAiChat` mutation
- Navigation: **Ask** bottom tab → stack → `AiChat` screen
- Icons: Lucide (e.g. `MessageCircle` / `Sparkles`) consistent with existing tabs
- On send: append user message → mutation → append assistant (or error banner)
- Clear / new session action resets sessionId + messages

## Acceptance

- [x] Tab visible and navigable  
- [x] Multi-turn keeps same sessionId until reset  
- [x] ViewModel unit/integration tests with mocked mutation  
- [x] Status updated  

## Status

`done`
