package domain

import "testing"

func TestMergeLiquidSymbols_SeedsFirstThenRanked(t *testing.T) {
	got := MergeLiquidSymbols(
		[]string{"BTCUSDT", "ETHUSDT"},
		[]string{"ethusdt", "SOLUSDT", "XRPUSDT", ""},
		3,
	)
	if len(got) != 3 || got[0] != "BTCUSDT" || got[1] != "ETHUSDT" || got[2] != "SOLUSDT" {
		t.Fatalf("%v", got)
	}
}

func TestMergeLiquidSymbols_DefaultCapKeepsAllSeeds(t *testing.T) {
	ranked := []string{"PEPEUSDT", "WIFUSDT", "BTCUSDT"}
	got := MergeLiquidSymbols(DefaultFuturesHistorySymbols, ranked, 0)
	if len(got) != len(DefaultFuturesHistorySymbols)+2 {
		t.Fatalf("len %d %v", len(got), got)
	}
	for i, seed := range DefaultFuturesHistorySymbols {
		if got[i] != seed {
			t.Fatalf("seed %d: %s want %s", i, got[i], seed)
		}
	}
}
