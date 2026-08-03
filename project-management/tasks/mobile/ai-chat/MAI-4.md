# MAI-4: Atomic chat UI (bubbles, composer, list)

| Field | Value |
|---|---|
| **ID** | MAI-4 |
| **Epic** | mobile-ai-chat |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MAI-3 types (can stub props) |
| **Path** | `project-management/tasks/mobile/ai-chat/MAI-4.md` |

## Summary

Presentational Atomic components only (kebab-case folders under `src/components/`):

| Level | Component | Role |
|-------|-----------|------|
| Molecule | `chat-bubble` | User / assistant message bubble |
| Molecule | `chat-composer` | Text input + Send (disabled while pending) |
| Organism | `chat-message-list` | Scrollable message list + empty state |
| Molecule/Organism | `chat-tools-chips` (optional) | Show `tools[]` used |
| Molecule | `chat-disclaimer` | Static “not financial advice” |

File split: `*.types.ts`, `*.styles.ts`, `index.ts`; tests for pure bits.

**No** network calls inside atoms/molecules.

## Acceptance

- [x] Components render from props only  
- [x] Matches brand tokens  
- [x] No `modules/*/components`  
- [x] Status updated  

## Status

`done`
