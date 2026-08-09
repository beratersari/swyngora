package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/apikey"
)

func TestAPIKeyHTTP_CreateListRevoke(t *testing.T) {
	h := NewAPIKeyHandler(apikey.New(accountstore.NewMemory()))
	body, _ := json.Marshal(map[string]any{"clientId": "c1", "name": "Bot", "permission": "trade"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/account/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	if created["secret"] == nil || created["id"] == nil || created["permission"] != "trade" {
		t.Fatalf("%v", created)
	}
	id := created["id"].(string)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/account/api-keys?clientId=c1", nil)
	rr = httptest.NewRecorder()
	h.List(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list %d %s", rr.Code, rr.Body.String())
	}
	var list map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	if int(list["count"].(float64)) != 1 {
		t.Fatalf("%v", list)
	}
	keys := list["keys"].([]any)
	if keys[0].(map[string]any)["secret"] != nil {
		t.Fatal("list must not include secret")
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/account/api-keys/"+id+"?clientId=c1", nil)
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	h.Revoke(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("revoke %d %s", rr.Code, rr.Body.String())
	}
}
