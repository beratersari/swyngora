package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	rr := httptest.NewRecorder()
	h.Chat(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d %s", rr.Code, rr.Body.String())
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
