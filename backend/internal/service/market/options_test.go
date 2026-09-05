package market

import (
	"context"
	"errors"
	"testing"
	"time"

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
