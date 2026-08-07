package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/aiagent"
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

// Chat handles POST /api/v1/ai/chat
func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		writeError(w, fmt.Errorf("%w: AI service not configured", domain.ErrUpstream))
		return
	}
	var body aiChatBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	msg := strings.TrimSpace(body.Message)
	if msg == "" {
		writeError(w, fmt.Errorf("%w: message is required", domain.ErrInvalidArgument))
		return
	}
	session := strings.TrimSpace(body.SessionID)
	if session == "" {
		session = "http-default"
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()
	res, err := h.client.Chat(ctx, msg, session)
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
