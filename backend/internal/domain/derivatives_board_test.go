package domain

import (
	"testing"
	"time"
)

func TestParseDerivativesBoardLimit(t *testing.T) {
	if ParseDerivativesBoardLimit(0) != derivativesBoardDefault || ParseDerivativesBoardLimit(2) != 5 || ParseDerivativesBoardLimit(99) != 99 || ParseDerivativesBoardLimit(500) != LiquidRecorderLimit {
		t.Fatalf("limits 0=%d 2=%d 99=%d 500=%d", ParseDerivativesBoardLimit(0), ParseDerivativesBoardLimit(2), ParseDerivativesBoardLimit(99), ParseDerivativesBoardLimit(500))
	}
}

func TestBuildDerivativesBoard_CombinesFreshVenues(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	got := BuildDerivativesBoard("all", []DerivativesCoinInput{
		boardCoin("ETHUSDT", now, 1e9, 2e9, 1e9, 0, 0),
		boardCoin("BTCUSDT", now, 8e9, 1e9, 1e9, 4e6, 2e6),
	}, now)
	if got.Universe != 2 || got.Coins[0].Symbol != "BTCUSDT" || !got.Definite {
		t.Fatalf("board %+v", got)
	}
	btc := got.Coins[0]
	if btc.OpenInterest != "10000000000" || !btc.OpenInterestDefinite {
		t.Fatalf("oi %+v", btc)
	}
	if btc.FundingWeight != DerivativesWeightOI || btc.FundingRatePct != "+0.009" {
		t.Fatalf("funding %s %s", btc.FundingWeight, btc.FundingRatePct)
	}
	if btc.Price != "100000" || btc.PriceExchange != "binance" {
		t.Fatalf("price %s %s", btc.Price, btc.PriceExchange)
	}
	if !btc.Change24hDefinite || btc.Change24h != "+5000000000" {
		t.Fatalf("24h %+v", btc.Change24h)
	}
	if btc.LiqLong != "12000000" || btc.LiqShort != "6000000" {
		t.Fatalf("liq %s %s", btc.LiqLong, btc.LiqShort)
	}
	if !btc.LongDefinite || btc.LongPct != "55" {
		t.Fatalf("long %+v", btc.LongPct)
	}
	if !btc.TakerDefinite || btc.TakerBuyPct != "60" {
		t.Fatalf("taker %+v", btc.TakerBuyPct)
	}
	if got.Summary == "" || got.Coverage.Explain() != "" {
		t.Fatalf("summary %q explain %q", got.Summary, got.Coverage.Explain())
	}
}

func TestBuildDerivativesBoard_MissingVenueWithholdsCombined(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	coin := boardCoin("BTCUSDT", now, 8e9, 1e9, 1e9, 0, 0)
	coin.Venues[2].OIOK = false
	coin.Venues[2].FundingOK = false
	got := BuildDerivativesBoard("all", []DerivativesCoinInput{coin}, now)
	row := got.Coins[0]
	if row.OpenInterestDefinite || row.OpenInterest != "" || row.FundingDefinite || row.FundingRatePct != "" {
		t.Fatalf("combined leaked %+v", row)
	}
	if row.Change1hDefinite || row.Change24hDefinite || got.Definite {
		t.Fatalf("changes definite %+v", row)
	}
	if row.Venues[0].OIValue != "8000000000" || row.Venues[2].OIStatus != VenueReadMissing {
		t.Fatalf("venues %+v", row.Venues)
	}
	if len(got.Coverage.Missing) != 1 || got.Coverage.Missing[0] != "okx" {
		t.Fatalf("coverage %+v", got.Coverage)
	}
	if got.Summary == "" {
		t.Fatal("expected a coverage sentence")
	}
}

func TestBuildDerivativesBoard_LongShareWeightsByOpenInterest(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	coin := boardCoin("BTCUSDT", now, 8e9, 1e9, 1e9, 0, 0)
	coin.Venues[0].LongShare = 0.80
	coin.Venues[1].LongShare = 0.40
	coin.Venues[2].LongShare = 0.40
	got := BuildDerivativesBoard("all", []DerivativesCoinInput{coin}, now)
	if got.Coins[0].LongPct != "72" {
		t.Fatalf("weighted %s", got.Coins[0].LongPct)
	}
	coin.Venues[2].OIOK = false
	got = BuildDerivativesBoard("all", []DerivativesCoinInput{coin}, now)
	if !got.Coins[0].LongDefinite || got.Coins[0].LongPct != "53.33" {
		t.Fatalf("equal %+v", got.Coins[0])
	}
}

func TestBuildDerivativesBoard_TakerSumsNotional(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	coin := boardCoin("BTCUSDT", now, 8e9, 1e9, 1e9, 0, 0)
	coin.Venues[0].TakerBuy, coin.Venues[0].TakerSell = 1000, 0
	coin.Venues[1].TakerBuy, coin.Venues[1].TakerSell = 0, 10
	coin.Venues[2].TakerBuy, coin.Venues[2].TakerSell = 0, 10
	got := BuildDerivativesBoard("all", []DerivativesCoinInput{coin}, now)
	if got.Coins[0].TakerBuyPct != "98.04" {
		t.Fatalf("summed %s", got.Coins[0].TakerBuyPct)
	}
	coin.Venues[2].TakerOK = false
	got = BuildDerivativesBoard("all", []DerivativesCoinInput{coin}, now)
	row := got.Coins[0]
	if row.TakerDefinite || row.TakerBuyPct != "" || row.Venues[0].TakerBuyPct != "100" || row.Venues[2].TakerStatus != VenueReadMissing {
		t.Fatalf("withheld %+v venues %+v", row.TakerBuyPct, row.Venues)
	}
}

func TestBuildDerivativesBoard_StaleLongShareIsBlank(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	coin := boardCoin("BTCUSDT", now, 8e9, 1e9, 1e9, 0, 0)
	coin.Venues[2].LongShortAt = now.Add(-DerivativesRatioMaxAge - time.Minute)
	got := BuildDerivativesBoard("all", []DerivativesCoinInput{coin}, now)
	row := got.Coins[0]
	if row.LongDefinite || row.LongPct != "" || row.Venues[0].LongPct != "55" || row.Venues[2].LongStatus != VenueReadStale {
		t.Fatalf("stale %+v venues %+v", row.LongPct, row.Venues)
	}
}

func TestBuildDerivativesBoard_ScanColumns(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	coin := boardCoin("BTCUSDT", now, 8e9, 1e9, 1e9, 0, 0)
	coin.Venues[0].TopLongShare = 0.65
	coin.Venues[0].TopLongOK = true
	coin.Venues[1].TopUnsupported = true
	coin.Venues[2].TopLongShare = 0.50
	coin.Venues[2].TopLongOK = true
	for i := range coin.Venues {
		coin.Venues[i].CVDDayOK = true
		coin.Venues[i].CVDAt = now
		coin.Venues[i].PrintOK = true
		coin.Venues[i].PrintAt = now
		coin.Venues[i].PrintCount = 10
		coin.Venues[i].PrintPriorCount = 10
		coin.Venues[i].BookOK = true
		coin.Venues[i].BookAt = now
		coin.Venues[i].LiqHeatOK = true
		coin.Venues[i].LiqGrade = CascadeGradeQuiet
	}
	coin.Venues[0].CVDDay = 100
	coin.Venues[1].CVDDay = -40
	coin.Venues[2].CVDDay = 10
	coin.Venues[0].PrintNotional = 200
	coin.Venues[0].PrintPriorNotional = 100
	coin.Venues[1].PrintNotional = 200
	coin.Venues[1].PrintPriorNotional = 100
	coin.Venues[2].PrintNotional = 200
	coin.Venues[2].PrintPriorNotional = 100
	coin.Venues[0].BidNotional = 60
	coin.Venues[0].AskNotional = 40
	coin.Venues[1].BidNotional = 60
	coin.Venues[1].AskNotional = 40
	coin.Venues[2].BidNotional = 60
	coin.Venues[2].AskNotional = 40
	coin.Venues[0].LiqGrade = CascadeGradeCascade
	coin.Venues[0].LiqRatio = 4.5
	coin.Venues[2].LiqGrade = CascadeGradeElevated
	coin.Venues[2].LiqRatio = 2

	got := BuildDerivativesBoard("all", []DerivativesCoinInput{coin}, now)
	row := got.Coins[0]
	if !row.GapDefinite || row.GapPct != "+8.33" || row.GapScope != "binance,okx" {
		t.Fatalf("gap %+v", row.GapPct)
	}
	if row.Venues[1].GapStatus != VenueReadUnsupported || row.Venues[1].GapPct != "" {
		t.Fatalf("bybit gap %+v", row.Venues[1])
	}
	if !row.CVDDayDefinite || row.CVDDay != "+70" {
		t.Fatalf("cvd %s", row.CVDDay)
	}
	if !row.PrintDefinite || row.PrintChangePct != "+100" || row.PrintRead != PrintSizeReadLarger {
		t.Fatalf("print %s %s", row.PrintChangePct, row.PrintRead)
	}
	if !row.LiqHeatDefinite || row.LiqHeat != CascadeGradeCascade || row.LiqHeatVenue != "binance" || row.LiqHeatRatio != "4.5" {
		t.Fatalf("heat %s %s %s", row.LiqHeat, row.LiqHeatVenue, row.LiqHeatRatio)
	}
	if !row.BidDefinite || row.BidPct != "60" {
		t.Fatalf("bid %s", row.BidPct)
	}
	if !got.Definite {
		t.Fatal("scan columns must not clear the original board definite flag")
	}
}

func TestBuildDerivativesBoard_OrderCountSumsFreshPrints(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	coin := boardCoin("BTCUSDT", now, 8e9, 1e9, 1e9, 0, 0)
	for i := range coin.Venues {
		coin.Venues[i].OrderOK = true
		coin.Venues[i].OrderAt = now
		coin.Venues[i].OrderBuy = 8
		coin.Venues[i].OrderSell = 2
	}
	got := BuildDerivativesBoard("all", []DerivativesCoinInput{coin}, now)
	row := got.Coins[0]
	if !row.OrderDefinite || row.OrderRatio != "+0.60" || row.OrderLean != OrderCountLeanBuy || row.OrderBuy != 24 || row.OrderSell != 6 {
		t.Fatalf("order %s %s %d %d", row.OrderRatio, row.OrderLean, row.OrderBuy, row.OrderSell)
	}
	if !got.Definite {
		t.Fatal("order count must not clear the original board definite flag")
	}
}

func TestBuildDerivativesBoard_OrderCountWithholdsAQuietVenue(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	coin := boardCoin("BTCUSDT", now, 8e9, 1e9, 1e9, 0, 0)
	coin.Venues[0].OrderOK = true
	coin.Venues[0].OrderAt = now
	coin.Venues[0].OrderBuy = 8
	coin.Venues[0].OrderSell = 2
	coin.Venues[1].OrderOK = true
	coin.Venues[1].OrderAt = now
	coin.Venues[1].OrderBuy = 8
	coin.Venues[1].OrderSell = 2
	got := BuildDerivativesBoard("all", []DerivativesCoinInput{coin}, now)
	row := got.Coins[0]
	if row.OrderDefinite || row.OrderRatio != "" {
		t.Fatalf("leaked %q", row.OrderRatio)
	}
	if row.Venues[0].OrderRatio != "+0.60" || row.Venues[2].OrderStatus != VenueReadMissing {
		t.Fatalf("venues %+v", row.Venues)
	}
}

func TestAttachBoardUnusual_NamesTheFarthestVenue(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	coin := boardCoin("BTCUSDT", now, 8e9, 1e9, 1e9, 0, 0)
	got := BuildDerivativesBoard("all", []DerivativesCoinInput{coin}, now)
	definite := got.Definite
	AttachBoardUnusual(got, []UnusualScanHit{
		{Symbol: "BTCUSDT", Exchange: "binance", Metric: UnusualMetricFunding, Band: UnusualBandExtremeHigh, Percentile: 99, Distance: 49},
		{Symbol: "BTCUSDT", Exchange: "okx", Metric: UnusualMetricSpread, Band: UnusualBandLow, Percentile: 10, Distance: 40},
		{Symbol: "BTCUSDT", Exchange: "bybit", Metric: UnusualMetricFunding, Band: UnusualBandTypical, Percentile: 50, Distance: 0},
	})
	row := got.Coins[0]
	if !row.UnusualDefinite || row.UnusualVenue != "binance" || row.UnusualMetric != UnusualMetricFunding || row.UnusualBand != UnusualBandExtremeHigh || row.UnusualDistance != "49" {
		t.Fatalf("unusual %s %s %s %s", row.UnusualVenue, row.UnusualMetric, row.UnusualBand, row.UnusualDistance)
	}
	var okxBand, bybitMetric string
	for _, v := range row.Venues {
		if v.Exchange == "okx" {
			okxBand = v.UnusualBand
		}
		if v.Exchange == "bybit" {
			bybitMetric = v.UnusualMetric
		}
	}
	if okxBand != UnusualBandLow || bybitMetric != "" {
		t.Fatalf("okx %s bybit %s", okxBand, bybitMetric)
	}
	if got.Definite != definite {
		t.Fatal("unusual must not change the original definite flag")
	}
}

func TestBuildDerivativesBoard_ScanColumnWithholdsIncompleteVenue(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	coin := boardCoin("BTCUSDT", now, 8e9, 1e9, 1e9, 0, 0)
	coin.Venues[0].TopLongOK = true
	coin.Venues[0].TopLongShare = 0.70
	coin.Venues[1].TopLongOK = true
	coin.Venues[1].TopLongShare = 0.70
	coin.Venues[0].CVDDayOK = true
	coin.Venues[0].CVDAt = now
	coin.Venues[0].CVDDay = 10
	coin.Venues[1].CVDDayOK = true
	coin.Venues[1].CVDAt = now
	coin.Venues[1].CVDDay = 10
	got := BuildDerivativesBoard("all", []DerivativesCoinInput{coin}, now)
	row := got.Coins[0]
	if row.GapDefinite || row.GapPct != "" || row.CVDDayDefinite || row.CVDDay != "" {
		t.Fatalf("leaked gap %q cvd %q", row.GapPct, row.CVDDay)
	}
	if row.Venues[0].GapPct != "+15" || row.Venues[2].GapStatus != VenueReadMissing {
		t.Fatalf("venues %+v", row.Venues)
	}
}

func TestDayCVDDelta_NeedsTheDayOpen(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	day := now.Truncate(24 * time.Hour)
	buckets := []TakerBucket{
		{Start: day, BuyNotional: 30, SellNotional: 10},
		{Start: now.Add(-time.Minute), BuyNotional: 5, SellNotional: 1},
	}
	delta, newest, ok := DayCVDDelta(buckets, now, day)
	if !ok || delta != 24 || !newest.Equal(now.Add(-time.Minute)) {
		t.Fatalf("delta %v newest %v ok %v", delta, newest, ok)
	}
	if _, _, late := DayCVDDelta(buckets, now, now.Add(-time.Hour)); late {
		t.Fatal("tape that starts after the day open is not a day CVD")
	}
}

func TestBuildDerivativesBoard_IncompleteWindowIsNotZero(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	coin := boardCoin("BTCUSDT", now, 8e9, 1e9, 1e9, 0, 0)
	coin.Venues[1].OI.History = []OpenInterestPoint{
		{Time: now.Add(-1 * time.Hour), Value: 1e9, Contracts: 1e9},
	}
	got := BuildDerivativesBoard("all", []DerivativesCoinInput{coin}, now)
	row := got.Coins[0]
	if !row.OpenInterestDefinite || !row.Change1hDefinite || row.Change24hDefinite || row.Change24h != "" {
		t.Fatalf("window %+v", row)
	}
}

func boardCoin(symbol string, now time.Time, binance, bybit, okx, liqLong, liqShort float64) DerivativesCoinInput {
	return DerivativesCoinInput{
		Symbol: symbol,
		Venues: []DerivativesVenueInput{
			boardVenue(ExchangeBinance, now, binance, 0.0001, 100000, liqLong, liqShort),
			boardVenue(ExchangeBybit, now, bybit, 0.0002, 100100, liqLong, liqShort),
			boardVenue(ExchangeOKX, now, okx, -0.0001, 99900, liqLong, liqShort),
		},
		LiqFeed: LiquidationFeed{Venues: []LiquidationVenueHealth{
			{Exchange: "binance", Live: true, LastSeenAt: now},
			{Exchange: "bybit", Live: true, LastSeenAt: now},
			{Exchange: "okx", Live: true, LastSeenAt: now},
		}},
	}
}

func boardVenue(ex Exchange, now time.Time, oi, rate, mark, liqLong, liqShort float64) DerivativesVenueInput {
	past := oi / 2
	return DerivativesVenueInput{
		Exchange: ex,
		OI: &OpenInterestSeries{
			Exchange: ex,
			Current:  OpenInterestPoint{Time: now, Value: oi, Contracts: oi},
			History: []OpenInterestPoint{
				{Time: now.Add(-1 * time.Hour), Value: oi, Contracts: oi},
				{Time: now.Add(-24 * time.Hour), Value: past, Contracts: past},
			},
		},
		OIValue:     oi,
		OIAt:        now,
		OIOK:        true,
		FundingRate: rate,
		FundingAt:   now,
		FundingOK:   true,
		MarkPrice:   mark,
		LiqLong:     liqLong,
		LiqShort:    liqShort,
		LongShare:   0.55,
		LongShortAt: now,
		LongShortOK: true,
		TakerBuy:    60,
		TakerSell:   40,
		TakerOK:     true,
	}
}
