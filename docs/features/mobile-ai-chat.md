# Feature: AI assistant chat (mobile)

**Status:** Implemented  
**Surface:** Product mobile (`mobile/`) — Chrome via react-native-web  
**Backend:** `POST /api/v1/ai/chat` (proxy to Python multi-agent); OpenAPI added in MAI-1  
**Epic:** `project-management/epics/mobile-ai-chat.md`  
**Design:** `docs/design/mobile-ai-chat.md`  
**Tasks:** `project-management/tasks/mobile/ai-chat/MAI-*.md`  
**Related:** `docs/features/ai-assistant.md`

---

## 1. Problem / goal

Mobile users need tool-backed natural language answers about markets (prices, RSI, pumps, news signal) without leaving the app.

---

## 2. Behavior (happy path)

1. User opens **Ask** tab.  
2. Types a question (or arrives from Coin detail with a prefilled draft).  
3. App sends `POST /api/v1/ai/chat` with device `sessionId`.  
4. Shows assistant `reply`; optionally tools used and thinking.  
5. Follow-up messages reuse `sessionId` for in-process memory.  
6. New chat clears messages and rotates/resets session.  
7. Disclaimer always visible.

### Error paths

| Case | UX |
|------|-----|
| AI not configured / 503 | Unavailable empty state + retry |
| Network failure | Keep draft; retry |
| Empty message | Disable send / validation |

---

## 3. APIs

| Method | Path |
|--------|------|
| `POST` | `/api/v1/ai/chat` |

OpenAPI: `backend/api/openapi/openapi.yaml` (after MAI-1).  
Python: `AI_SERVICE_URL` (default `http://127.0.0.1:8090`).

---

## 4. Where the code will live

| Area | Path |
|------|------|
| Page + VM | `mobile/src/modules/ai/pages/ai-chat-page/` (name may vary) |
| Organisms / molecules | `mobile/src/components/{molecules,organisms}/chat-*` |
| RTK | `mobile/src/libs/api/endpoints/aiApi.ts` |
| Session helpers | `mobile/src/libs/utils/` |
| Design | `docs/design/mobile-ai-chat.md` |

---

## 5. How to verify

```bash
cd backend && go run ./cmd/server
# AI service on :8090 with Ollama or Grok
cd mobile && npm run web
# Ask tab → send "What is BTC 24h change on binance?"
```

---

## 6. Known limitations

- Session memory dies when Python process restarts.  
- No streaming in v1.  
- No multi-device sync.  
- Answers are informational only — not financial advice.
