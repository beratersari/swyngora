package equities

import "testing"

func TestSpotFromNasdaqRowParsesMcap(t *testing.T) {
	row := spotFromNasdaqRow(nasdaqScreenerRow{
		Symbol:    "NVDA",
		LastSale:  "$224.96",
		NetChange: "-0.34",
		PctChange: "-0.151%",
		Volume:    "24_180_000",
		MarketCap: "5,444,032,000,000",
		Sector:    "Technology",
	})
	// volume with underscore won't parse — use comma form
	row = spotFromNasdaqRow(nasdaqScreenerRow{
		Symbol:    "NVDA",
		LastSale:  "$224.96",
		NetChange: "-0.34",
		PctChange: "-0.151%",
		Volume:    "24180000",
		MarketCap: "5,444,032,000,000",
		Sector:    "Technology",
	})
	if row == nil || row.Symbol != "NVDA" {
		t.Fatalf("%+v", row)
	}
	if row.MarketCapCirculating == nil || *row.MarketCapCirculating < 1e12 {
		t.Fatalf("mcap=%v", row.MarketCapCirculating)
	}
	if row.PriceChangePercent != "-0.151" {
		t.Fatalf("pct=%s", row.PriceChangePercent)
	}
	if len(row.Tags) != 1 || row.Tags[0] != "Technology" {
		t.Fatalf("tags=%v", row.Tags)
	}
}

func TestSpotFromBistScanRowParsesMcapAndSector(t *testing.T) {
	row := spotFromBistScanRow(tvScanRow{
		Symbol: "BIST:LINK",
		Values: []tvScanValue{
			{str: "LINK", ok: true},
			{num: 5.94, ok: true},
			{num: 3.66, ok: true},
			{num: 0.21, ok: true},
			{num: 46816417, ok: true},
			{num: 5109755845, ok: true},
			{str: "Technology Services", ok: true},
		},
	})
	if row == nil || row.Symbol != "LINK" || row.QuoteAsset != "TRY" {
		t.Fatalf("%+v", row)
	}
	if row.LastPrice != "5.94" || row.PriceChangePercent != "3.66" {
		t.Fatalf("px=%s pct=%s", row.LastPrice, row.PriceChangePercent)
	}
	if row.MarketCapCirculating == nil || *row.MarketCapCirculating < 5e9 || *row.MarketCapCirculating > 6e9 {
		t.Fatalf("mcap=%v", row.MarketCapCirculating)
	}
	if len(row.Tags) != 1 || row.Tags[0] != "Technology Services" {
		t.Fatalf("tags=%v", row.Tags)
	}
}

func TestSpotFromBistScanRowDropsGarbageMcap(t *testing.T) {
	row := spotFromBistScanRow(tvScanRow{
		Symbol: "BIST:FOO",
		Values: []tvScanValue{
			{str: "FOO", ok: true},
			{num: 10, ok: true},
			{num: 0, ok: true},
			{num: 0, ok: true},
			{num: 1, ok: true},
			{num: 9e13, ok: true},
		},
	})
	if row == nil || row.MarketCapCirculating != nil {
		t.Fatalf("expected dropped mcap: %+v", row)
	}
}

func TestSanitizeTicker(t *testing.T) {
	if sanitizeTicker("thyao.is") != "THYAO" {
		t.Fatal("bist suffix")
	}
	if sanitizeTicker("BRK.A") != "BRK.A" {
		t.Fatal("class share")
	}
	if sanitizeTicker("AA^B") != "" {
		t.Fatal("preferred junk")
	}
}
