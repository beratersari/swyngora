package market

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetRSIHeatmap_FillsDots(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 5, 14)
	if err != nil {
		t.Fatal(err)
	}
	if got.Period != 14 || got.Exchange != domain.ExchangeBinance || got.Interval != "1h" {
		t.Fatalf("%+v", got)
	}
	if len(got.Items) == 0 {
		t.Fatal("expected rows")
	}
	if got.AverageRSI == nil {
		t.Fatal("expected average RSI")
	}
	for _, row := range got.Items {
		if row.Error != "" {
			t.Fatalf("%s: %s", row.Symbol, row.Error)
		}
		if row.RSI == nil {
			t.Fatalf("missing RSI for %s", row.Symbol)
		}
		if row.Zone == domain.RSIZoneUnknown {
			t.Fatalf("zone empty for %s", row.Symbol)
		}
		if row.Rank < 1 {
			t.Fatalf("rank=%d", row.Rank)
		}
	}
}

func TestGetRSIHeatmap_DropsStables(t *testing.T) {
	m := &indicatorMarket{}
	m.spot = []domain.SpotMarket{
		{Symbol: "USDCUSDT", BaseAsset: "USDC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "99999", LastPrice: "1"},
		{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1000", LastPrice: "100"},
	}
	svc := New(m, &fakeSupply{})
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 5, 14)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].Symbol != "BTCUSDT" {
		t.Fatalf("items=%+v", got.Items)
	}
}

func TestGetRSIHeatmap_RejectsBadInterval(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	_, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "3y", "quoteVolume", 5, 14)
	if err == nil {
		t.Fatal("expected invalid interval")
	}
}

type gatedListMarket struct {
	indicatorMarket
	fail    chan struct{}
	hold    chan struct{}
	entered chan struct{}
}

func (m *gatedListMarket) ListSpotMarkets(ctx context.Context) ([]domain.SpotMarket, error) {
	if ctx.Value(failBuildKey{}) != nil {
		if m.entered != nil {
			select {
			case <-m.entered:
			default:
				close(m.entered)
			}
		}
		if m.hold != nil {
			<-m.hold
		}
		return nil, context.DeadlineExceeded
	}
	return m.indicatorMarket.ListSpotMarkets(ctx)
}

type failBuildKey struct{}

func TestGetRSIHeatmap_DoMissUsesLargerCache(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	rsiHeatBeforeBuild = func() {
		seed, err := svc.buildRSIHeatmap(context.Background(), domain.ExchangeBinance, "USDT", "1h", "quoteVolume", 3, 14)
		if err != nil {
			t.Error(err)
			return
		}
		key := domain.RSIHeatmapCacheKey(domain.ExchangeBinance, "USDT", "quoteVolume", "1h", 14)
		svc.rsiHeat.Set(key, seed)
	}
	t.Cleanup(func() { rsiHeatBeforeBuild = nil })
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 1, 14)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("%+v", got.Items)
	}
}

func TestGetRSIHeatmap_BuildFailReturnsStaleFromPeer(t *testing.T) {
	m := &gatedListMarket{entered: make(chan struct{}), hold: make(chan struct{})}
	svc := New(m, &fakeSupply{})
	failCtx := context.WithValue(context.Background(), failBuildKey{}, true)
	errc := make(chan error, 1)
	var stale *domain.RSIHeatmap
	go func() {
		var err error
		stale, err = svc.GetRSIHeatmap(failCtx, "binance", "USDT", "1h", "quoteVolume", 2, 14)
		errc <- err
	}()
	select {
	case <-m.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("failing list never entered")
	}
	if _, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 3, 14); err != nil {
		t.Fatal(err)
	}
	close(m.hold)
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
	if stale == nil || !stale.Stale || len(stale.Items) == 0 {
		t.Fatalf("want stale from peer build: %+v", stale)
	}
}

func TestGetRSIHeatmap_CanceledWhileWaitingOnSemaphore(t *testing.T) {
	for i := 0; i < cap(batchUpstreamSem); i++ {
		batchUpstreamSem <- struct{}{}
	}
	t.Cleanup(func() {
		for i := 0; i < cap(batchUpstreamSem); i++ {
			<-batchUpstreamSem
		}
	})
	svc := New(&indicatorMarket{}, &fakeSupply{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var got *domain.RSIHeatmap
	var err error
	go func() {
		got, err = svc.GetRSIHeatmap(ctx, "binance", "USDT", "1h", "quoteVolume", 2, 14)
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("blocked on semaphore")
	}
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got.Items) == 0 {
		t.Fatal("expected rows")
	}
}

func TestGetRSIHeatmap_SingleflightSecondSeesCache(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	errc := make(chan error, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			limit := 1
			if i%2 == 0 {
				limit = 3
			}
			_, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", limit, 14)
			errc <- err
		}()
	}
	wg.Wait()
	close(errc)
	for err := range errc {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetRSIHeatmap_ScanLimitCap(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 200, 14)
	if err != nil || got == nil {
		t.Fatalf("%v %v", got, err)
	}
}

func TestGetRSIHeatmap_CacheHit(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	first, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 3, 14)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 3, 14)
	if err != nil {
		t.Fatal(err)
	}
	if first.AsOf != second.AsOf {
		t.Fatalf("cache miss: %v vs %v", first.AsOf, second.AsOf)
	}
}

type countingCandleMarket struct {
	indicatorMarket
	calls int
}

func (m *countingCandleMarket) GetCandles(ctx context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	m.calls++
	return m.indicatorMarket.GetCandles(ctx, q)
}

func TestGetRSIHeatmap_SmallerLimitReusesLargerCache(t *testing.T) {
	m := &countingCandleMarket{}
	m.spot = []domain.SpotMarket{
		{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "3", LastPrice: "1"},
		{Symbol: "ETHUSDT", BaseAsset: "ETH", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "2", LastPrice: "1"},
		{Symbol: "SOLUSDT", BaseAsset: "SOL", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1", LastPrice: "1"},
	}
	svc := New(m, &fakeSupply{})
	full, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 3, 14)
	if err != nil {
		t.Fatal(err)
	}
	if len(full.Items) != 3 {
		t.Fatalf("full=%d", len(full.Items))
	}
	after := m.calls
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 1, 14)
	if err != nil {
		t.Fatal(err)
	}
	if m.calls != after {
		t.Fatalf("top-1 refetch: calls %d -> %d", after, m.calls)
	}
	if len(got.Items) != 1 || got.Items[0].Symbol != full.Items[0].Symbol {
		t.Fatalf("clip=%+v full0=%s", got.Items, full.Items[0].Symbol)
	}
}

func TestGetRSIHeatmap_FetchesShortSeed(t *testing.T) {
	var gotLimit int
	m := &limitSpyMarket{}
	m.spot = []domain.SpotMarket{
		{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1", LastPrice: "1"},
	}
	m.onCandles = func(q domain.CandleQuery) { gotLimit = q.Limit }
	svc := New(m, &fakeSupply{})
	if _, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 1, 14); err != nil {
		t.Fatal(err)
	}
	if gotLimit != domain.RSIHeatmapCandleLimit {
		t.Fatalf("candle limit=%d want %d", gotLimit, domain.RSIHeatmapCandleLimit)
	}
}

type limitSpyMarket struct {
	indicatorMarket
	onCandles func(domain.CandleQuery)
}

func (m *limitSpyMarket) GetCandles(ctx context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	if m.onCandles != nil {
		m.onCandles(q)
	}
	return m.indicatorMarket.GetCandles(ctx, q)
}

// formingRSIMarket returns closed +1h up-closes, then a still-forming crash bar.
// Last closed Wilder RSI is ~100 (overbought). Including the forming dump flips it to ~9 (oversold).
type formingRSIMarket struct {
	fakeMarket
}

func (m *formingRSIMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	n := q.Limit
	if n < 30 {
		n = 30
	}
	now := time.Now().UTC()
	out := make([]domain.Candle, n)
	// n-1 closed hourly up-bars, then a crash whose CloseTime is still ahead of now.
	lastClosedOpen := now.Truncate(time.Hour).Add(-time.Hour)
	for i := 0; i < n-1; i++ {
		open := lastClosedOpen.Add(-time.Duration(n-2-i) * time.Hour)
		out[i] = domain.Candle{
			OpenTime:  open,
			CloseTime: open.Add(time.Hour).Add(-time.Millisecond),
			Close:     fmt.Sprintf("%g", 100+float64(i)),
		}
	}
	out[n-1] = domain.Candle{
		OpenTime:  now.Truncate(time.Hour),
		CloseTime: now.Add(30 * time.Minute),
		Close:     "1",
	}
	return out, nil
}

func TestGetRSIHeatmap_ValidationAndDefaults(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	if _, err := svc.GetRSIHeatmap(context.Background(), "nope", "", "1h", "quoteVolume", 1, 14); err == nil {
		t.Fatal("bad exchange")
	}
	if _, err := svc.GetRSIHeatmap(context.Background(), "binance", "", "1h", "quoteVolume", 1, 1); err == nil {
		t.Fatal("period too small")
	}
	if _, err := svc.GetRSIHeatmap(context.Background(), "binance", "", "1h", "not-a-sort", 1, 14); err == nil {
		t.Fatal("bad sort")
	}
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "", "1h", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Quote != "USDT" || got.Period != 14 || len(got.Items) == 0 {
		t.Fatalf("defaults %+v", got)
	}
}

func TestGetRSIHeatmap_FallsBackWhenMarketCapUnavailable(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{err: domain.ErrNotFound})
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "marketCapCirculating", 2, 14)
	if err != nil {
		t.Fatal(err)
	}
	if got.SortBy != string(domain.SpotSortQuoteVolume) {
		t.Fatalf("sort=%s", got.SortBy)
	}
}

func TestGetRSIHeatmap_ListError(t *testing.T) {
	svc := New(&fakeMarket{err: context.DeadlineExceeded}, &fakeSupply{})
	if _, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 2, 14); err == nil {
		t.Fatal("expected list error")
	}
}

func TestGetRSIHeatmap_SkipsBlankAndStableAndSplitsBase(t *testing.T) {
	m := &indicatorMarket{}
	m.spot = []domain.SpotMarket{
		{Symbol: "", QuoteAsset: "USDT", QuoteVolume: "9", Status: "TRADING"},
		{Symbol: "USDCUSDT", BaseAsset: "USDC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "8"},
		{Symbol: "FOOUSDT", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "7", LastPrice: "1"},
	}
	svc := New(m, &fakeSupply{})
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 5, 14)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].Symbol != "FOOUSDT" || got.Items[0].Base != "FOO" {
		t.Fatalf("%+v", got.Items)
	}
}

func TestGetRSIHeatmap_RowCandleAndParseErrors(t *testing.T) {
	m := &rowErrMarket{}
	m.spot = []domain.SpotMarket{
		{Symbol: "BADUSDT", BaseAsset: "BAD", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "2"},
		{Symbol: "NANUSDT", BaseAsset: "NAN", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1"},
	}
	svc := New(m, &fakeSupply{})
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 2, 14)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("%+v", got.Items)
	}
	if got.Items[0].Error == "" || got.Items[1].Error == "" {
		t.Fatalf("want row errors: %+v", got.Items)
	}
}

type rowErrMarket struct {
	indicatorMarket
}

func (m *rowErrMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	if q.Symbol == "BADUSDT" {
		return nil, context.DeadlineExceeded
	}
	return []domain.Candle{{Close: "n/a"}, {Close: "also-bad"}}, nil
}

func TestGetRSIHeatmap_CanceledContextMarksRows(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := New(&indicatorMarket{}, &fakeSupply{})
	got, err := svc.GetRSIHeatmap(ctx, "binance", "USDT", "1h", "quoteVolume", 2, 14)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) == 0 {
		t.Fatal("expected rows")
	}
	for _, row := range got.Items {
		if row.Error == "" && row.RSI == nil {
			t.Fatalf("canceled row should error or have RSI: %+v", row)
		}
	}
}

func TestGetRSIHeatmap_StaleServedWhileRefresh(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	first, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 2, 14)
	if err != nil || first == nil {
		t.Fatalf("seed %v %v", first, err)
	}
	key := domain.RSIHeatmapCacheKey(domain.ExchangeBinance, "USDT", "quoteVolume", "1h", 14)
	svc.rsiHeat.SetWithTTL(key, first, time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 2, 14)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Stale || len(got.Items) == 0 {
		t.Fatalf("want stale hit: %+v", got)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hit, ok := svc.rsiHeat.Get(key); ok && hit != nil && !hit.AsOf.Equal(first.AsOf) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestGetRSIHeatmap_BuildErrorFallsBackToStale(t *testing.T) {
	good := New(&indicatorMarket{}, &fakeSupply{})
	seed, err := good.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 2, 14)
	if err != nil {
		t.Fatal(err)
	}
	broken := New(&fakeMarket{err: context.DeadlineExceeded}, &fakeSupply{})
	key := domain.RSIHeatmapCacheKey(domain.ExchangeBinance, "USDT", "quoteVolume", "1h", 14)
	broken.rsiHeat.SetWithTTL(key, seed, time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	got, err := broken.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 2, 14)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Stale || len(got.Items) == 0 {
		t.Fatalf("want stale after build fail: %+v err=%v", got, err)
	}
}

func TestBuildRSIHeatmap_DoesNotShrinkCachedMap(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	full, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 3, 14)
	if err != nil || len(full.Items) < 2 {
		t.Fatalf("full=%+v err=%v", full, err)
	}
	small, err := svc.buildRSIHeatmap(context.Background(), domain.ExchangeBinance, "USDT", "1h", "quoteVolume", 1, 14)
	if err != nil || len(small.Items) != 1 {
		t.Fatalf("small=%+v err=%v", small, err)
	}
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 3, 14)
	if err != nil || len(got.Items) < 2 {
		t.Fatalf("cache shrunk: %+v err=%v", got, err)
	}
}

func TestRefreshRSIHeatmap_FreshCacheAndBadExchange(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	_, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 2, 14)
	if err != nil {
		t.Fatal(err)
	}
	key := domain.RSIHeatmapCacheKey(domain.ExchangeBinance, "USDT", "quoteVolume", "1h", 14)
	svc.refreshRSIHeatmap(key, "binance", "USDT", "1h", "quoteVolume", 2, 14)
	svc.rsiHeat.Delete(key)
	svc.refreshRSIHeatmap(key, "not-an-exchange", "USDT", "1h", "quoteVolume", 2, 14)
}

func TestGetRSIHeatmap_IgnoresFormingCandleSoZoneDoesNotFlip(t *testing.T) {
	m := &formingRSIMarket{}
	m.spot = []domain.SpotMarket{
		{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1000", LastPrice: "128"},
	}
	svc := New(m, &fakeSupply{})
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 1, 14)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("items=%+v", got.Items)
	}
	row := got.Items[0]
	if row.Error != "" {
		t.Fatalf("row error: %s", row.Error)
	}
	if row.RSI == nil {
		t.Fatal("missing RSI")
	}
	// Closed series is all-up → Wilder RSI ~100, overbought. The forming crash
	// must not be seeded or the cell flips to oversold (~9) mid-hour.
	if *row.RSI < 70 {
		t.Fatalf("forming bar leaked into RSI: rsi=%.4f zone=%s (want last-closed ~100 / overbought)", *row.RSI, row.Zone)
	}
	if row.Zone != domain.RSIZoneOverbought {
		t.Fatalf("zone=%s rsi=%.4f; forming dump must not flip last-closed overbought to %s", row.Zone, *row.RSI, row.Zone)
	}
}
