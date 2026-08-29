package domain

import "testing"

func TestResolveHolderBalance_DustUsesShareTimesCirc(t *testing.T) {
	t.Parallel()
	circ := 1_000_000_000.0
	got := ResolveHolderBalance(0.004, 8.37, &circ)
	if got == nil || *got < 83_000_000 || *got > 84_000_000 {
		t.Fatalf("got=%v", got)
	}
}

func TestResolveHolderBalance_KeepsMatchingReported(t *testing.T) {
	t.Parallel()
	circ := 21_000_000.0
	got := ResolveHolderBalance(248_597, 1.18, &circ)
	if got == nil || *got != 248_597 {
		t.Fatalf("got=%v", got)
	}
}

func TestHolderUsdValue(t *testing.T) {
	t.Parallel()
	circ, px := 1000.0, 2.0
	got := HolderUsdValue(10, &circ, &px)
	if got == nil || *got != 200 {
		t.Fatalf("got=%v", got)
	}
	if HolderUsdValue(8.37, nil, &px) != nil {
		t.Fatal("expected nil without circ")
	}
}

func TestEnrichHolderRows(t *testing.T) {
	t.Parallel()
	circ, px := 1_000_000_000.0, 1.0
	h := &AssetHolders{TopHolders: []AssetHolder{{Balance: 0.004, SharePct: 8.37}}}
	EnrichHolderRows(h, &circ, &px)
	if h.TopHolders[0].ResolvedBalance == nil || *h.TopHolders[0].ResolvedBalance < 1 {
		t.Fatalf("resolved=%v", h.TopHolders[0].ResolvedBalance)
	}
	if h.TopHolders[0].UsdValue == nil || *h.TopHolders[0].UsdValue < 1 {
		t.Fatalf("usd=%v", h.TopHolders[0].UsdValue)
	}
}
