package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

func TestGetDerivativesBoard_NamesMissingVenues(t *testing.T) {
	h := NewMarketHandler(market.New(nil, nil))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/derivatives-board?limit=5", nil)
	rr := httptest.NewRecorder()
	h.GetDerivativesBoard(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body derivativesBoardResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Universe != 5 || body.Definite || len(body.Coins) != 5 || body.Coins[0].OpenInterest != "" {
		t.Fatalf("%+v", body)
	}
	if len(body.Coverage.Missing) == 0 {
		t.Fatalf("coverage %+v", body.Coverage)
	}
}

func TestDerivativesBoardToDTO_CVDSplit(t *testing.T) {
	dto := derivativesBoardToDTO(&domain.DerivativesBoard{
		Coins: []domain.DerivativesCoin{{
			Symbol:           "BTCUSDT",
			CVDSplit:         domain.CVDDivPriceUpCVDDown,
			CVDSplitDuration: "35m",
			CVDSplitSeconds:  "2100",
			CVDSplitDefinite: true,
			Venues: []domain.DerivativesVenueRow{{
				Exchange:        "binance",
				CVDSplit:        domain.CVDSplitAgrees,
				CVDSplitSeconds: "0",
				CVDSplitStatus:  domain.VenueReadFresh,
			}},
		}},
	})
	coin := dto.Coins[0]
	if !coin.CVDSplitDefinite || coin.CVDSplit != domain.CVDDivPriceUpCVDDown || coin.CVDSplitSeconds != "2100" || coin.CVDSplitDuration != "35m" {
		t.Fatalf("coin %+v", coin)
	}
	if coin.Venues[0].CVDSplit != domain.CVDSplitAgrees || coin.Venues[0].CVDSplitStatus != domain.VenueReadFresh {
		t.Fatalf("venue %+v", coin.Venues[0])
	}
}

func TestGetDerivativesBoard_BadLimit(t *testing.T) {
	h := NewMarketHandler(market.New(nil, nil))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/derivatives-board?limit=nope", nil)
	rr := httptest.NewRecorder()
	h.GetDerivativesBoard(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}
