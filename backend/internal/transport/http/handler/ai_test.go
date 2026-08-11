package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/aiagent"
)

func TestAIHandler_Chat(t *testing.T) {
	// mock AI service via httptest wrapped in aiagent.Client
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"reply": "ok from ai", "sessionId": "s"})
	}))
	defer aiSrv.Close()

	h := NewAIHandler(aiagent.New(aiSrv.URL, 5*time.Second), 5*time.Second)
	body, _ := json.Marshal(map[string]string{"message": "hi", "sessionId": "s"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	req.Header.Set("X-Client-Id", "web-user-1")
	rr := httptest.NewRecorder()
	h.Chat(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d %s", rr.Code, rr.Body.String())
	}
}

func TestAIHandler_ChatStream(t *testing.T) {
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/stream" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		fl, _ := w.(http.Flusher)
		lines := []string{
			`{"type":"status","text":"Planning…"}`,
			`{"type":"thinking","text":"Need BTC RSI"}`,
			`{"type":"tool","text":"market_agent(task=RSI)"}`,
			`{"type":"final","reply":"RSI is 55","tools":["market_agent"],"thinking":["Need BTC RSI"]}`,
			`{"type":"done"}`,
		}
		for _, line := range lines {
			_, _ = w.Write([]byte(line + "\n"))
			if fl != nil {
				fl.Flush()
			}
		}
	}))
	defer aiSrv.Close()

	h := NewAIHandler(aiagent.New(aiSrv.URL, 5*time.Second), 5*time.Second)
	body, _ := json.Marshal(map[string]string{"message": "rsi?", "sessionId": "s"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat/stream", bytes.NewReader(body))
	req.Header.Set("X-Client-Id", "web-user-1")
	rr := httptest.NewRecorder()
	h.ChatStream(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d %s", rr.Code, rr.Body.String())
	}
	out := rr.Body.String()
	if !strings.Contains(out, `"type":"thinking"`) || !strings.Contains(out, "RSI is 55") {
		t.Fatalf("stream body=%s", out)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "ndjson") {
		t.Fatalf("content-type=%s", ct)
	}
}

func TestAIHandler_NotConfigured(t *testing.T) {
	h := NewAIHandler(nil, 0)
	body, _ := json.Marshal(map[string]string{"message": "hi"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Chat(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d", rr.Code)
	}
}
