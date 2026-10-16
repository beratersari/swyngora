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
	if equity.Positions[0].EntrySituation != nil || equity.Positions[0].EntryAt != "" || len(equity.Positions[0].OpenReads) != 0 {
		t.Fatalf("equity %+v", equity.Positions[0])
	}
}

func TestPortfolioViewDTO_OpenReadsKeepEveryBuy(t *testing.T) {
	at := time.Date(2026, 10, 12, 11, 0, 0, 0, time.UTC)
	cmp := domain.SituationComparison{Same: false, Clauses: []domain.SituationClauseLeft{{
		Clause: "priceCvd", Status: domain.SituationLeftChanged, Entry: domain.SituationCVDPriceUp, Live: domain.SituationCVDAgrees,
	}}}
	rows := openChangesDTO([]domain.OpenLotChange{
		{
			OpenLotRead: domain.OpenLotRead{
				ID: "new", Quantity: 0.4, OpenedAt: at, SourceTradeID: "buy-new", Newest: true,
				Situation: &domain.CoinSituation{PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceDown}},
			},
			Comparison: &cmp,
		},
		{
			OpenLotRead: domain.OpenLotRead{
				ID: "old", Quantity: 1, OpenedAt: at.Add(-time.Hour), SourceTradeID: "buy-old",
				Situation: &domain.CoinSituation{PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp}},
			},
		},
	})
	if len(rows) != 2 || !rows[0].Newest || rows[0].Quantity != 0.4 || rows[0].Comparison == nil || rows[0].OpenedAt != at.Format(time.RFC3339Nano) {
		t.Fatalf("newest %+v", rows)
	}
	if rows[1].Newest || rows[1].Comparison != nil || rows[1].Situation.PriceCVD.Kind != domain.SituationCVDPriceUp {
		t.Fatalf("older %+v", rows[1])
	}
	dto := portfolioViewDTO(&domain.PortfolioView{
		Positions: []domain.PositionView{{
			Exchange: domain.ExchangeBinance,
			Symbol:   "BTCUSDT",
			OpenReads: []domain.OpenLotRead{
				{ID: "new", Quantity: 0.4, OpenedAt: at, Newest: true},
				{ID: "old", Quantity: 1, OpenedAt: at.Add(-time.Hour)},
			},
		}},
	})
	if len(dto.Positions[0].OpenReads) != 2 || dto.Positions[0].OpenReads[0].Comparison != nil || !dto.Positions[0].OpenReads[0].Newest {
		t.Fatalf("view %+v", dto.Positions[0].OpenReads)
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
