package domain

import (
	"testing"
	"time"
)

func TestParseDerivativesBoardLimit(t *testing.T) {
	if ParseDerivativesBoardLimit(0) != 20 || ParseDerivativesBoardLimit(2) != 5 || ParseDerivativesBoardLimit(99) != LiquidRecorderLimit {
		t.Fatalf("limits 0=%d 2=%d 99=%d", ParseDerivativesBoardLimit(0), ParseDerivativesBoardLimit(2), ParseDerivativesBoardLimit(99))
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
	}
}
