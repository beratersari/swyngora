package domain

import (
	"fmt"
	"math"
)

const (
	OptionsFindingIVRicher      = "iv_richer"
	OptionsFindingIVCheaper     = "iv_cheaper"
	OptionsFindingIVRisingCalm  = "iv_rising_spot_calm"
	OptionsFindingIVFallingHot  = "iv_falling_spot_hot"
	OptionsFindingPutsVsLongs   = "puts_vs_futures_long"
	OptionsFindingCallsVsShorts = "calls_vs_futures_short"
	OptionsFindingMoveRicher    = "expected_move_richer"
	OptionsFindingMoveCheaper   = "expected_move_cheaper"
	OptionsFindingMoveSpent     = "spot_spent_expected_move"
	OptionsFindingPainAway      = "max_pain_away"

	OptionsFindingTopicVol  = "vol"
	OptionsFindingTopicLean = "lean"
	OptionsFindingTopicPin  = "pin"

	OptionsFindingWatch   = "watch"
	OptionsFindingNotable = "notable"

	optionsVsIVRatioHigh   = 1.25
	optionsVsIVRatioLow    = 0.80
	optionsVsIVGapPts      = 5.0
	optionsVsIVMinAnn      = 5.0
	optionsVsDVOLShift     = 2.0
	optionsVsRangeShrink   = 0.85
	optionsVsRangeExpand   = 1.15
	optionsVsMoveRatioHigh = 1.45
	optionsVsMoveRatioLow  = 0.60
	optionsVsMoveGapPct    = 2.0
	optionsVsSpentDTE      = 7
	optionsVsPainGapPct    = 6.0
	optionsVsLongCrowded   = 55.0
	optionsVsShortCrowded  = 45.0
	optionsVsFundLong      = 0.0001
	optionsVsFundShort     = -0.0001
)

// OptionsMarketTape is spot + futures context used to compare with options.
type OptionsMarketTape struct {
	SpotLast        float64
	HasSpot         bool
	Range24hPct     float64 // (high−low)/last as percent
	HasRange        bool
	PrevRange24hPct float64
	HasPrevRange    bool
	FundingRate     float64 // decimal (0.0001 = 0.01%)
	HasFunding      bool
	LongPct         float64 // 0–100 account share
	HasLongPct      bool
}

// Usable is true when at least one comparison input is present.
func (t OptionsMarketTape) Usable() bool {
	return t.HasRange || t.HasFunding || t.HasLongPct
}

// OptionsVsFinding is one place options disagree with spot or futures.
type OptionsVsFinding struct {
	Kind     string
	Topic    string
	Severity string
	Title    string
	Detail   string
	Options  float64
	Market   float64
	HasNums  bool
}

// OptionsVsMarket is the options-vs-spot/futures comparison on a snapshot.
type OptionsVsMarket struct {
	Compared    bool
	SpotLast    float64
	HasSpot     bool
	ImpliedIV   float64
	RealizedAnn float64
	Range24hPct float64
	HasVol      bool
	FundingRate float64
	HasFunding  bool
	LongPct     float64
	HasLongPct  bool
	Findings    []OptionsVsFinding
	Summary     string
}

// OptionsSpotSymbol is the USDT pair used to load spot/futures tape.
func OptionsSpotSymbol(asset string) string {
	if asset == OptionsAssetETH {
		return "ETHUSDT"
	}
	return "BTCUSDT"
}

// OptionsAnnualizeRange turns a 24h high–low percent into a 365-day vol.
func OptionsAnnualizeRange(rangePct float64) float64 {
	if rangePct <= 0 {
		return 0
	}
	return rangePct * math.Sqrt(365)
}

// CompareOptionsToMarket flags only real IV / put-call disagreements.
func CompareOptionsToMarket(snap *OptionsSnapshot, tape OptionsMarketTape) *OptionsVsMarket {
	if snap == nil || !tape.Usable() {
		return nil
	}
	out := &OptionsVsMarket{
		Compared:    true,
		SpotLast:    tape.SpotLast,
		HasSpot:     tape.HasSpot,
		Range24hPct: tape.Range24hPct,
		FundingRate: tape.FundingRate,
		HasFunding:  tape.HasFunding,
		LongPct:     tape.LongPct,
		HasLongPct:  tape.HasLongPct,
	}
	if iv, ok := optionsImpliedIV(snap); ok {
		out.ImpliedIV = iv
	}
	if tape.HasRange {
		out.RealizedAnn = OptionsAnnualizeRange(tape.Range24hPct)
		out.HasVol = out.ImpliedIV > 0 && out.RealizedAnn > 0
	}

	var findings []OptionsVsFinding
	findings = append(findings, optionsVolVsRealized(out)...)
	findings = append(findings, optionsVolVsRangeTrend(snap, tape)...)
	move := optionsMoveVsRange(snap, tape)
	spent := optionsSpotSpentMove(snap, tape)
	if spent != nil {
		findings = append(findings, *spent)
		if move != nil && move.Kind != OptionsFindingMoveCheaper {
			findings = append(findings, *move)
		}
	} else if move != nil {
		findings = append(findings, *move)
	}
	if f := optionsLeanVsFutures(snap, tape); f != nil {
		findings = append(findings, *f)
	}
	if f := optionsPainVsSpot(snap); f != nil {
		findings = append(findings, *f)
	}
	out.Findings = findings
	out.Summary = optionsVsSummary(findings)
	return out
}

func optionsImpliedIV(snap *OptionsSnapshot) (float64, bool) {
	if snap.ATM != nil && snap.ATM.IV > 0 {
		return snap.ATM.IV, true
	}
	if snap.HasDVOL && snap.DVOL > 0 {
		return snap.DVOL, true
	}
	return 0, false
}

func optionsVolVsRealized(out *OptionsVsMarket) []OptionsVsFinding {
	if out == nil || !out.HasVol || out.RealizedAnn < optionsVsIVMinAnn || out.ImpliedIV < optionsVsIVMinAnn {
		return nil
	}
	ratio := out.ImpliedIV / out.RealizedAnn
	gap := math.Abs(out.ImpliedIV - out.RealizedAnn)
	switch {
	case ratio >= optionsVsIVRatioHigh && gap >= optionsVsIVGapPts:
		sev := OptionsFindingWatch
		if ratio >= 1.45 || gap >= 12 {
			sev = OptionsFindingNotable
		}
		return []OptionsVsFinding{{
			Kind: OptionsFindingIVRicher, Topic: OptionsFindingTopicVol, Severity: sev,
			Title:   "Options are pricing more volatility than spot has printed",
			Detail:  fmt.Sprintf("Implied vol %.0f%% vs 24h high–low annualized %.0f%%. The options market is richer than the swings spot just made.", out.ImpliedIV, out.RealizedAnn),
			Options: out.ImpliedIV, Market: out.RealizedAnn, HasNums: true,
		}}
	case ratio <= optionsVsIVRatioLow && gap >= optionsVsIVGapPts:
		sev := OptionsFindingWatch
		if ratio <= 0.65 || gap >= 12 {
			sev = OptionsFindingNotable
		}
		return []OptionsVsFinding{{
			Kind: OptionsFindingIVCheaper, Topic: OptionsFindingTopicVol, Severity: sev,
			Title:   "Options are cheaper than the swings spot already made",
			Detail:  fmt.Sprintf("Implied vol %.0f%% vs 24h high–low annualized %.0f%%. Listed options are not pricing the move spot has been making.", out.ImpliedIV, out.RealizedAnn),
			Options: out.ImpliedIV, Market: out.RealizedAnn, HasNums: true,
		}}
	default:
		return nil
	}
}

func optionsVolVsRangeTrend(snap *OptionsSnapshot, tape OptionsMarketTape) []OptionsVsFinding {
	if snap == nil || !snap.HasDVOLChange || !tape.HasRange || !tape.HasPrevRange || tape.PrevRange24hPct < 0.3 {
		return nil
	}
	switch {
	case snap.DVOLChange24h >= optionsVsDVOLShift && tape.Range24hPct <= optionsVsRangeShrink*tape.PrevRange24hPct:
		return []OptionsVsFinding{{
			Kind: OptionsFindingIVRisingCalm, Topic: OptionsFindingTopicVol, Severity: OptionsFindingWatch,
			Title:   "Implied vol is rising while the spot range is shrinking",
			Detail:  fmt.Sprintf("DVOL %+0.1f pts vs 24h ago, but the 24h high–low (%.1f%%) is tighter than the prior day (%.1f%%).", snap.DVOLChange24h, tape.Range24hPct, tape.PrevRange24hPct),
			Options: snap.DVOLChange24h, Market: tape.Range24hPct, HasNums: true,
		}}
	case snap.DVOLChange24h <= -optionsVsDVOLShift && tape.Range24hPct >= optionsVsRangeExpand*tape.PrevRange24hPct:
		return []OptionsVsFinding{{
			Kind: OptionsFindingIVFallingHot, Topic: OptionsFindingTopicVol, Severity: OptionsFindingWatch,
			Title:   "Implied vol is falling while spot is swinging harder",
			Detail:  fmt.Sprintf("DVOL %+0.1f pts vs 24h ago, but the 24h high–low (%.1f%%) is wider than the prior day (%.1f%%).", snap.DVOLChange24h, tape.Range24hPct, tape.PrevRange24hPct),
			Options: snap.DVOLChange24h, Market: tape.Range24hPct, HasNums: true,
		}}
	default:
		return nil
	}
}

func optionsMoveVsRange(snap *OptionsSnapshot, tape OptionsMarketTape) *OptionsVsFinding {
	if snap == nil || !tape.HasRange || tape.Range24hPct <= 0 {
		return nil
	}
	b := pickOptionsCompareExpiry(snap, true)
	if b == nil || !b.HasExpectedMove || b.DaysToExpiry < 1 {
		return nil
	}
	scaled := tape.Range24hPct * math.Sqrt(float64(b.DaysToExpiry))
	if scaled <= 0 {
		return nil
	}
	ratio := b.ExpectedMovePct / scaled
	gap := math.Abs(b.ExpectedMovePct - scaled)
	switch {
	case ratio >= optionsVsMoveRatioHigh && gap >= optionsVsMoveGapPct:
		sev := OptionsFindingWatch
		if ratio >= 1.8 {
			sev = OptionsFindingNotable
		}
		return &OptionsVsFinding{
			Kind: OptionsFindingMoveRicher, Topic: OptionsFindingTopicVol, Severity: sev,
			Title:   fmt.Sprintf("%s prices a bigger move than recent spot range", b.Label),
			Detail:  fmt.Sprintf("1-SD expected move %.1f%% vs the last 24h range scaled to %dd (%.1f%%).", b.ExpectedMovePct, b.DaysToExpiry, scaled),
			Options: b.ExpectedMovePct, Market: scaled, HasNums: true,
		}
	case ratio <= optionsVsMoveRatioLow && gap >= optionsVsMoveGapPct:
		return &OptionsVsFinding{
			Kind: OptionsFindingMoveCheaper, Topic: OptionsFindingTopicVol, Severity: OptionsFindingWatch,
			Title:   fmt.Sprintf("%s prices a quieter path than spot has been making", b.Label),
			Detail:  fmt.Sprintf("1-SD expected move %.1f%% vs the last 24h range scaled to %dd (%.1f%%).", b.ExpectedMovePct, b.DaysToExpiry, scaled),
			Options: b.ExpectedMovePct, Market: scaled, HasNums: true,
		}
	default:
		return nil
	}
}

func optionsSpotSpentMove(snap *OptionsSnapshot, tape OptionsMarketTape) *OptionsVsFinding {
	if snap == nil || !tape.HasRange || tape.Range24hPct <= 0 {
		return nil
	}
	var near *OptionsExpiryBucket
	for i := range snap.Expiries {
		e := &snap.Expiries[i]
		if e.HasExpectedMove && e.DaysToExpiry >= 0 && e.DaysToExpiry <= optionsVsSpentDTE {
			near = e
			break
		}
	}
	if near == nil || tape.Range24hPct < near.ExpectedMovePct*1.05 {
		return nil
	}
	return &OptionsVsFinding{
		Kind: OptionsFindingMoveSpent, Topic: OptionsFindingTopicVol, Severity: OptionsFindingNotable,
		Title:   fmt.Sprintf("Spot already moved more than %s still prices", near.Label),
		Detail:  fmt.Sprintf("The last 24h high–low is %.1f%%; this expiry's remaining 1-SD move is %.1f%%.", tape.Range24hPct, near.ExpectedMovePct),
		Options: near.ExpectedMovePct, Market: tape.Range24hPct, HasNums: true,
	}
}

func optionsLeanVsFutures(snap *OptionsSnapshot, tape OptionsMarketTape) *OptionsVsFinding {
	if snap == nil {
		return nil
	}
	opt := optionsDirectionalLean(snap)
	fut := futuresDirectionalLean(tape)
	switch {
	case opt == OptionsLeanPuts && fut == "long":
		sev := OptionsFindingWatch
		if tape.HasLongPct && tape.HasFunding && tape.LongPct >= optionsVsLongCrowded && tape.FundingRate >= optionsVsFundLong {
			sev = OptionsFindingNotable
		}
		return &OptionsVsFinding{
			Kind: OptionsFindingPutsVsLongs, Topic: OptionsFindingTopicLean, Severity: sev,
			Title:   "Options lean put while futures accounts are long",
			Detail:  optionsLeanDetail(snap, tape, "put open interest / richer put IV", "long"),
			Options: snap.PutCallOI, Market: tape.LongPct, HasNums: tape.HasLongPct && snap.HasPutCallOI,
		}
	case opt == OptionsLeanCalls && fut == "short":
		sev := OptionsFindingWatch
		if tape.HasLongPct && tape.HasFunding && tape.LongPct <= optionsVsShortCrowded && tape.FundingRate <= optionsVsFundShort {
			sev = OptionsFindingNotable
		}
		return &OptionsVsFinding{
			Kind: OptionsFindingCallsVsShorts, Topic: OptionsFindingTopicLean, Severity: sev,
			Title:   "Options lean call while futures accounts are short",
			Detail:  optionsLeanDetail(snap, tape, "call open interest / richer call IV", "short"),
			Options: snap.PutCallOI, Market: tape.LongPct, HasNums: tape.HasLongPct && snap.HasPutCallOI,
		}
	default:
		return nil
	}
}

func optionsPainVsSpot(snap *OptionsSnapshot) *OptionsVsFinding {
	if snap == nil || snap.IndexPrice <= 0 {
		return nil
	}
	b := pickOptionsCompareExpiry(snap, false)
	if b == nil || !b.HasMaxPain || b.MaxPain <= 0 {
		return nil
	}
	gap := (b.MaxPain - snap.IndexPrice) / snap.IndexPrice * 100
	need := optionsVsPainGapPct
	if b.HasExpectedMove && b.ExpectedMovePct > 0 {
		if floor := b.ExpectedMovePct * 0.4; floor > need {
			need = floor
		}
	}
	if math.Abs(gap) < need {
		return nil
	}
	dir := "above"
	if gap < 0 {
		dir = "below"
	}
	return &OptionsVsFinding{
		Kind: OptionsFindingPainAway, Topic: OptionsFindingTopicPin, Severity: OptionsFindingWatch,
		Title:   fmt.Sprintf("Lowest-payout strike on %s sits well %s spot", b.Label, dir),
		Detail:  fmt.Sprintf("Max pain %.0f is %+.1f%% from the index (%.0f). The listed book is not pinned at the current price.", b.MaxPain, gap, snap.IndexPrice),
		Options: b.MaxPain, Market: snap.IndexPrice, HasNums: true,
	}
}

func optionsDirectionalLean(snap *OptionsSnapshot) string {
	if snap == nil {
		return OptionsLeanBalanced
	}
	if snap.Lean == OptionsLeanPuts || snap.Lean == OptionsLeanCalls {
		return snap.Lean
	}
	if snap.HasPutCallVol {
		if v := optionsLean(snap.PutCallVolume); v == OptionsLeanPuts || v == OptionsLeanCalls {
			return v
		}
	}
	if snap.Skew != nil && (snap.Skew.Lean == OptionsLeanPuts || snap.Skew.Lean == OptionsLeanCalls) {
		return snap.Skew.Lean
	}
	return OptionsLeanBalanced
}

func futuresDirectionalLean(tape OptionsMarketTape) string {
	longish := tape.HasLongPct && tape.LongPct >= optionsVsLongCrowded
	shortish := tape.HasLongPct && tape.LongPct <= optionsVsShortCrowded
	fundLong := tape.HasFunding && tape.FundingRate >= optionsVsFundLong
	fundShort := tape.HasFunding && tape.FundingRate <= optionsVsFundShort
	switch {
	case longish || (fundLong && !shortish):
		return "long"
	case shortish || (fundShort && !longish):
		return "short"
	default:
		return "balanced"
	}
}

func optionsLeanDetail(snap *OptionsSnapshot, tape OptionsMarketTape, optSide, futSide string) string {
	pcr := "n/a"
	if snap.HasPutCallOI {
		pcr = fmt.Sprintf("%.2f", snap.PutCallOI)
	}
	ls := "n/a"
	if tape.HasLongPct {
		ls = fmt.Sprintf("%.0f%% long", tape.LongPct)
	}
	fund := "n/a"
	if tape.HasFunding {
		fund = fmt.Sprintf("%+.4f%%", tape.FundingRate*100)
	}
	return fmt.Sprintf("Options show %s (put/call OI %s). Futures look %s (%s accounts, funding %s).", optSide, pcr, futSide, ls, fund)
}

func pickOptionsCompareExpiry(snap *OptionsSnapshot, needMove bool) *OptionsExpiryBucket {
	if snap == nil || len(snap.Expiries) == 0 {
		return nil
	}
	if snap.ATM != nil {
		for i := range snap.Expiries {
			e := &snap.Expiries[i]
			if e.Expiry.Equal(snap.ATM.Expiry) && (!needMove || e.HasExpectedMove) {
				if !needMove && e.HasMaxPain {
					return e
				}
				if needMove {
					return e
				}
			}
		}
	}
	for i := range snap.Expiries {
		e := &snap.Expiries[i]
		if e.DaysToExpiry < 0 {
			continue
		}
		if needMove && e.HasExpectedMove {
			return e
		}
		if !needMove && e.HasMaxPain {
			return e
		}
	}
	return nil
}

func optionsVsSummary(findings []OptionsVsFinding) string {
	switch len(findings) {
	case 0:
		return ""
	case 1:
		return findings[0].Title
	default:
		return fmt.Sprintf("Options disagree with spot/futures on %d points.", len(findings))
	}
}
