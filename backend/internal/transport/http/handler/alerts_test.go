package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/alertstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
)

func newAlertHandler(t *testing.T) *AlertHandler {
	t.Helper()
	store, err := alertstore.Open(filepath.Join(t.TempDir(), "alerts.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	svc := pricealert.New(store)
	// Handler tests use example.com hosts without requiring live DNS; production keeps SSRF on.
	svc.AllowPrivateWebhooks = true
	return NewAlertHandler(svc)
}

func TestAlertHTTP_CreateListGetDelete(t *testing.T) {
	h := newAlertHandler(t)

	body, _ := json.Marshal(map[string]any{
		"clientId":    "http-user",
		"exchange":    "binance",
		"symbol":      "BTCUSDT",
		"condition":   "above",
		"targetPrice": 95000.5,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}
	var created alertDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Status != "active" || created.TargetPrice != 95000.5 {
		t.Fatalf("%+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/alerts?clientId=http-user", nil)
	rr = httptest.NewRecorder()
	h.List(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d", rr.Code)
	}
	var listBody map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &listBody)
	if int(listBody["count"].(float64)) != 1 {
		t.Fatalf("%v", listBody)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/alerts/"+created.ID+"?clientId=http-user", nil)
	req.SetPathValue("id", created.ID)
	rr = httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/alerts/"+created.ID+"?clientId=http-user", nil)
	req.SetPathValue("id", created.ID)
	rr = httptest.NewRecorder()
	h.Delete(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status=%d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/alerts/"+created.ID+"?clientId=http-user", nil)
	req.SetPathValue("id", created.ID)
	rr = httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404 after delete, got %d", rr.Code)
	}
}

func TestAlertHTTP_Validation(t *testing.T) {
	h := newAlertHandler(t)
	body, _ := json.Marshal(map[string]any{
		"clientId": "u", "symbol": "BTCUSDT", "condition": "nope", "targetPrice": 1,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}
func TestAlertHTTP_WebhookCRUD(t *testing.T) {
	h := newAlertHandler(t)
	body, _ := json.Marshal(map[string]any{
		"clientId":     "wh-user",
		"url":          "https://hooks.example.com/a",
		"deliveryMode": "hourly_digest",
		"timeZone":     "UTC",
		"quietHours": map[string]any{
			"enabled": true,
			"start":   "22:00",
			"end":     "08:00",
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.PutWebhook(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("put status=%d body=%s", rr.Code, rr.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/alerts/webhook?clientId=wh-user", nil)
	rr = httptest.NewRecorder()
	h.GetWebhook(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status=%d", rr.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["url"] != "https://hooks.example.com/a" || got["configured"] != true || got["deliveryMode"] != "hourly_digest" {
		t.Fatalf("%v", got)
	}
	if got["timeZone"] != "UTC" {
		t.Fatalf("timeZone=%v", got["timeZone"])
	}
	qh, _ := got["quietHours"].(map[string]any)
	if qh == nil || qh["enabled"] != true || qh["start"] != "22:00" || qh["end"] != "08:00" {
		t.Fatalf("quietHours=%v", got["quietHours"])
	}
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/alerts/webhook?clientId=wh-user", nil)
	rr = httptest.NewRecorder()
	h.DeleteWebhook(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status=%d", rr.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/alerts/webhook?clientId=wh-user", nil)
	rr = httptest.NewRecorder()
	h.GetWebhook(rr, req)
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["configured"] != false {
		t.Fatalf("%v", got)
	}
}
