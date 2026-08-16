package domain

import (
	"testing"
	"time"
)

func TestCloneHolders(t *testing.T) {
	if CloneHolders(nil) != nil {
		t.Fatal("nil should stay nil")
	}
	active := int64(9)
	top10 := 12.5
	in := &AssetHolders{
		Asset:          "BTC",
		Name:           "Bitcoin",
		HolderCount:    100,
		DailyActive:    &active,
		TopTenSharePct: &top10,
		TopHolders:     []AssetHolder{{Address: "abc", Balance: 1, SharePct: 0.5}},
		AsOf:           time.Unix(1, 0).UTC(),
		Source:         "coinmarketcap",
	}
	out := CloneHolders(in)
	out.TopHolders[0].Address = "mutated"
	*out.DailyActive = 0
	*out.TopTenSharePct = 0
	if in.TopHolders[0].Address != "abc" || *in.DailyActive != 9 || *in.TopTenSharePct != 12.5 {
		t.Fatalf("clone leaked mutations: %+v", in)
	}
}

func TestCapHolderList(t *testing.T) {
	list := make([]AssetHolder, 25)
	got := CapHolderList(list, 0)
	if len(got) != MaxHolderList {
		t.Fatalf("default cap %d", len(got))
	}
	if len(CapHolderList(list[:3], 10)) != 3 {
		t.Fatal("short list should stay short")
	}
}
