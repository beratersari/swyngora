# MAI-7: Loading / 503 / error / disclaimer UX

| Field | Value |
|---|---|
| **ID** | MAI-7 |
| **Epic** | mobile-ai-chat |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MAI-5 |
| **Path** | `project-management/tasks/mobile/ai-chat/MAI-7.md` |

## Summary

Production-ready failure and trust UX:

| Case | UX |
|------|-----|
| Pending | Disable send; typing / skeleton bubble |
| 503 / AI not configured | Clear empty-state: “Assistant unavailable” + retry |
| 400 | “Message required” / validation |
| Network / timeout | Retry; keep user message |
| Success with note | Always show disclaimer footer |
| Long reply | Scroll to end; no crash on large text |

No silent empty screens. Copy stays informational, not “guaranteed signals.”

## Acceptance

- [x] All cases covered in View or ViewModel tests  
- [x] Disclaimer always present on chat screen  
- [x] Status updated  

## Status

`done`
