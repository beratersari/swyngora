package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/optionsstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type stubOptions struct {
	quotes []domain.OptionQuote
	dvol   domain.OptionsDVOL
	index  float64
	err    error
}

func (s stubOptions) GetOptionQuotes(_ context.Context, asset string) ([]domain.OptionQuote, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make([]domain.OptionQuote, 0, len(s.quotes))
	for _, q := range s.quotes {
		if q.Asset == asset {
			out = append(out, q)
		}
	}
	return out, nil
}

func (s stubOptions) GetOptionsDVOL(context.Context, string) (domain.OptionsDVOL, error) {
	return s.dvol, nil
}

func (s stubOptions) GetOptionsIndex(context.Context, string) (float64, error) {
	return s.index, nil
}

func (s stubOptions) GetOptionsDVOLHistory(_ context.Context, _ string, from, to time.Time) ([]domain.OptionsDVOLPoint, error) {
	if to.IsZero() {
		to = time.Now().UTC()
	}
	if from.IsZero() {
		from = to.Add(-24 * time.Hour)
	}
	return []domain.OptionsDVOLPoint{
		{Time: from, Value: s.dvol.Value + 2},
		{Time: to, Value: s.dvol.Value},
	}, nil
}

func TestGetOptions(t *testing.T) {
	exp := time.Now().UTC().Add(30 * 24 * time.Hour).Truncate(time.Hour)
	svc := New(nil, nil).WithOptionsChain(stubOptions{
		index: 100000,
		dvol:  domain.OptionsDVOL{Value: 49, HasChange: true, Change24h: -1},
		quotes: []domain.OptionQuote{{
			Asset: "BTC", Expiry: exp, Strike: 100000, Side: domain.OptionSideCall,
			OpenInterest: 10, Volume: 1, MarkIV: 50, Underlying: 100000,
		}, {
			Asset: "BTC", Expiry: exp, Strike: 100000, Side: domain.OptionSidePut,
			OpenInterest: 20, Volume: 2, MarkIV: 52, Underlying: 100000,
		}},
	})
	got, err := svc.GetOptions(context.Background(), "", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Asset != "BTC" || got.Exchange != "deribit" || got.OpenInterest != 30 {
		t.Fatalf("%+v", got)
	}
	if !got.HasPutCallOI || got.PutCallOI != 2 || got.Lean != domain.OptionsLeanPuts {
		t.Fatalf("pcr=%v lean=%s", got.PutCallOI, got.Lean)
	}
	if got.ATM == nil || got.ATM.IV != 51 {
		t.Fatalf("atm=%+v", got.ATM)
	}
}

func TestGetOptions_BadAsset(t *testing.T) {
	svc := New(nil, nil).WithOptionsChain(stubOptions{})
	if _, err := svc.GetOptions(context.Background(), "", "SOL"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetOptions_MissingPort(t *testing.T) {
	svc := New(nil, nil)
	if _, err := svc.GetOptions(context.Background(), "", "BTC"); !errors.Is(err, domain.ErrUpstream) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetOptions_AttachesVsMarketWhenTapeDisagrees(t *testing.T) {
	exp := time.Now().UTC().Add(30 * 24 * time.Hour).Truncate(time.Hour)
	now := time.Now().UTC()
	mkt := &fakeMarket{ticker: &domain.Ticker24h{
		Symbol: "BTCUSDT", LastPrice: "100000", HighPrice: "102000", LowPrice: "99000",
	}}
	fund := &fakeFunding{ser: &domain.FundingSeries{
		Exchange: domain.ExchangeBinance,
		Current:  domain.FundingPoint{Time: now, Rate: 0.00015, Predicted: true},
	}}
	ls := &fakeLS{ser: &domain.LongShortSeries{
		Exchange: domain.ExchangeBinance,
		Current:  domain.LongShortPoint{Time: now, LongShare: 0.64, ShortShare: 0.36, Ratio: 1.78},
	}}
	svc := New(mkt, nil).
		WithOptionsChain(stubOptions{
			index: 100000,
			dvol:  domain.OptionsDVOL{Value: 70, HasChange: true, Change24h: 0.4},
			quotes: []domain.OptionQuote{{
				Asset: "BTC", Expiry: exp, Strike: 100000, Side: domain.OptionSideCall,
				OpenInterest: 10, Volume: 1, MarkIV: 70, Underlying: 100000,
			}, {
				Asset: "BTC", Expiry: exp, Strike: 100000, Side: domain.OptionSidePut,
				OpenInterest: 25, Volume: 2, MarkIV: 72, Underlying: 100000,
			}},
		}).
		WithFundingRate(map[domain.Exchange]domain.FundingRatePort{domain.ExchangeBinance: fund}).
		WithLongShortRatio(map[domain.Exchange]domain.LongShortRatioPort{domain.ExchangeBinance: ls})
	got, err := svc.GetOptions(context.Background(), "", "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if got.VsMarket == nil || !got.VsMarket.Compared || len(got.VsMarket.Findings) == 0 {
		t.Fatalf("vsMarket=%+v", got.VsMarket)
	}
	kinds := map[string]bool{}
	for _, f := range got.VsMarket.Findings {
		kinds[f.Kind] = true
	}
	if !kinds[domain.OptionsFindingPutsVsLongs] {
		t.Fatalf("want puts_vs_futures_long in %+v", got.VsMarket.Findings)
	}
}

func TestGetOptions_AttachesHistory(t *testing.T) {
	exp := time.Now().UTC().Add(30 * 24 * time.Hour).Truncate(time.Hour)
	store := &optionsstore.Memory{}
	past := time.Now().UTC().Add(-25 * time.Hour).Truncate(5 * time.Minute)
	_, _ = store.InsertSample(context.Background(), domain.OptionsSample{
		Asset: "BTC", SampledAt: past, OpenInterest: 20, OpenInterestUSD: 2e6,
		HasPutCallOI: true, PutCallOI: 0.8, HasDVOL: true, DVOL: 40,
	})
	svc := New(nil, nil).WithOptionsChain(stubOptions{
		index: 100000,
		dvol:  domain.OptionsDVOL{Value: 49, HasChange: true, Change24h: -1},
		quotes: []domain.OptionQuote{{
			Asset: "BTC", Expiry: exp, Strike: 100000, Side: domain.OptionSideCall,
			OpenInterest: 10, Volume: 1, MarkIV: 50, Underlying: 100000,
		}, {
			Asset: "BTC", Expiry: exp, Strike: 100000, Side: domain.OptionSidePut,
			OpenInterest: 20, Volume: 2, MarkIV: 52, Underlying: 100000,
		}},
	}).WithOptionsHistory(store)
	got, err := svc.GetOptions(context.Background(), "", "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if got.History == nil || len(got.History.Points) < 2 {
		t.Fatalf("history=%+v", got.History)
	}
	var h24 *domain.OptionsHistoryWindow
	for i := range got.History.Windows {
		if got.History.Windows[i].Window == domain.OptionsHistoryWindow24h {
			h24 = &got.History.Windows[i]
		}
	}
	if h24 == nil || !h24.OI.Complete || !h24.PutCall.Complete {
		t.Fatalf("24h window %+v", h24)
	}
}
