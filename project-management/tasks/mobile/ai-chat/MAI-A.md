# MAI-A: AI chat API field matrix (analysis)

| Field | Value |
|---|---|
| **ID** | MAI-A |
| **Epic** | mobile-ai-chat |
| **Status** | done |
| **Area** | mobile / analysis |
| **Path** | `project-management/tasks/mobile/ai-chat/MAI-A.md` |

## Purpose

Map backend AI chat HTTP contract to mobile UI needs.

Sources:

- Handler: `backend/internal/transport/http/handler/ai.go`
- Router: `backend/internal/transport/http/router.go` (`POST /api/v1/ai/chat`)
- Client: `backend/internal/service/aiagent/client.go` (`Chat`, `ChatStream`)
- Config: `backend/.env.example` (`AI_*`)
- Feature: `docs/features/ai-assistant.md`
- OpenAPI (after MAI-1): `postAiChat`

---

## 1. Endpoints

| Method | Path | operationId | Mobile use |
|--------|------|-------------|------------|
| `POST` | `/api/v1/ai/chat` | `postAiChat` | Ask tab chat (non-streaming) |

Python upstream (not called from mobile):

| Method | Path | Notes |
|--------|------|--------|
| `POST` | `/v1/chat` | JSON reply |
| `POST` | `/v1/chat/stream` | NDJSON — **not proxied** on public HTTP; out of mobile v1 |

---

## 2. Request body

| Field | Required | Type | UI |
|-------|----------|------|-----|
| `message` | **yes** | string (non-empty after trim) | Composer |
| `sessionId` | no | string | Device `mobile-ai-<uuid>`; server defaults to `http-default` if empty |

---

## 3. Response 200

| Field | Type | UI |
|-------|------|-----|
| `reply` | string | Assistant bubble |
| `sessionId` | string | Persist for multi-turn |
| `tools` | string[] optional | Chips under reply |
| `thinking` | string[] optional | Optional expand (v1: omit main UI clutter) |
| `note` | string optional | Disclaimer footer |

---

## 4. Errors

| Status | Cause | Mobile UX |
|--------|-------|-----------|
| 400 | empty message / bad JSON | Validation / disable send |
| 502 | upstream AI error | “upstream error” + retry |
| 503 | AI client nil / not configured | “Assistant unavailable” + ops hint |
| network | backend down | Network error + retry |

---

## 5. Session semantics

- Multi-turn memory lives in the **Python process** keyed by `sessionId`.
- Mobile generates stable `mobile-ai-<uuid>` (not `default` / `http-default`).
- “New chat” rotates session id and clears local transcript.
- Process restart clears server memory.

---

## 6. Mobile must not call

- Python port `8090` directly
- MCP `/mcp` from the UI
- Streaming path until public OpenAPI proxy exists

---

## 7. OpenAPI proposal (MAI-1)

- Tag `AI`
- Schemas `AiChatRequest` / `AiChatResponse`
- Errors 400 / 502 / 503

## Status

`done`
