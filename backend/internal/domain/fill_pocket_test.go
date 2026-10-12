package domain

import "testing"

func TestBuildFillPocketMark_SpotFillAndLongPocket(t *testing.T) {
	got := BuildFillPocketMark(FillPocketFacts{
		Symbol: "BTCUSDT", Exchange: "binance",
		FillPrice: 100, FillKind: FillPocketSpot,
		PocketLoaded: true,
		Pocket:       CloserPocket{Covered: true, Side: SituationLiqLong, Price: 99, DistancePct: 1},
	})
	if !got.Open || got.FillPrice != 100 || got.FillKind != FillPocketSpot {
		t.Fatalf("fill %+v", got)
	}
	if got.PocketSide != SituationLiqLong || got.PocketPrice != 99 || got.PocketFromFill != FillPocketBelow {
		t.Fatalf("pocket %+v", got)
	}
	if got.GapPct == nil || *got.GapPct != 1 {
		t.Fatalf("gap %+v", got.GapPct)
	}
	if got.MarginLiqPrice != 0 || got.MarginGapPct != nil || len(got.Missing) != 0 {
		t.Fatalf("extra %+v", got)
	}
}

func TestBuildFillPocketMark_MissingVenueLeavesThePocketUnmarked(t *testing.T) {
	got := BuildFillPocketMark(FillPocketFacts{
		Symbol: "BTCUSDT", Exchange: "binance",
		FillPrice: 100, FillKind: FillPocketSpot,
		MarginLiqPrice: 80.5,
		PocketLoaded:   true,
		Pocket:         CloserPocket{Covered: false, Missing: []string{"okx"}, Price: 99, Side: SituationLiqLong},
	})
	if !got.Open || got.FillPrice != 100 || got.PocketPrice != 0 || got.GapPct != nil {
		t.Fatalf("pocket must stay off %+v", got)
	}
	if len(got.Missing) != 1 || got.Missing[0] != "okx" {
		t.Fatalf("missing %+v", got.Missing)
	}
	if got.MarginLiqPrice != 80.5 || got.MarginGapPct != nil {
		t.Fatalf("margin percent needs a pocket %+v", got)
	}
}

func TestBuildFillPocketMark_MarginLiquidationBesideThePocket(t *testing.T) {
	got := BuildFillPocketMark(FillPocketFacts{
		Symbol: "BTCUSDT", Exchange: "binance",
		FillPrice: 100, FillKind: FillPocketMargin,
		MarginLiqPrice: 80.5,
		PocketLoaded:   true,
		Pocket:         CloserPocket{Covered: true, Side: SituationLiqLong, Price: 90},
	})
	if got.PocketFromFill != FillPocketBelow || got.GapPct == nil || *got.GapPct != 10 {
		t.Fatalf("fill gap %+v", got)
	}
	// (90-80.5)/90 = 10.555...% → 10.6 below the pocket.
	if got.MarginFromPocket != FillPocketBelow || got.MarginGapPct == nil || *got.MarginGapPct != 10.6 {
		t.Fatalf("margin gap %+v", got)
	}
}

func TestBuildFillPocketMark_SamePriceKeepsZero(t *testing.T) {
	got := BuildFillPocketMark(FillPocketFacts{
		Symbol: "ETHUSDT", Exchange: "bybit",
		FillPrice: 101, FillKind: FillPocketSpot,
		PocketLoaded: true,
		Pocket:       CloserPocket{Covered: true, Side: SituationLiqShort, Price: 101},
	})
	if got.GapPct == nil || *got.GapPct != 0 || got.PocketFromFill != "" {
		t.Fatalf("zero gap %+v dir %q", got.GapPct, got.PocketFromFill)
	}
}

func TestBuildFillPocketMark_EmptyMapAndNoPosition(t *testing.T) {
	none := BuildFillPocketMark(FillPocketFacts{
		Symbol: "BTCUSDT", Exchange: "binance",
		FillPrice: 100, FillKind: FillPocketSpot,
		PocketLoaded: true,
		Pocket:       CloserPocket{Covered: true, Side: SituationLiqNone},
	})
	if none.PocketSide != SituationLiqNone || none.PocketPrice != 0 || none.GapPct != nil || len(none.Missing) != 0 {
		t.Fatalf("none %+v", none)
	}
	flat := BuildFillPocketMark(FillPocketFacts{Symbol: "BTCUSDT", Exchange: "binance", PocketLoaded: true})
	if flat.Open || flat.FillPrice != 0 {
		t.Fatalf("flat %+v", flat)
	}
}

func TestBuildFillPocketMark_UnreadMapNamesEveryVenue(t *testing.T) {
	got := BuildFillPocketMark(FillPocketFacts{
		Symbol: "BTCUSDT", Exchange: "binance",
		FillPrice: 50, FillKind: FillPocketSpot,
	})
	if len(got.Missing) != 3 || got.Missing[0] != "binance" || got.Missing[1] != "bybit" || got.Missing[2] != "okx" {
		t.Fatalf("missing %+v", got.Missing)
	}
	if got.PocketPrice != 0 {
		t.Fatalf("price %v", got.PocketPrice)
	}
}
