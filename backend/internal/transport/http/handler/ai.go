package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/aiagent"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/http/middleware"
)

// AIHandler proxies chat to the Python multi-agent service.
type AIHandler struct {
	client  *aiagent.Client
	timeout time.Duration
}

// NewAIHandler constructs the handler. client may be nil → 503.
func NewAIHandler(client *aiagent.Client, timeout time.Duration) *AIHandler {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &AIHandler{client: client, timeout: timeout}
}

type aiChatBody struct {
	Message   string `json:"message"`
	SessionID string `json:"sessionId"`
}

func (h *AIHandler) parseChat(r *http.Request) (msg, session, clientID string, opts *aiagent.ChatOptions, err error) {
	if h.client == nil {
		return "", "", "", nil, fmt.Errorf("%w: AI service not configured", domain.ErrUpstream)
	}
	var body aiChatBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		return "", "", "", nil, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument)
	}
	msg = strings.TrimSpace(body.Message)
	if msg == "" {
		return "", "", "", nil, fmt.Errorf("%w: message is required", domain.ErrInvalidArgument)
	}
	clientID, err = domain.NormalizeClientID(clientIDFrom(r))
	if err != nil {
		return "", "", "", nil, err
	}
	session = strings.TrimSpace(body.SessionID)
	if session == "" {
		session = clientID
	}
	opts = chatOptionsFromRequest(r)
	return msg, session, clientID, opts, nil
}

// chatOptionsFromRequest maps API key identity to AI tool scope.
// Master/open identities keep full access. User keys never manage other keys;
// read-only user keys cannot invoke mutating portfolio/alert tools via AI.
func chatOptionsFromRequest(r *http.Request) *aiagent.ChatOptions {
	id := middleware.IdentityFrom(r.Context())
	if id == nil || !id.UserKey {
		return &aiagent.ChatOptions{CanTrade: true, CanManageKeys: true}
	}
	return &aiagent.ChatOptions{CanTrade: id.CanTrade, CanManageKeys: false}
}

// Chat handles POST /api/v1/ai/chat
func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	msg, session, clientID, opts, err := h.parseChat(r)
	if err != nil {
		writeError(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()
	res, err := h.client.Chat(ctx, msg, session, clientID, opts)
	if err != nil {
		writeError(w, fmt.Errorf("%w: %v", domain.ErrUpstream, err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"reply":      res.Reply,
		"sessionId":  session,
		"tools":      res.Tools,
		"thinking":   res.Thinking,
		"references": res.References,
		"note":       "Informational only — not financial advice.",
	})
}

// ChatStream handles POST /api/v1/ai/chat/stream (NDJSON thinking/tool/final events).
func (h *AIHandler) ChatStream(w http.ResponseWriter, r *http.Request) {
	msg, session, clientID, opts, err := h.parseChat(r)
	if err != nil {
		writeError(w, err)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, fmt.Errorf("%w: streaming not supported", domain.ErrUpstream))
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	writeEv := func(v any) {
		_ = enc.Encode(v)
		flusher.Flush()
	}
	writeEv(map[string]string{"type": "status", "text": "Planning…", "sessionId": session})

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()
	var sawFinal bool
	res, err := h.client.ChatStream(ctx, msg, session, clientID, opts, func(ev aiagent.StreamEvent) {
		if ev.Type == "" {
			return
		}
		if ev.Type == "final" {
			sawFinal = true
		}
		writeEv(ev)
	})
	if err != nil {
		writeEv(map[string]string{"type": "error", "message": err.Error(), "sessionId": session})
		writeEv(map[string]string{"type": "done", "sessionId": session})
		return
	}
	if !sawFinal {
		writeEv(map[string]any{
			"type":       "final",
			"reply":      res.Reply,
			"tools":      res.Tools,
			"thinking":   res.Thinking,
			"references": res.References,
			"sessionId":  session,
			"note":       "Informational only — not financial advice.",
		})
	}
	writeEv(map[string]string{"type": "done", "sessionId": session})
}
