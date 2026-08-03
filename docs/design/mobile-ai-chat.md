# Design: Mobile AI assistant chat

**Status:** Implemented  
**Epic:** `project-management/epics/mobile-ai-chat.md`  
**Feature:** `docs/features/mobile-ai-chat.md`  
**Tasks:** `project-management/tasks/mobile/ai-chat/MAI-*.md`  
**Analysis:** `project-management/tasks/mobile/ai-chat/MAI-A.md`  
**Branch:** `feature/mobile-ai-chat`  
**Depends on:** Mobile shell + markets/detail (done); Go `POST /api/v1/ai/chat` + Python multi-agent  
**Backend gap:** Route exists but **not yet in OpenAPI** — MAI-1 closes that  

---

## 1. Problem / goal

Users get prices, RSI, and pumps on mobile, but still need **explanations and multi-step analysis** in plain language. The multi-agent assistant already exists (Telegram `/ask`, HTTP proxy). Mobile should expose it as an **Ask** tab with tool-backed answers.

---

## 2. Product framing

| In this epic | Out of scope |
|--------------|--------------|
| Ask tab + chat thread | Streaming tokens (public stream proxy later) |
| Non-stream `POST /api/v1/ai/chat` | Durable multi-user history / auth |
| sessionId multi-turn | Web frontend chat |
| Context prefill from detail | Voice, push, paper trading |
| Tools/thinking optional UI | New LLM vendors |
| 503 when AI off | Changing agent graph |

**Disclaimer:** always show informational / not financial advice (response `note` + static copy).

---

## 3. Backend contract (summary)

Full matrix: **MAI-A**.

| Method | Path | Role |
|--------|------|------|
| `POST` | `/api/v1/ai/chat` | Proxy to Python `/v1/chat` |

```json
// request
{ "message": "BTC RSI on binance 1h?", "sessionId": "mobile-ai-<uuid>" }

// response 200
{
  "reply": "...",
  "sessionId": "mobile-ai-<uuid>",
  "tools": ["market_agent(...)"],
  "thinking": ["plan..."],
  "note": "Informational only — not financial advice."
}
```

| Status | Meaning | Mobile UX |
|--------|---------|-----------|
| 400 | empty message / bad JSON | Inline validation |
| 502 / 503 | AI down or not configured | Unavailable state + retry |

Mobile **never** calls Python `8090` or MCP directly.

---

## 4. UX flows

### Happy path

```text
Open Ask tab
  → empty state + disclaimer
  → type message → Send
  → pending bubble
  → assistant reply (+ optional tools chips)
  → continue multi-turn (same sessionId)
```

### From coin detail

```text
Detail → "Ask about BTCUSDT"
  → navigate AskTab with params
  → composer prefilled (user edits / sends)
```

### AI offline

```text
Send → 503
  → keep user message
  → banner: Assistant unavailable (start AI service / check AI_*)
```

---

## 5. Architecture (mobile)

```text
modules/ai/ (or assistant/)
  pages/ai-chat-page/     # View + ViewModel
components/
  molecules/chat-bubble|chat-composer|chat-disclaimer/
  organisms/chat-message-list/
libs/api/endpoints/aiApi.ts
libs/utils/aiSession.ts + aiContextPrompt.ts
config/aiChatConstants.ts
```

Navigation: add `AskTab` to `MainTabParamList` alongside Home / Markets / Favorites / Pumps.

---

## 6. Ops for local demo

```bash
# terminal 1 — API
cd backend && go run ./cmd/server   # :8080

# terminal 2 — AI (or AI_AUTOSTART=true + AI_PYTHON)
cd ai && pip install -e ".[dev]"
export AI_LLM_PROVIDER=ollama   # or grok + XAI_API_KEY
# start serve process per ai README (port 8090)

# terminal 3
cd mobile && npm run web        # :5180
```

Without AI, mobile must still load; chat shows unavailable.

---

## 7. Testing

| Layer | What |
|-------|------|
| Helpers | sessionId, context prompt |
| ViewModel | send/retry/session reset with mocked mutation |
| Page | empty / pending / error / success smoke |
| Contract | OpenAPI matches handler (MAI-1) |

Never hit live LLM in unit tests.

---

## 8. Task map

| ID | Focus |
|----|--------|
| MAI-A | Field matrix |
| MAI-1 | OpenAPI + codegen |
| MAI-2 | RTK mutation |
| MAI-3 | Session + models |
| MAI-4 | Atomic UI |
| MAI-5 | Tab + page |
| MAI-6 | Context entry points |
| MAI-7 | Error / disclaimer polish |
| MAI-8 | Docs closeout |
