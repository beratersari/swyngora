package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/swing"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

func TestCustomSignalFactorsAndProfileHTTP(t *testing.T) {
	m := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{domain.ExchangeBinance: swingPort{}}, swingSupply{})
	store := &memSwingProfiles{byID: map[string]domain.CustomSignalProfile{}}
	svc := swing.New(m, watchlist.New(watchliststore.NewMemory())).WithProfiles(store)
	h := NewSwingHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/swing/custom/factors", nil)
	rec := httptest.NewRecorder()
	h.ListCustomFactors(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("factors %d %s", rec.Code, rec.Body.String())
	}
	var cat map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &cat); err != nil {
		t.Fatal(err)
	}
	factors, _ := cat["factors"].([]any)
	if len(factors) != len(domain.CustomSignalFactorIDs) {
		t.Fatalf("factors=%v", cat)
	}

	body := []byte(`{"name":"Hunt lean","weights":{"hunt_up":70,"rsi_recovery":30}}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/swing/custom/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "user-http")
	rec = httptest.NewRecorder()
	h.CreateCustomProfile(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	var created customSignalProfileDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Weights["hunt_up"] != 70 {
		t.Fatalf("%+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/swing/custom/profiles", nil)
	req.Header.Set("X-Client-Id", "user-http")
	rec = httptest.NewRecorder()
	h.ListCustomProfiles(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list %d %s", rec.Code, rec.Body.String())
	}

	bad := []byte(`{"name":"bad","weights":{"hunt_up":10}}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/swing/custom/profiles", bytes.NewReader(bad))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "user-http")
	rec = httptest.NewRecorder()
	h.CreateCustomProfile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for bad weights, got %d %s", rec.Code, rec.Body.String())
	}
}

type memSwingProfiles struct {
	byID map[string]domain.CustomSignalProfile
}

func (m *memSwingProfiles) CreateProfile(_ context.Context, p domain.CustomSignalProfile) (*domain.CustomSignalProfile, error) {
	m.byID[p.ID] = p
	return &p, nil
}
func (m *memSwingProfiles) GetProfile(_ context.Context, clientID, id string) (*domain.CustomSignalProfile, error) {
	p, ok := m.byID[id]
	if !ok || p.ClientID != clientID {
		return nil, domain.ErrNotFound
	}
	return &p, nil
}
func (m *memSwingProfiles) UpdateProfile(_ context.Context, p domain.CustomSignalProfile) (*domain.CustomSignalProfile, error) {
	m.byID[p.ID] = p
	return &p, nil
}
func (m *memSwingProfiles) ListProfiles(_ context.Context, clientID string) ([]domain.CustomSignalProfile, error) {
	out := []domain.CustomSignalProfile{}
	for _, p := range m.byID {
		if p.ClientID == clientID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (m *memSwingProfiles) DeleteProfile(_ context.Context, clientID, id string) error {
	p, ok := m.byID[id]
	if !ok || p.ClientID != clientID {
		return domain.ErrNotFound
	}
	delete(m.byID, id)
	return nil
}
func (m *memSwingProfiles) CountProfiles(_ context.Context, clientID string) (int, error) {
	n := 0
	for _, p := range m.byID {
		if p.ClientID == clientID {
			n++
		}
	}
	return n, nil
}
func (m *memSwingProfiles) PurgeProfiles(_ context.Context, clientID string) error {
	for id, p := range m.byID {
		if p.ClientID == clientID {
			delete(m.byID, id)
		}
	}
	return nil
}
