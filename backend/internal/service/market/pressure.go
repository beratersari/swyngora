package market

import (
	"context"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const marketPressureNote = "Net market pressure from −1 (sell) to +1 (buy) blends the resting order book, aggressive CVD, and absorption. When the book looks bid-heavy but CVD is selling, absorption is weighted more so the net is not a wash. change1h is this score minus the score ~1h ago. Informational only — not financial advice."

// GetMarketPressure combines order-book imbalance, CVD, and absorption into −1…+1.
func (s *Service) GetMarketPressure(ctx context.Context, exchange, symbol string) (*domain.MarketPressureReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex := strings.TrimSpace(exchange)
	if ex == "" {
		ex = "all"
	}
	now := time.Now().UTC()
	cur := domain.MarketPressureInputs{}
	past := domain.MarketPressureInputs{}
	pastAt := now.Add(-time.Hour)

	var (
		bookImb          float64
		bookOK           bool
		flow             float64
		flowOK           bool
		cvdPts           []domain.CVDPoint
		absKind          string
		absScore         int
		absOK            bool
		absGrade, absSum string
		absPts           []domain.AbsorptionPoint
		wg               sync.WaitGroup
	)
	wg.Add(3)
	go func() {
		defer wg.Done()
		bookImb, bookOK = s.pressureBook(ctx, ex, symbol)
	}()
	go func() {
		defer wg.Done()
		cvd, err := s.GetCVD(ctx, takerExchangeForPressure(ex), symbol, "", "")
		if err != nil || cvd == nil {
			return
		}
		ser := pickCVDForPressure(cvd)
		if ser == nil {
			return
		}
		cvdPts = append([]domain.CVDPoint(nil), ser.Points...)
		if w, ok := cvdWindowByID(ser.Windows, domain.CVDWindow1h); ok {
			if v, ok := domain.CVDFlowShare(w); ok {
				flow, flowOK = v, true
			}
		}
	}()
	go func() {
		defer wg.Done()
		rep, err := s.GetAbsorption(ctx, takerExchangeForPressure(ex), symbol)
		if err != nil || rep == nil {
			return
		}
		ser := pickAbsorptionForPressure(rep)
		if ser == nil {
			return
		}
		absPts = append([]domain.AbsorptionPoint(nil), ser.Points...)
		if kind, score, ok := domain.AbsorptionSignedFromWindow(ser.Windows, domain.CVDWindow1h); ok {
			absKind, absScore, absOK = kind, score, true
			absGrade = domain.AbsorptionGrade(score)
		} else if ser.Current != nil && ser.Current.Kind != "" && ser.Current.Score > 0 {
			absKind, absScore, absOK = ser.Current.Kind, ser.Current.Score, true
			absGrade = ser.Current.Grade
			absSum = ser.Current.Summary
		}
	}()
	wg.Wait()
	cur.Book, cur.BookOK = bookImb, bookOK
	cur.Flow, cur.FlowOK = flow, flowOK
	cur.AbsKind, cur.AbsScore, cur.AbsOK = absKind, absScore, absOK
	cur.AbsGrade, cur.AbsSummary = absGrade, absSum

	if imb, ok := s.pressureBookAt(ctx, ex, symbol, pastAt); ok {
		past.Book, past.BookOK = imb, true
	}
	if v, ok := domain.CVDFlowShareInRange(cvdPts, now.Add(-2*time.Hour), pastAt); ok {
		past.Flow, past.FlowOK = v, true
	}
	if kind, score, ok := domain.AbsorptionInRange(absPts, now.Add(-2*time.Hour), pastAt, 0.35); ok {
		past.AbsKind, past.AbsScore, past.AbsOK = kind, score, true
		past.AbsGrade = domain.AbsorptionGrade(score)
	}
	pastComplete := past.BookOK || (past.FlowOK && past.AbsOK) || (past.FlowOK && cur.BookOK)
	got := domain.BuildMarketPressure(symbol, ex, now, cur, past, pastAt, pastComplete)
	got.Note = marketPressureNote
	return got, nil
}

func takerExchangeForPressure(exchange string) string {
	if _, err := domain.ParseTakerExchange(exchange); err != nil {
		return "all"
	}
	return exchange
}

func (s *Service) pressureBook(ctx context.Context, exchange, symbol string) (float64, bool) {
	if exchange == "" || exchange == "all" {
		comb, err := s.GetCombinedOrderBookAnalysis(ctx, symbol, domain.DefaultOrderBookRangePct)
		if err != nil || comb == nil {
			return 0, false
		}
		return comb.Imbalance, true
	}
	ob, err := s.GetSpotOrderBook(ctx, exchange, symbol, "", domain.DefaultOrderBookLevels, domain.DefaultOrderBookRangePct)
	if err != nil || ob == nil {
		return 0, false
	}
	return ob.Analysis.Imbalance, true
}

func (s *Service) pressureBookAt(ctx context.Context, exchange, symbol string, at time.Time) (float64, bool) {
	if s == nil || s.bookHist == nil || at.IsZero() {
		return 0, false
	}
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit, domain.ExchangeCoinbase}
	if exchange != "" && exchange != "all" {
		if ex, err := s.ResolveExchange(exchange); err == nil {
			want = []domain.Exchange{ex}
		}
	}
	for _, ex := range want {
		got, err := s.bookHist.SnapshotAt(ctx, string(ex), symbol, at)
		if err != nil || got == nil {
			continue
		}
		return got.Imbalance, true
	}
	return 0, false
}

func pickCVDForPressure(rep *domain.CVDReport) *domain.CVDVenueSeries {
	if rep == nil {
		return nil
	}
	if rep.Combined != nil && len(rep.Combined.Windows) > 0 {
		return rep.Combined
	}
	for i := range rep.Venues {
		if rep.Venues[i].Error == "" && (len(rep.Venues[i].Windows) > 0 || len(rep.Venues[i].Points) > 0) {
			return &rep.Venues[i]
		}
	}
	return nil
}

func pickAbsorptionForPressure(rep *domain.AbsorptionReport) *domain.AbsorptionVenue {
	if rep == nil {
		return nil
	}
	if rep.Combined != nil && (len(rep.Combined.Windows) > 0 || rep.Combined.Current != nil) {
		return rep.Combined
	}
	for i := range rep.Venues {
		if rep.Venues[i].Error == "" {
			return &rep.Venues[i]
		}
	}
	return nil
}

func cvdWindowByID(windows []domain.CVDWindowStat, id string) (domain.CVDWindowStat, bool) {
	for _, w := range windows {
		if w.Window == id {
			return w, true
		}
	}
	return domain.CVDWindowStat{}, false
}
