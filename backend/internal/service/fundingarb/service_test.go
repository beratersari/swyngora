package fundingarb

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/fundingarbstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"path/filepath"
)

type fakeQuotes struct {
	rep  *domain.FundingArbReport
	scan *domain.FundingArbScan
	err  error
}

func (f *fakeQuotes) GetFundingArb(context.Context, market.FundingArbParams) (*domain.FundingArbReport, error) {
	return f.rep, f.err
}

func (f *fakeQuotes) ScanFundingArb(context.Context, market.FundingArbScanParams) (*domain.FundingArbScan, error) {
	return f.scan, f.err
}

type fakeNotify struct {
	n int
}

func (f *fakeNotify) NotifyClient(context.Context, string, string, string) error {
	f.n++
	return nil
}

func TestCreateAndProcessWatch(t *testing.T) {
	st, err := fundingarbstore.Open(filepath.Join(t.TempDir(), "fa.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	q := &fakeQuotes{rep: &domain.FundingArbReport{
		Symbol: "BTCUSDT", HorizonNet: 12,
		Trade:   &domain.FundingArbTradeView{LongExchange: "binance", ShortExchange: "bybit", WorthIt: true, Summary: "ok"},
		Summary: "ok",
	}}
	n := &fakeNotify{}
	svc := New(st, q)
	svc.SetNotifier(n)
	minP := 5.0
	w, err := svc.CreateWatch(context.Background(), CreateInput{
		ClientID: "client-a", Symbol: "BTCUSDT", Notional: 10000, HoldHours: 24, MinProfit: minP,
	})
	if err != nil {
		t.Fatal(err)
	}
	opened, closed, touched, err := svc.ProcessActiveWatches(context.Background(), time.Now().UTC())
	if err != nil || opened != 1 || closed != 0 || touched != 1 {
		t.Fatalf("open %d close %d touch %d err=%v", opened, closed, touched, err)
	}
	if n.n != 1 {
		t.Fatalf("notify %d", n.n)
	}
	opened, _, _, err = svc.ProcessActiveWatches(context.Background(), time.Now().UTC())
	if err != nil || opened != 0 {
		t.Fatalf("no duplicate open %d %v", opened, err)
	}
	if n.n != 1 {
		t.Fatalf("no duplicate notify %d", n.n)
	}
	q.rep = &domain.FundingArbReport{Symbol: "BTCUSDT", HorizonNet: 0, Summary: "not an opportunity"}
	_, closed, _, err = svc.ProcessActiveWatches(context.Background(), time.Now().UTC())
	if err != nil || closed != 1 {
		t.Fatalf("close %d %v", closed, err)
	}
	q.rep = &domain.FundingArbReport{
		Symbol: "BTCUSDT", HorizonNet: 20,
		Trade: &domain.FundingArbTradeView{LongExchange: "binance", ShortExchange: "bybit", WorthIt: true},
	}
	opened, _, _, err = svc.ProcessActiveWatches(context.Background(), time.Now().UTC())
	if err != nil || opened != 1 {
		t.Fatalf("reopen %d %v", opened, err)
	}
	if n.n != 2 {
		t.Fatalf("re-arm notify %d", n.n)
	}
	if err := svc.DeleteWatch(context.Background(), "client-a", w.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetWatch(context.Background(), "client-a", w.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestProcessActiveWatches_NoTradeIsNotNotify(t *testing.T) {
	st, err := fundingarbstore.Open(filepath.Join(t.TempDir(), "fa.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	q := &fakeQuotes{rep: &domain.FundingArbReport{Symbol: "BTCUSDT", HorizonNet: 99, Summary: "loser"}}
	n := &fakeNotify{}
	svc := New(st, q)
	svc.SetNotifier(n)
	if _, err := svc.CreateWatch(context.Background(), CreateInput{
		ClientID: "client-a", Symbol: "BTCUSDT", Notional: 10000, HoldHours: 24, MinProfit: 5,
	}); err != nil {
		t.Fatal(err)
	}
	opened, closed, _, err := svc.ProcessActiveWatches(context.Background(), time.Now().UTC())
	if err != nil || opened != 0 || closed != 0 || n.n != 0 {
		t.Fatalf("no trade must not notify open=%d close=%d n=%d err=%v", opened, closed, n.n, err)
	}
}

func TestProcessActiveWatches_DirectionFlipOpensNewSignal(t *testing.T) {
	st, err := fundingarbstore.Open(filepath.Join(t.TempDir(), "fa.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	q := &fakeQuotes{rep: &domain.FundingArbReport{
		Symbol: "BTCUSDT", HorizonNet: 12,
		Trade: &domain.FundingArbTradeView{LongExchange: "binance", ShortExchange: "bybit", WorthIt: true},
	}}
	n := &fakeNotify{}
	svc := New(st, q)
	svc.SetNotifier(n)
	if _, err := svc.CreateWatch(context.Background(), CreateInput{
		ClientID: "client-a", Symbol: "BTCUSDT", Notional: 10000, HoldHours: 24, MinProfit: 5,
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := svc.ProcessActiveWatches(context.Background(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if n.n != 1 {
		t.Fatalf("first notify %d", n.n)
	}
	q.rep = &domain.FundingArbReport{
		Symbol: "BTCUSDT", HorizonNet: 18,
		Trade: &domain.FundingArbTradeView{LongExchange: "bybit", ShortExchange: "binance", WorthIt: true},
	}
	opened, closed, _, err := svc.ProcessActiveWatches(context.Background(), time.Now().UTC())
	if err != nil || opened != 1 || closed != 1 {
		t.Fatalf("flip open=%d close=%d err=%v", opened, closed, err)
	}
	if n.n != 2 {
		t.Fatalf("flip must notify new direction %d", n.n)
	}
	sigs, err := svc.ListSignals(context.Background(), "client-a", "open", 10)
	if err != nil || len(sigs) != 1 || sigs[0].LongExchange != domain.ExchangeBybit {
		t.Fatalf("open signal %+v %v", sigs, err)
	}
	q.rep = &domain.FundingArbReport{
		Symbol: "BTCUSDT", HorizonNet: 22,
		Trade: &domain.FundingArbTradeView{LongExchange: "bybit", ShortExchange: "binance", WorthIt: true},
	}
	opened, closed, _, err = svc.ProcessActiveWatches(context.Background(), time.Now().UTC())
	if err != nil || opened != 0 || closed != 0 {
		t.Fatalf("same direction open=%d close=%d err=%v", opened, closed, err)
	}
	if n.n != 2 {
		t.Fatalf("same direction must not notify again %d", n.n)
	}
}

func TestProcessScanWatch_NewCoinAndReentry(t *testing.T) {
	st, err := fundingarbstore.Open(filepath.Join(t.TempDir(), "fa.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	q := &fakeQuotes{scan: &domain.FundingArbScan{Hits: []domain.FundingArbHit{
		{Symbol: "BTCUSDT", LongExchange: "binance", ShortExchange: "bybit", RankScore: 12, Summary: "btc"},
		{Symbol: "ETHUSDT", LongExchange: "bybit", ShortExchange: "binance", RankScore: 8, Summary: "eth"},
	}}}
	n := &fakeNotify{}
	svc := New(st, q)
	svc.SetNotifier(n)
	w, err := svc.CreateWatch(context.Background(), CreateInput{
		ClientID: "client-a", MinProfit: 5, Notional: 10000, HoldHours: 24,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !w.IsScan() {
		t.Fatalf("expected scan watch %+v", w)
	}
	opened, closed, _, err := svc.ProcessActiveWatches(context.Background(), time.Now().UTC())
	if err != nil || opened != 2 || closed != 0 || n.n != 2 {
		t.Fatalf("first scan open=%d close=%d n=%d err=%v", opened, closed, n.n, err)
	}
	opened, closed, _, err = svc.ProcessActiveWatches(context.Background(), time.Now().UTC())
	if err != nil || opened != 0 || closed != 0 || n.n != 2 {
		t.Fatalf("same hits must not re-notify open=%d close=%d n=%d err=%v", opened, closed, n.n, err)
	}
	q.scan = &domain.FundingArbScan{Hits: []domain.FundingArbHit{
		{Symbol: "ETHUSDT", LongExchange: "bybit", ShortExchange: "binance", RankScore: 9, Summary: "eth"},
	}}
	opened, closed, _, err = svc.ProcessActiveWatches(context.Background(), time.Now().UTC())
	if err != nil || opened != 0 || closed != 1 {
		t.Fatalf("btc drop open=%d close=%d err=%v", opened, closed, err)
	}
	q.scan = &domain.FundingArbScan{Hits: []domain.FundingArbHit{
		{Symbol: "BTCUSDT", LongExchange: "binance", ShortExchange: "bybit", RankScore: 15, Summary: "btc back"},
		{Symbol: "ETHUSDT", LongExchange: "bybit", ShortExchange: "binance", RankScore: 9, Summary: "eth"},
	}}
	opened, closed, _, err = svc.ProcessActiveWatches(context.Background(), time.Now().UTC())
	if err != nil || opened != 1 || closed != 0 || n.n != 3 {
		t.Fatalf("btc re-entry open=%d close=%d n=%d err=%v", opened, closed, n.n, err)
	}
	if _, err := svc.CreateWatch(context.Background(), CreateInput{ClientID: "client-a", MinProfit: 6}); err == nil {
		t.Fatal("second scan follow")
	}
}

func TestCreateWatch_BadMinProfit(t *testing.T) {
	st, err := fundingarbstore.Open(filepath.Join(t.TempDir(), "fa.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := New(st, &fakeQuotes{})
	_, err = svc.CreateWatch(context.Background(), CreateInput{ClientID: "c1", Symbol: "BTCUSDT", MinProfit: 0})
	if err == nil {
		t.Fatal("minProfit")
	}
}
