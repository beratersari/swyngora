package domain

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestParseOptionInstrument(t *testing.T) {
	t.Parallel()
	asset, exp, strike, side, ok := ParseOptionInstrument("BTC-27MAR26-100000-C")
	if !ok || asset != "BTC" || side != OptionSideCall || strike != 100000 {
		t.Fatalf("got %s %s %v %v ok=%v", asset, side, strike, exp, ok)
	}
	if exp.Year() != 2026 || exp.Month() != time.March || exp.Day() != 27 || exp.Hour() != 8 {
		t.Fatalf("expiry=%v", exp)
	}
	asset, exp, strike, side, ok = ParseOptionInstrument("ETH-5SEP25-4000-P")
	if !ok || asset != "ETH" || side != OptionSidePut || strike != 4000 || exp.Day() != 5 {
		t.Fatalf("got %s %s %v %v ok=%v", asset, side, strike, exp, ok)
	}
	if _, _, _, _, ok = ParseOptionInstrument("BTC-PERPETUAL"); ok {
		t.Fatal("perpetual is not an option")
	}
	_, exp2, _, _, ok := ParseOptionInstrument("BTC-5SEP27-90000-C")
	if !ok || exp2.Day() != 5 || exp2.Month() != time.September || exp2.Year() != 2027 {
		t.Fatalf("single-digit day %v ok=%v", exp2, ok)
	}
}

func TestParseOptionsAsset(t *testing.T) {
	t.Parallel()
	got, err := ParseOptionsAsset("btcusdt")
	if err != nil || got != "BTC" {
		t.Fatalf("got %q %v", got, err)
	}
	got, err = ParseOptionsAsset("ETH")
	if err != nil || got != "ETH" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := ParseOptionsAsset("SOL"); err == nil || !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("sol err=%v", err)
	}
}

func TestParseOptionsExchange(t *testing.T) {
	t.Parallel()
	got, err := ParseOptionsExchange("")
	if err != nil || got != string(ExchangeDeribit) {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := ParseOptionsExchange("binance"); err == nil {
		t.Fatal("binance should fail")
	}
}

func TestBuildOptionsSnapshot(t *testing.T) {
	t.Parallel()
	asOf := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	near := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	far := time.Date(2026, 6, 26, 8, 0, 0, 0, time.UTC)
	quotes := []OptionQuote{
		{Instrument: "BTC-27MAR26-100000-C", Asset: "BTC", Expiry: near, Strike: 100000, Side: OptionSideCall, OpenInterest: 200, Volume: 10, VolumeUSD: 1_000_000, MarkIV: 50, Underlying: 100000},
		{Instrument: "BTC-27MAR26-100000-P", Asset: "BTC", Expiry: near, Strike: 100000, Side: OptionSidePut, OpenInterest: 100, Volume: 5, VolumeUSD: 500_000, MarkIV: 52, Underlying: 100000},
		{Instrument: "BTC-26JUN26-120000-C", Asset: "BTC", Expiry: far, Strike: 120000, Side: OptionSideCall, OpenInterest: 50, Volume: 2, VolumeUSD: 200_000, MarkIV: 48, Underlying: 100000},
	}
	snap := BuildOptionsSnapshot("BTC", quotes, 100000, OptionsDVOL{Value: 49.5, Change24h: -1.2, HasChange: true}, asOf)
	if snap.OpenInterest != 350 || snap.CallOI != 250 || snap.PutOI != 100 {
		t.Fatalf("oi=%v call=%v put=%v", snap.OpenInterest, snap.CallOI, snap.PutOI)
	}
	if !snap.HasPutCallOI || math.Abs(snap.PutCallOI-0.4) > 1e-9 {
		t.Fatalf("pcr=%v", snap.PutCallOI)
	}
	if snap.Lean != OptionsLeanCalls {
		t.Fatalf("lean=%s", snap.Lean)
	}
	if !snap.HasDVOL || snap.DVOL != 49.5 || !snap.HasDVOLChange {
		t.Fatalf("dvol=%v", snap.DVOL)
	}
	if snap.ATM == nil || math.Abs(snap.ATM.IV-51) > 1e-9 || snap.ATM.Strike != 100000 {
		t.Fatalf("atm=%+v", snap.ATM)
	}
	if len(snap.Expiries) != 2 {
		t.Fatalf("expiries=%d", len(snap.Expiries))
	}
	if snap.Expiries[0].SharePct < 80 || snap.Expiries[0].ContractCount != 2 {
		t.Fatalf("first expiry=%+v", snap.Expiries[0])
	}
	if snap.Listed != 3 || snap.WithOI != 3 || snap.WithIV != 3 {
		t.Fatalf("coverage listed=%d oi=%d iv=%d", snap.Listed, snap.WithOI, snap.WithIV)
	}
	if len(snap.TermStructure) != 2 {
		t.Fatalf("term=%d", len(snap.TermStructure))
	}
	if !snap.TermStructure[0].HasSkew || math.Abs(snap.TermStructure[0].Skew-2) > 1e-9 {
		t.Fatalf("near skew=%+v", snap.TermStructure[0])
	}
	if snap.Skew == nil || !snap.Skew.HasAtmSkew || math.Abs(snap.Skew.AtmSkew-2) > 1e-9 {
		t.Fatalf("skew=%+v", snap.Skew)
	}
	if snap.Skew.Lean != OptionsLeanPuts {
		t.Fatalf("vol lean=%s", snap.Skew.Lean)
	}
	if !snap.Skew.HasTerm || math.Abs(snap.Skew.TermSlope-(-3)) > 1e-9 {
		// near ATM 51, far only call 48
		t.Fatalf("term slope=%+v", snap.Skew)
	}
	if !snap.Expiries[0].HasSkew || snap.Expiries[0].Skew != 2 {
		t.Fatalf("expiry vol=%+v", snap.Expiries[0])
	}
	if !snap.Expiries[0].HasExpectedMove || snap.Expiries[0].ExpectedMovePct <= 0 {
		t.Fatalf("expected move=%+v", snap.Expiries[0])
	}
	if !snap.Expiries[0].HasMaxPain || snap.Expiries[0].MaxPain != 100000 {
		t.Fatalf("max pain=%+v", snap.Expiries[0])
	}
}

func TestOptionsExpectedMove(t *testing.T) {
	t.Parallel()
	abs, pct, low, high, ok := OptionsExpectedMove(100000, 50, 0.25)
	if !ok || math.Abs(pct-25) > 1e-9 || math.Abs(abs-25000) > 1e-6 {
		t.Fatalf("abs=%v pct=%v", abs, pct)
	}
	if low != 75000 || high != 125000 {
		t.Fatalf("band %v %v", low, high)
	}
	if _, _, _, _, ok = OptionsExpectedMove(100000, 50, 0); ok {
		t.Fatal("zero tenor")
	}
}

func TestOptionsMaxPainTwoStrikes(t *testing.T) {
	t.Parallel()
	asOf := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	snap := BuildOptionsSnapshot("BTC", []OptionQuote{
		{Asset: "BTC", Expiry: exp, Strike: 90_000, Side: OptionSidePut, OpenInterest: 10, Underlying: 100000},
		{Asset: "BTC", Expiry: exp, Strike: 110_000, Side: OptionSideCall, OpenInterest: 10, Underlying: 100000},
		{Asset: "BTC", Expiry: exp, Strike: 100_000, Side: OptionSideCall, OpenInterest: 1, MarkIV: 50, Underlying: 100000},
		{Asset: "BTC", Expiry: exp, Strike: 100_000, Side: OptionSidePut, OpenInterest: 1, MarkIV: 50, Underlying: 100000},
	}, 100000, OptionsDVOL{}, asOf)
	if !snap.Expiries[0].HasMaxPain || snap.Expiries[0].MaxPain != 100000 {
		t.Fatalf("pain=%+v", snap.Expiries[0])
	}
	if len(snap.Expiries[0].Strikes) != 3 {
		t.Fatalf("strikes=%d", len(snap.Expiries[0].Strikes))
	}
}

func TestOptionsWingSkew(t *testing.T) {
	t.Parallel()
	asOf := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	quotes := []OptionQuote{
		{Asset: "BTC", Expiry: exp, Strike: 100000, Side: OptionSideCall, MarkIV: 50, Underlying: 100000},
		{Asset: "BTC", Expiry: exp, Strike: 100000, Side: OptionSidePut, MarkIV: 51, Underlying: 100000},
		{Asset: "BTC", Expiry: exp, Strike: 95000, Side: OptionSidePut, MarkIV: 58, Underlying: 100000},
		{Asset: "BTC", Expiry: exp, Strike: 105000, Side: OptionSideCall, MarkIV: 47, Underlying: 100000},
	}
	snap := BuildOptionsSnapshot("BTC", quotes, 100000, OptionsDVOL{}, asOf)
	if snap.Skew == nil || !snap.Skew.HasWing || math.Abs(snap.Skew.WingSkew-11) > 1e-9 {
		t.Fatalf("wing=%+v", snap.Skew)
	}
}

func TestBuildOptionsSnapshot_PCRMissingWhenNoCalls(t *testing.T) {
	t.Parallel()
	asOf := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	snap := BuildOptionsSnapshot("BTC", []OptionQuote{{
		Asset: "BTC", Expiry: exp, Strike: 1, Side: OptionSidePut, OpenInterest: 10, Underlying: 1,
	}}, 1, OptionsDVOL{}, asOf)
	if snap.HasPutCallOI || snap.Lean != "" {
		t.Fatalf("expected omitted PCR, got %v %q", snap.HasPutCallOI, snap.Lean)
	}
}

func TestOptionsDaysToExpiry(t *testing.T) {
	t.Parallel()
	asOf := time.Date(2026, 3, 1, 15, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	if d := OptionsDaysToExpiry(exp, asOf); d != 26 {
		t.Fatalf("dte=%d", d)
	}
}
