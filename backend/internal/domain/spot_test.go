package domain

import "testing"

func TestIsValidSpotSortField(t *testing.T) {
	for _, f := range SupportedSpotSortFields {
		if !IsValidSpotSortField(string(f)) {
			t.Fatalf("expected valid %s", f)
		}
	}
	if IsValidSpotSortField("marketCap") || IsValidSpotSortField("") {
		t.Fatal("expected invalid")
	}
}

func TestIsValidSortOrder(t *testing.T) {
	if !IsValidSortOrder("asc") || !IsValidSortOrder("desc") {
		t.Fatal("asc/desc should be valid")
	}
	if IsValidSortOrder("up") || IsValidSortOrder("") {
		t.Fatal("invalid orders")
	}
}

func TestNeedsSupplyEnrichment(t *testing.T) {
	if !SpotSortMarketCapCirculating.NeedsSupplyEnrichment() {
		t.Fatal("mcap sort needs enrichment")
	}
	if SpotSortQuoteVolume.NeedsSupplyEnrichment() {
		t.Fatal("volume sort does not need enrichment before page")
	}
}
