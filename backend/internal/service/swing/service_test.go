package swing

import (
	"context"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

type fakePort struct {
	candles []domain.Candle
}

func (f fakePort) GetCandles(context.Context, domain.CandleQuery) ([]domain.Candle, error) {
	return append([]domain.Candle(nil), f.candles...), nil
}
func (f fakePort) GetTicker24h(_ context.Context, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{Symbol: symbol, LastPrice: "100", QuoteVolume: "5000000", Volume: "1000",
		OpenTime: time.Now().Add(-24 * time.Hour), CloseTime: time.Now().Add(-time.Minute)}, nil
}
func (f fakePort) GetOrderBook(context.Context, domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return &domain.RawOrderBook{}, nil
}
func (f fakePort) ListSpotMarkets(context.Context) ([]domain.SpotMarket, error) { return nil, nil }
func (f fakePort) ListProductTags(context.Context) ([]string, error)            { return nil, nil }
func (f fakePort) TagsByBase(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

type fakeSupply struct{}

func (fakeSupply) GetSupply(_ context.Context, asset string) (*domain.AssetSupply, error) {
	c := 1e6
	return &domain.AssetSupply{Asset: asset, CirculatingSupply: &c}, nil
}
func (fakeSupply) Refresh(context.Context) (int, error) { return 0, nil }

func trendCandles(n int) []domain.Candle {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	px := 50.0
	out := make([]domain.Candle, n)
	for i := 0; i < n; i++ {
		o := px
		px += 0.7
		out[i] = domain.Candle{
			OpenTime:  t0.Add(time.Duration(i) * 4 * time.Hour),
			CloseTime: t0.Add(time.Duration(i+1) * 4 * time.Hour),
			Open:      strconv.FormatFloat(o, 'f', -1, 64),
			High:      strconv.FormatFloat(px+0.3, 'f', -1, 64),
			Low:       strconv.FormatFloat(o-0.2, 'f', -1, 64),
			Close:     strconv.FormatFloat(px, 'f', -1, 64),
			Volume:    "1000",
		}
	}
	// last bar closed
	out[n-1].CloseTime = time.Now().UTC().Add(-time.Minute)
	return out
}

func TestAnalyze_ReturnsDecision(t *testing.T) {
	bars := trendCandles(120)
	m := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fakePort{candles: bars},
	}, fakeSupply{})
	svc := New(m, watchlist.New(watchliststore.NewMemory()))
	dec, err := svc.Analyze(context.Background(), "binance", "ETHUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if dec == nil || dec.Price <= 0 || dec.Interval != "4h" {
		t.Fatalf("%+v", dec)
	}
	if dec.Levels != nil && dec.Levels.StopLoss >= dec.Levels.Entry {
		t.Fatalf("levels %+v", dec.Levels)
	}
}

func TestScanWatchlist_UsesClientList(t *testing.T) {
	bars := trendCandles(80)
	m := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fakePort{candles: bars},
	}, fakeSupply{})
	wl := watchlist.New(watchliststore.NewMemory())
	_, err := wl.Add(context.Background(), "user-sw", "", "binance", "ETHUSDT", "", domain.WatchlistUnconditionalVersion)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(m, wl)
	list, err := svc.ScanWatchlist(context.Background(), "user-sw", "binance", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Symbol != "ETHUSDT" {
		t.Fatalf("%+v", list)
	}
}

type memProfiles struct {
	byID map[string]domain.CustomSignalProfile
}

func (m *memProfiles) CreateProfile(_ context.Context, p domain.CustomSignalProfile) (*domain.CustomSignalProfile, error) {
	if m.byID == nil {
		m.byID = map[string]domain.CustomSignalProfile{}
	}
	m.byID[p.ID] = p
	return &p, nil
}
func (m *memProfiles) GetProfile(_ context.Context, clientID, id string) (*domain.CustomSignalProfile, error) {
	p, ok := m.byID[id]
	if !ok || p.ClientID != clientID {
		return nil, domain.ErrNotFound
	}
	return &p, nil
}
func (m *memProfiles) UpdateProfile(_ context.Context, p domain.CustomSignalProfile) (*domain.CustomSignalProfile, error) {
	cur, ok := m.byID[p.ID]
	if !ok || cur.ClientID != p.ClientID {
		return nil, domain.ErrNotFound
	}
	m.byID[p.ID] = p
	return &p, nil
}
func (m *memProfiles) ListProfiles(_ context.Context, clientID string) ([]domain.CustomSignalProfile, error) {
	out := make([]domain.CustomSignalProfile, 0)
	for _, p := range m.byID {
		if p.ClientID == clientID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (m *memProfiles) DeleteProfile(_ context.Context, clientID, id string) error {
	p, ok := m.byID[id]
	if !ok || p.ClientID != clientID {
		return domain.ErrNotFound
	}
	delete(m.byID, id)
	return nil
}
func (m *memProfiles) CountProfiles(_ context.Context, clientID string) (int, error) {
	n := 0
	for _, p := range m.byID {
		if p.ClientID == clientID {
			n++
		}
	}
	return n, nil
}
func (m *memProfiles) PurgeProfiles(_ context.Context, clientID string) error {
	for id, p := range m.byID {
		if p.ClientID == clientID {
			delete(m.byID, id)
		}
	}
	return nil
}

type fakeHunt struct {
	up float64
}

func (f fakeHunt) GetLiquidationHunt(context.Context, string, string) (*domain.HuntReport, error) {
	return &domain.HuntReport{Bias: &domain.HuntBias{
		UpScore: f.up, DownScore: 30, Lean: domain.HuntLeanUp,
		Coverage: domain.HuntCoverage{Usable: true},
	}}, nil
}

func TestCreateAndScanCustom(t *testing.T) {
	bars := trendCandles(80)
	m := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fakePort{candles: bars},
	}, fakeSupply{})
	wl := watchlist.New(watchliststore.NewMemory())
	if _, err := wl.Add(context.Background(), "user-cs", "", "binance", "ETHUSDT", "", domain.WatchlistUnconditionalVersion); err != nil {
		t.Fatal(err)
	}
	store := &memProfiles{}
	svc := New(m, wl).WithProfiles(store).WithHunt(fakeHunt{up: 88})
	p, err := svc.CreateProfile(context.Background(), CreateProfileInput{
		ClientID: "user-cs", Name: "Hunt lean",
		Weights: map[string]float64{domain.CustomFactorHuntUp: 100},
	})
	if err != nil || p.ID == "" {
		t.Fatalf("%+v %v", p, err)
	}
	wts, id, err := svc.ResolveCustomWeights(context.Background(), "user-cs", p.ID, domain.CustomSignalWeights{}, false)
	if err != nil || id != p.ID || !wts.NeedsHunt() {
		t.Fatalf("%+v %s %v", wts, id, err)
	}
	res, err := svc.ScanCustom(context.Background(), CustomScanInput{
		ClientID: "user-cs", ProfileID: p.ID, Weights: wts, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 1 || res.Items[0].Symbol != "ETHUSDT" {
		t.Fatalf("%+v", res)
	}
	if res.Items[0].Custom.Score < 80 {
		t.Fatalf("expected hunt-heavy score, got %+v", res.Items[0].Custom)
	}
	if res.Items[0].Compare.Delta.LargestChange == nil {
		t.Fatal("expected compare largest change")
	}
}
