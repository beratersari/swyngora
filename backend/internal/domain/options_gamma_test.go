package domain

import (
	"math"
	"testing"
	"time"
)

func TestBlackScholesGammaATM(t *testing.T) {
	t.Parallel()
	// ATM (S=K): d1 = σ√T/2, γ = n(d1) / (S σ √T).
	got := BlackScholesGamma(100000, 100000, 0.50, 0.25)
	d1 := 0.50 * 0.50 / 2
	nd1 := math.Exp(-0.5*d1*d1) / math.Sqrt(2*math.Pi)
	want := nd1 / (100000 * 0.50 * 0.50)
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("gamma=%v want=%v", got, want)
	}
	if BlackScholesGamma(100000, 100000, 0.50, 0) != 0 {
		t.Fatal("zero tenor")
	}
	call := BlackScholesGamma(100000, 95000, 0.50, 0.25)
	put := BlackScholesGamma(100000, 95000, 0.50, 0.25)
	if call != put || call <= 0 {
		t.Fatalf("call/put gamma %v %v", call, put)
	}
}

func TestDealerGammaUSDSigns(t *testing.T) {
	t.Parallel()
	call, put := DealerGammaUSD(100000, 100000, 0.50, 0.25, 10, OptionSideCall)
	if call >= 0 || put != 0 {
		t.Fatalf("short-call should be negative: call=%v put=%v", call, put)
	}
	call, put = DealerGammaUSD(100000, 100000, 0.50, 0.25, 10, OptionSidePut)
	if put <= 0 || call != 0 {
		t.Fatalf("long-put should be positive: call=%v put=%v", call, put)
	}
}

func TestBuildOptionsGammaMoreCallsIsShort(t *testing.T) {
	t.Parallel()
	asOf := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	quotes := []OptionQuote{
		{Expiry: exp, Strike: 100000, Side: OptionSideCall, OpenInterest: 200, MarkIV: 50, Underlying: 100000},
		{Expiry: exp, Strike: 100000, Side: OptionSidePut, OpenInterest: 50, MarkIV: 50, Underlying: 100000},
	}
	g := BuildOptionsGamma(quotes, 100000, asOf)
	if g == nil {
		t.Fatal("nil gamma")
	}
	if g.NetUSD >= 0 || g.Regime != GammaRegimeShort {
		t.Fatalf("more call OI should be short gamma: %+v", g)
	}
	if g.Used != 2 {
		t.Fatalf("used=%d", g.Used)
	}
}

func TestBuildOptionsGammaMorePutsIsLong(t *testing.T) {
	t.Parallel()
	asOf := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	quotes := []OptionQuote{
		{Expiry: exp, Strike: 100000, Side: OptionSideCall, OpenInterest: 50, MarkIV: 50, Underlying: 100000},
		{Expiry: exp, Strike: 100000, Side: OptionSidePut, OpenInterest: 200, MarkIV: 50, Underlying: 100000},
	}
	g := BuildOptionsGamma(quotes, 100000, asOf)
	if g == nil || g.NetUSD <= 0 || g.Regime != GammaRegimeLong {
		t.Fatalf("more put OI should be long gamma: %+v", g)
	}
}

func TestBuildOptionsGammaFlipBetweenPutAndCallWalls(t *testing.T) {
	t.Parallel()
	asOf := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	quotes := []OptionQuote{
		{Expiry: exp, Strike: 90000, Side: OptionSidePut, OpenInterest: 300, MarkIV: 50},
		{Expiry: exp, Strike: 110000, Side: OptionSideCall, OpenInterest: 300, MarkIV: 50},
	}
	g := BuildOptionsGamma(quotes, 100000, asOf)
	if g == nil || !g.HasFlip {
		t.Fatalf("expected a flip: %+v", g)
	}
	if g.Flip < 90000 || g.Flip > 110000 {
		t.Fatalf("flip=%v outside 90k–110k", g.Flip)
	}
	// Below the put wall the book should be long gamma; above the call wall, short.
	var low, high OptionsGammaLevel
	for _, p := range g.Profile {
		if p.Spot == 90000 {
			low = p
		}
		if p.Spot == 110000 {
			high = p
		}
	}
	if low.NetUSD <= 0 || high.NetUSD >= 0 {
		t.Fatalf("profile low=%+v high=%+v", low, high)
	}
	if g.FlipSide != GammaFlipBelow && g.FlipSide != GammaFlipAbove && g.FlipSide != GammaFlipAt {
		t.Fatalf("flipSide=%s", g.FlipSide)
	}
}

func TestBuildOptionsGammaSkipsMissingIV(t *testing.T) {
	t.Parallel()
	asOf := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	quotes := []OptionQuote{
		{Expiry: exp, Strike: 100000, Side: OptionSideCall, OpenInterest: 10, MarkIV: 50},
		{Expiry: exp, Strike: 101000, Side: OptionSidePut, OpenInterest: 8},
	}
	g := BuildOptionsGamma(quotes, 100000, asOf)
	if g == nil || g.Used != 1 || g.Skipped != 1 {
		t.Fatalf("used=%d skipped=%d", g.Used, g.Skipped)
	}
}

func TestBuildOptionsSnapshotIncludesGamma(t *testing.T) {
	t.Parallel()
	asOf := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	near := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	quotes := []OptionQuote{
		{Asset: "BTC", Expiry: near, Strike: 90000, Side: OptionSidePut, OpenInterest: 200, MarkIV: 52, Underlying: 100000},
		{Asset: "BTC", Expiry: near, Strike: 100000, Side: OptionSideCall, OpenInterest: 80, MarkIV: 50, Underlying: 100000},
		{Asset: "BTC", Expiry: near, Strike: 100000, Side: OptionSidePut, OpenInterest: 40, MarkIV: 52, Underlying: 100000},
		{Asset: "BTC", Expiry: near, Strike: 110000, Side: OptionSideCall, OpenInterest: 200, MarkIV: 48, Underlying: 100000},
	}
	snap := BuildOptionsSnapshot("BTC", quotes, 100000, OptionsDVOL{Value: 49}, asOf)
	if snap.Gamma == nil || snap.Gamma.Used < 4 {
		t.Fatalf("gamma=%+v", snap.Gamma)
	}
	if !snap.Expiries[0].HasGamma {
		t.Fatalf("expiry gamma missing %+v", snap.Expiries[0])
	}
	var withGamma int
	for _, s := range snap.Expiries[0].Strikes {
		if s.HasGamma {
			withGamma++
		}
	}
	if withGamma == 0 {
		t.Fatalf("no strike gamma %+v", snap.Expiries[0].Strikes)
	}
}

func TestInterpolateGammaFlipPrefersPlusToMinus(t *testing.T) {
	t.Parallel()
	profile := []OptionsGammaLevel{
		{Spot: 90, NetUSD: 10},
		{Spot: 95, NetUSD: -10},
		{Spot: 105, NetUSD: 8},
		{Spot: 110, NetUSD: -8},
	}
	flip, ok := interpolateGammaFlip(profile, 104)
	if !ok {
		t.Fatal("no flip")
	}
	// Ignore the −→+ at 100. Nearest +→− to 104 is 105→110 (107.5).
	if math.Abs(flip-107.5) > 1e-9 {
		t.Fatalf("flip=%v", flip)
	}
}
