package domain

import (
	"errors"
	"testing"
	"time"
)

func TestParseBookLiquidityWindow(t *testing.T) {
	id, dur, err := ParseBookLiquidityWindow("")
	if err != nil || id != BookLiquidityWindow5m || dur != 5*time.Minute {
		t.Fatalf("default %s %s %v", id, dur, err)
	}
	if _, _, err := ParseBookLiquidityWindow("24h"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("bad window %v", err)
	}
}

func TestAttributeBookSize_PulledVsTraded(t *testing.T) {
	added, pulled, eaten := AttributeBookSize(8_000_000, 2_000_000, 3_000_000)
	if added != 0 || pulled != 3_000_000 || eaten != 3_000_000 {
		t.Fatalf("8M→2M 3M traded: added=%v pulled=%v eaten=%v", added, pulled, eaten)
	}
	added, pulled, eaten = AttributeBookSize(8_000_000, 2_000_000, 7_000_000)
	if added != 1_000_000 || pulled != 0 || eaten != 7_000_000 {
		t.Fatalf("eaten more than shrink: added=%v pulled=%v eaten=%v", added, pulled, eaten)
	}
	added, pulled, eaten = AttributeBookSize(8_000_000, 10_000_000, 2_000_000)
	if added != 4_000_000 || pulled != 0 || eaten != 2_000_000 {
		t.Fatalf("grow: added=%v pulled=%v eaten=%v", added, pulled, eaten)
	}
}

func TestBuildBookLiquidityVenue_BidPullAndAskTrade(t *testing.T) {
	from := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	to := from.Add(5 * time.Minute)
	snaps := []BookHistorySnapshot{
		{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", SampledAt: from, GroupSize: 10,
			Bids: []BookHistoryLevel{{Price: 65500, Notional: 8_000_000}},
			Asks: []BookHistoryLevel{{Price: 65520, Notional: 4_000_000}},
		},
		{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", SampledAt: to, GroupSize: 10,
			Bids: []BookHistoryLevel{{Price: 65500, Notional: 2_000_000}},
			Asks: []BookHistoryLevel{{Price: 65520, Notional: 1_000_000}},
		},
	}
	prints := []TakerPrint{
		{Side: TakerSideSell, Price: 65500, Notional: 3_000_000, Time: from.Add(2 * time.Minute)},
		{Side: TakerSideBuy, Price: 65520, Notional: 3_000_000, Time: from.Add(3 * time.Minute)},
	}
	got := BuildBookLiquidityVenue(ExchangeBinance, "BTCUSDT", snaps, prints, 0, 0, from, to)
	if got.Error != "" {
		t.Fatal(got.Error)
	}
	if len(got.Bids.Levels) != 1 || got.Bids.Levels[0].Pulled != 3_000_000 || got.Bids.Levels[0].Traded != 3_000_000 {
		t.Fatalf("bids %+v", got.Bids)
	}
	if got.Bids.Read != BookLiquidityReadMixed {
		t.Fatalf("bid read %s", got.Bids.Read)
	}
	if len(got.Asks.Levels) != 1 || got.Asks.Levels[0].Pulled != 0 || got.Asks.Levels[0].Traded != 3_000_000 {
		t.Fatalf("asks %+v", got.Asks)
	}
	if got.Asks.Read != BookLiquidityReadTraded {
		t.Fatalf("ask read %s", got.Asks.Read)
	}
	if len(got.Series) != 1 || got.Series[0].BidPulled != 3_000_000 || got.Series[0].AskTraded != 3_000_000 {
		t.Fatalf("series %+v", got.Series)
	}
}

func TestBuildBookLiquidityVenue_AllPulledWhenNoPrints(t *testing.T) {
	from := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	to := from.Add(5 * time.Minute)
	snaps := []BookHistorySnapshot{
		{SampledAt: from, GroupSize: 1, Bids: []BookHistoryLevel{{Price: 100, Notional: 50}}},
		{SampledAt: to, GroupSize: 1, Bids: []BookHistoryLevel{{Price: 100, Notional: 10}}},
	}
	got := BuildBookLiquidityVenue(ExchangeBinance, "ETHUSDT", snaps, nil, 0, 0, from, to)
	if got.Bids.Pulled != 40 || got.Bids.Traded != 0 || got.Bids.Read != BookLiquidityReadPulled {
		t.Fatalf("%+v", got.Bids)
	}
}

func TestBuildBookLiquidityVenue_NeedTwoBooks(t *testing.T) {
	now := time.Now().UTC()
	got := BuildBookLiquidityVenue(ExchangeBinance, "BTCUSDT", []BookHistorySnapshot{
		{SampledAt: now, Bids: []BookHistoryLevel{{Price: 1, Notional: 1}}},
	}, nil, 0, 0, now.Add(-time.Minute), now)
	if got.Error == "" {
		t.Fatalf("%+v", got)
	}
}

func TestCombineBookLiquidityVenues(t *testing.T) {
	from := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	to := from.Add(5 * time.Minute)
	mk := func(ex Exchange, bidFrom, bidTo, traded float64) BookLiquidityVenue {
		return BuildBookLiquidityVenue(ex, "BTCUSDT", []BookHistorySnapshot{
			{SampledAt: from, GroupSize: 10, Bids: []BookHistoryLevel{{Price: 65500, Notional: bidFrom}}},
			{SampledAt: to, GroupSize: 10, Bids: []BookHistoryLevel{{Price: 65500, Notional: bidTo}}},
		}, []TakerPrint{{Side: TakerSideSell, Price: 65500, Notional: traded, Time: from.Add(time.Minute)}}, 0, 0, from, to)
	}
	got := CombineBookLiquidityVenues("BTCUSDT", []BookLiquidityVenue{
		mk(ExchangeBinance, 8_000_000, 2_000_000, 3_000_000),
		mk(ExchangeBybit, 2_000_000, 1_000_000, 500_000),
	}, from, to)
	if got == nil || got.Bids.Traded != 3_500_000 || got.Bids.Pulled != 3_500_000 {
		t.Fatalf("%+v", got)
	}
}
