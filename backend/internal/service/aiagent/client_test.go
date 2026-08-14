package aiagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_Chat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat" {
			http.NotFound(w, r)
			return
		}
		var req ChatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Message == "" {
			t.Fatal("empty message")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"reply":     "hello " + req.Message,
			"sessionId": req.SessionID,
			"tools":     []string{"t1"},
			"thinking":  []string{"think"},
			"references": []map[string]string{
				{"title": "Bitcoin", "url": "https://coinmarketcap.com/currencies/bitcoin/", "source": "web"},
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, 5*time.Second)
	res, err := c.Chat(context.Background(), "world", "s1", "client-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Reply != "hello world" {
		t.Fatalf("reply=%q", res.Reply)
	}
	if len(res.Tools) != 1 || res.Tools[0] != "t1" {
		t.Fatalf("tools=%v", res.Tools)
	}
	if len(res.References) != 1 || res.References[0].URL == "" {
		t.Fatalf("references=%v", res.References)
	}
}

func TestClient_ChatStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/stream" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		flusher, _ := w.(http.Flusher)
		lines := []string{
			`{"type":"status","text":"Planning…"}`,
			`{"type":"tool","text":"market_agent(task=BTC)"}`,
			`{"type":"tool_result","text":"get_ticker ✓ ok"}`,
			`{"type":"final","reply":"BTC is up","tools":["market_agent(task=BTC)"],"thinking":["plan"]}`,
			`{"type":"done"}`,
		}
		for _, line := range lines {
			_, _ = w.Write([]byte(line + "\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer srv.Close()

	c := New(srv.URL, 5*time.Second)
	var seen []string
	res, err := c.ChatStream(context.Background(), "BTC?", "s1", "client-1", nil, func(ev StreamEvent) {
		seen = append(seen, ev.Type+":"+ev.Text+ev.Reply)
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Reply != "BTC is up" {
		t.Fatalf("reply=%q", res.Reply)
	}
	joined := strings.Join(seen, "|")
	if !strings.Contains(joined, "tool:market_agent") {
		t.Fatalf("expected live tool event, got %v", seen)
	}
	if !strings.Contains(joined, "final:BTC is up") && !strings.Contains(joined, "BTC is up") {
		t.Fatalf("expected final in events: %v", seen)
	}
}

func TestClient_Unreachable(t *testing.T) {
	c := New("http://127.0.0.1:1", 200*time.Millisecond)
	_, err := c.Chat(context.Background(), "x", "s", "c", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClient_ChatForwardsScope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.CanTrade || req.CanManageKeys {
			t.Fatalf("want read-only scope, got trade=%v keys=%v", req.CanTrade, req.CanManageKeys)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"reply": "ok", "sessionId": req.SessionID})
	}))
	defer srv.Close()
	c := New(srv.URL, 5*time.Second)
	_, err := c.Chat(context.Background(), "hi", "s", "c", &ChatOptions{CanTrade: false, CanManageKeys: false})
	if err != nil {
		t.Fatal(err)
	}
}
