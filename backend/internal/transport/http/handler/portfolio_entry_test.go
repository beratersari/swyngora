package handler

import (
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestPortfolioViewDTO_EntrySituation(t *testing.T) {
	at := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	dto := portfolioViewDTO(&domain.PortfolioView{
		Positions: []domain.PositionView{{
			Exchange: domain.ExchangeBinance,
			Symbol:   "BTCUSDT",
			EntrySituation: &domain.CoinSituation{
				PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDAgrees},
			},
			EntryAt: at,
		}},
	})
	if len(dto.Positions) != 1 || dto.Positions[0].EntrySituation == nil || dto.Positions[0].EntrySituation.PriceCVD.Kind != domain.SituationCVDAgrees {
		t.Fatalf("entry %+v", dto.Positions)
	}
	if dto.Positions[0].EntryAt != at.Format(time.RFC3339Nano) {
		t.Fatalf("entryAt %q", dto.Positions[0].EntryAt)
	}
	equity := portfolioViewDTO(&domain.PortfolioView{
		Positions: []domain.PositionView{{
			Exchange: domain.ExchangeNasdaq,
			Symbol:   "AAPL",
		}},
	})
	if equity.Positions[0].EntrySituation != nil || equity.Positions[0].EntryAt != "" {
		t.Fatalf("equity %+v", equity.Positions[0])
	}
}

func TestMarginPosDTO_EntrySituation(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	dto := marginPosDTO(&domain.MarginPosition{
		Exchange:  domain.ExchangeBinance,
		Symbol:    "BTCUSDT",
		Side:      domain.MarginLong,
		OpenedAt:  now,
		UpdatedAt: now,
		EntrySituation: &domain.CoinSituation{
			PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceDown},
		},
	})
	if dto.EntrySituation == nil || dto.EntrySituation.PriceCVD == nil || dto.EntrySituation.PriceCVD.Kind != domain.SituationCVDPriceDown {
		t.Fatalf("%+v", dto.EntrySituation)
	}
	blank := marginPosDTO(&domain.MarginPosition{OpenedAt: now, UpdatedAt: now})
	if blank.EntrySituation != nil {
		t.Fatalf("blank %+v", blank.EntrySituation)
	}
}
