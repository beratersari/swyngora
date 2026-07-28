# Epic I: Mobile AI assistant chat

**Priority:** P1 (mobile track — next after batch indicators)  
**Status:** done  
**Depends on:** Epics C–H (markets, detail, favorites, pumps, batch) — done; backend `POST /api/v1/ai/chat` + Python multi-agent service  
**Branch:** `feature/mobile-ai-chat` (from latest `develop`)  
**Design plan:** `docs/design/mobile-ai-chat.md`  
**Feature:** `docs/features/mobile-ai-chat.md`  
**Analysis:** `project-management/tasks/mobile/ai-chat/MAI-A.md`  
**Tasks folder:** `project-management/tasks/mobile/ai-chat/`

## Goal

Give mobile users an **Ask** experience: natural-language market questions answered by the existing multi-agent stack (orchestrator + market/web/X/analyst specialists) via the Go proxy  
`POST /api/v1/ai/chat`. Chat is **Atomic Design** + **View + ViewModel**; primary runtime **Chrome** (`npm run web`).

Market numbers must come from tools (not hallucinated). Surface the API **not financial advice** note.

## APIs

| Layer | Path | Status for mobile |
|-------|------|-------------------|
| Public HTTP (proxy) | `POST /api/v1/ai/chat` | Implemented in Go; **missing OpenAPI** → MAI-1 |
| Python service | `POST /v1/chat` (and stream) | Behind proxy; mobile talks only to Go |
| MCP tools | market tools at `/mcp` | Used by agents; not called from mobile UI |

### Request / response (handler today)

| Field | Side | Notes |
|-------|------|--------|
| `message` | body required | Non-empty trimmed string |
| `sessionId` | body optional | Default `http-default` server-side if empty; mobile generates stable device session |
| `reply` | response | Assistant text |
| `sessionId` | response | Echo / assigned session |
| `tools` | response | Tool names used (optional chips) |
| `thinking` | response | Plan/thinking lines (optional collapsible) |
| `note` | response | Informational disclaimer |

AI unavailable → **503** (client not configured / upstream down).

## Tasks

- [x] MAI-A — API field matrix analysis  
- [x] MAI-1 — OpenAPI for AI chat + client codegen  
- [x] MAI-2 — RTK `aiApi` chat mutation  
- [x] MAI-3 — sessionId + message model helpers  
- [x] MAI-4 — Atomic chat UI (bubbles, composer, list)  
- [x] MAI-5 — Ask tab + AiChatPage ViewModel  
- [x] MAI-6 — Context chips from Markets / Detail / Pumps  
- [x] MAI-7 — Loading / 503 / error / disclaimer UX  
- [x] MAI-8 — Docs + board + changelog closeout  

## Acceptance

- **Ask** tab opens a chat screen  
- User sends a message → sees assistant reply (or clear unavailable/error)  
- `sessionId` persists for multi-turn context within the session  
- Optional tools/thinking UI without cluttering the main reply  
- Prefill from coin detail / markets context works  
- “Not financial advice” always visible for market answers  
- OpenAPI documents the route; mobile types from codegen  
- Atomic-only UI under `src/components/`; RTK under `libs/api`  
- Tests for helpers + ViewModel; docs closed  

## Out of scope

- Streaming UI (`/v1/chat/stream` not on public OpenAPI proxy yet)  
- Durable multi-device conversation history / auth accounts  
- Web (`frontend/`) chat UI (separate backlog)  
- Changing agent graph, LLM providers, or MCP tool set (except if OpenAPI/docs only)  
- Paper trading, alerts, voice input  

## Dependencies / ops

| Env | Purpose |
|-----|---------|
| `AI_SERVICE_URL` | Python service (default `http://127.0.0.1:8090`) |
| `AI_AUTOSTART` / `AI_PYTHON` | Optional child process start from Go server |
| `AI_LLM_PROVIDER` | `ollama` or `grok` (Python) |
| `XAI_API_KEY` | When provider is grok |

Manual verify needs backend + AI service (or graceful 503 UX when AI is off).

## MR grouping

`(A+1)` · `(2–3)` · `(4–5)` · `(6–7)` · `(8)`
