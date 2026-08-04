package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/scannerstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/scanner"
)

type scanCandles struct{}

func (scanCandles) GetCandles(_ context.Context, _, _, _ string, _ int, _, _ *time.Time) ([]domain.Candle, error) {
	return nil, nil
}

type scanWatch struct{}

func (scanWatch) Get(_ context.Context, actorID, ownerID string) (*domain.WatchlistAccess, error) {
	id := ownerID
	if id == "" {
		id = actorID
	}
	return &domain.WatchlistAccess{
		Watchlist: domain.Watchlist{ClientID: id, Items: nil},
		OwnerClientID: id,
		Role: domain.WatchlistRoleOwner,
	}, nil
}

func TestScannerHTTP_CreateListDelete(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	h := NewScannerHandler(scanner.New(st, scanCandles{}, scanWatch{}))

	body, _ := json.Marshal(map[string]any{
		"clientId": "http-scan", "type": "rsi", "interval": "1h",
		"rsiPeriod": 14, "rsiCondition": "below", "rsiThreshold": 30,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scanner/rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	id := created["id"].(string)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/scanner/rules?clientId=http-scan", nil)
	rr = httptest.NewRecorder()
	h.ListRules(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/scanner/rules/"+id+"?clientId=http-scan", nil)
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	h.DeleteRule(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete %d %s", rr.Code, rr.Body.String())
	}
}
