package domain

import (
	"testing"
	"time"
)

func TestBuildVolumeProfileFromCells_KeepsPrintsOnTheirPrice(t *testing.T) {
	from := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	cells := []FlowAtPricesCell{
		{Minute: from, Price: 64000, Buy: 1000, Sell: 0, BuyN: 1},
		{Minute: from.Add(time.Minute), Price: 65200, Buy: 50000, Sell: 30000, BuyN: 4, SellN: 2},
	}
	got := BuildVolumeProfileFromCells(ExchangeBinance, "BTCUSDT", cells, 100, 65500, from, from.Add(4*time.Hour), false)
	if got.Error != "" || !got.Complete || !got.BuySellKnown {
		t.Fatalf("%+v", got)
	}
	if got.POC.Price != 65200 || got.POC.BuyVolume != 50000 || got.POC.SellVolume != 30000 {
		t.Fatalf("poc %+v", got.POC)
	}
	var thin *VolumeProfileBin
	for i := range got.Bins {
		if got.Bins[i].Price == 64000 {
			thin = &got.Bins[i]
		}
	}
	if thin == nil || thin.Volume != 1000 || thin.BuyVolume != 1000 || thin.SellVolume != 0 {
		t.Fatalf("thin row %+v bins=%+v", thin, got.Bins)
	}
	if got.ValueArea.Low == 0 || got.ValueArea.High == 0 {
		t.Fatalf("va %+v", got.ValueArea)
	}
}

func TestBuildVolumeProfileFromCells_NamesALateStart(t *testing.T) {
	from := time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC)
	late := from.Add(3 * time.Hour)
	got := BuildVolumeProfileFromCells(ExchangeBybit, "BTCUSDT", []FlowAtPricesCell{
		{Minute: late, Price: 65000, Buy: 10, Sell: 4, BuyN: 1, SellN: 1},
	}, 50, 65000, from, from.Add(4*time.Hour), false)
	if got.Complete || got.Started.IsZero() {
		t.Fatalf("%+v", got)
	}
	if got.Summary == "" || got.POC.Price != 65000 {
		t.Fatalf("summary %q poc %+v", got.Summary, got.POC)
	}
}

func TestProfileBarsFromCells_GroupsAMinute(t *testing.T) {
	at := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	bars := ProfileBarsFromCells([]FlowAtPricesCell{
		{Minute: at, Price: 100, Buy: 3, Sell: 1},
		{Minute: at, Price: 110, Buy: 1, Sell: 5},
		{Minute: at.Add(time.Minute), Price: 90, Buy: 2, Sell: 2},
	})
	if len(bars) != 2 {
		t.Fatalf("%+v", bars)
	}
	if bars[0].Low != 100 || bars[0].High != 110 || bars[0].Volume != 10 || !bars[0].BuySellKnown {
		t.Fatalf("first %+v", bars[0])
	}
}
