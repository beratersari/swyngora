# MAI-6: Context chips from Markets / Detail / Pumps

| Field | Value |
|---|---|
| **ID** | MAI-6 |
| **Epic** | mobile-ai-chat |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MAI-5 |
| **Path** | `project-management/tasks/mobile/ai-chat/MAI-6.md` |

## Summary

Deep-link / prefill into Ask:

- Coin detail: “Ask AI about {symbol}” → navigate to Ask with params  
  `exchange`, `symbol`, optional `interval`  
- Optional Markets / Pumps: shorter chip “Explain this pump” / “What is RSI saying?”
- ViewModel builds initial draft via `buildContextPrompt` (user can edit before send)
- Params cleared after apply so remounts don’t loop

Prefer React Navigation params over global mutable store.

## Acceptance

- [x] Detail entry point works  
- [x] Prefill text is editable  
- [x] No forced auto-send without user action  
- [x] Tests for param → draft mapping  
- [x] Status updated  

## Status

`done`
