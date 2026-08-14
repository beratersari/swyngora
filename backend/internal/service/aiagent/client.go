// Package aiagent calls the Swyngora Python multi-agent HTTP service.
package aiagent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to the AI chat HTTP API (default http://127.0.0.1:8090).
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New constructs a client. baseURL empty → http://127.0.0.1:8090
func New(baseURL string, timeout time.Duration) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8090"
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ChatRequest is the JSON body for POST /v1/chat.
type ChatRequest struct {
	Message       string `json:"message"`
	SessionID     string `json:"sessionId"`
	ClientID      string `json:"clientId,omitempty"`
	CanTrade      bool   `json:"canTrade"`
	CanManageKeys bool   `json:"canManageKeys"`
}

// ChatOptions controls tool scope for the AI service (read vs trade vs key admin).
type ChatOptions struct {
	CanTrade      bool
	CanManageKeys bool
}

// ChatReference is one public web/X source collected during the turn.
type ChatReference struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Source  string `json:"source"`
	Snippet string `json:"snippet,omitempty"`
}

// ChatResult is the structured assistant result for Telegram/UI.
type ChatResult struct {
	Reply      string
	Tools      []string
	Thinking   []string
	References []ChatReference
}

// StreamEvent is one NDJSON line from /v1/chat/stream.
type StreamEvent struct {
	Type       string          `json:"type"`
	Text       string          `json:"text"`
	Reply      string          `json:"reply"`
	Tools      []string        `json:"tools"`
	Thinking   []string        `json:"thinking"`
	References []ChatReference `json:"references"`
	Message    string          `json:"message"`
	SessionID  string          `json:"sessionId"`
}

// Chat sends a user message (non-streaming). opts nil → full tool access (legacy).
func (c *Client) Chat(ctx context.Context, message, sessionID, clientID string, opts *ChatOptions) (ChatResult, error) {
	var zero ChatResult
	message = strings.TrimSpace(message)
	if message == "" {
		return zero, fmt.Errorf("message is required")
	}
	sessionID = strings.TrimSpace(sessionID)
	clientID = strings.TrimSpace(clientID)
	if sessionID == "" {
		sessionID = clientID
	}
	if sessionID == "" {
		return zero, fmt.Errorf("sessionId is required")
	}
	reqBody := ChatRequest{Message: message, SessionID: sessionID, ClientID: clientID, CanTrade: true, CanManageKeys: true}
	if opts != nil {
		reqBody.CanTrade = opts.CanTrade
		reqBody.CanManageKeys = opts.CanManageKeys
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return zero, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat", bytes.NewReader(body))
	if err != nil {
		return zero, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return zero, fmt.Errorf("AI service unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return zero, err
	}
	var out struct {
		Reply      string          `json:"reply"`
		SessionID  string          `json:"sessionId"`
		Tools      []string        `json:"tools"`
		Thinking   []string        `json:"thinking"`
		References []ChatReference `json:"references"`
		Error      string          `json:"error"`
		Message    string          `json:"message"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return zero, fmt.Errorf("AI bad response: %s", truncate(string(raw), 200))
	}
	if resp.StatusCode >= 400 {
		msg := out.Message
		if msg == "" {
			msg = out.Error
		}
		if msg == "" {
			msg = string(raw)
		}
		return zero, fmt.Errorf("AI error (%d): %s", resp.StatusCode, truncate(msg, 300))
	}
	if strings.TrimSpace(out.Reply) == "" {
		return zero, fmt.Errorf("AI returned empty reply")
	}
	return ChatResult{Reply: out.Reply, Tools: out.Tools, Thinking: out.Thinking, References: out.References}, nil
}

// ChatStream streams NDJSON events. onEvent is called for each event (including final).
// Returns the final ChatResult when type=final is seen. opts nil → full tool access.
func (c *Client) ChatStream(ctx context.Context, message, sessionID, clientID string, opts *ChatOptions, onEvent func(StreamEvent)) (ChatResult, error) {
	var zero ChatResult
	message = strings.TrimSpace(message)
	if message == "" {
		return zero, fmt.Errorf("message is required")
	}
	sessionID = strings.TrimSpace(sessionID)
	clientID = strings.TrimSpace(clientID)
	if sessionID == "" {
		sessionID = clientID
	}
	if sessionID == "" {
		return zero, fmt.Errorf("sessionId is required")
	}
	reqBody := ChatRequest{Message: message, SessionID: sessionID, ClientID: clientID, CanTrade: true, CanManageKeys: true}
	if opts != nil {
		reqBody.CanTrade = opts.CanTrade
		reqBody.CanManageKeys = opts.CanManageKeys
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return zero, err
	}
	// Streaming client must not use global Timeout that kills long streams mid-way;
	// use context deadline only.
	httpClient := &http.Client{Timeout: 0}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/stream", bytes.NewReader(body))
	if err != nil {
		return zero, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")
	resp, err := httpClient.Do(req)
	if err != nil {
		return zero, fmt.Errorf("AI service unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return zero, fmt.Errorf("AI stream error (%d): %s", resp.StatusCode, truncate(string(raw), 300))
	}

	var final ChatResult
	sc := bufio.NewScanner(resp.Body)
	// larger lines for tool previews
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ev StreamEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if onEvent != nil {
			onEvent(ev)
		}
		switch ev.Type {
		case "final":
			final.Reply = ev.Reply
			final.Tools = ev.Tools
			final.Thinking = ev.Thinking
			final.References = ev.References
		case "error":
			msg := ev.Message
			if msg == "" {
				msg = ev.Text
			}
			return zero, fmt.Errorf("AI error: %s", truncate(msg, 400))
		case "done":
			// end
		}
	}
	if err := sc.Err(); err != nil {
		if final.Reply != "" {
			return final, nil
		}
		return zero, err
	}
	if strings.TrimSpace(final.Reply) == "" {
		return zero, fmt.Errorf("AI returned empty reply")
	}
	return final, nil
}

// Healthy probes GET /health on the AI service.
func (c *Client) Healthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
