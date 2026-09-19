package domain

import (
	"fmt"
	"strings"
)

const forensicsWhyCaution = "This is a read of the tape, not a prediction or financial advice. Missing book, open interest, options samples (BTC/ETH only), or a short after-window can change the story."

// ExplainForensicsWhy writes why the move may have happened in plain language.
// It interprets the before / during / after tape and similar past cases.
func ExplainForensicsWhy(r ForensicsReport) ForensicsWhy {
	venue := forensicsTapeVenue(r)
	if venue == nil {
		return ForensicsWhy{
			Headline: "There is not enough tape around that time to explain the move.",
			Caution:  forensicsWhyCaution,
		}
	}
	before, _ := AroundPhaseByID(*venue, AroundPhaseBefore)
	during, _ := AroundPhaseByID(*venue, AroundPhaseDuring)
	after, _ := AroundPhaseByID(*venue, AroundPhaseAfter)
	name := prettyBase(r.Symbol)
	if name == "" {
		name = "This coin"
	}

	out := ForensicsWhy{
		Before:  explainForensicsBefore(name, before, during),
		During:  explainForensicsDuring(name, before, during, r.Tape),
		After:   explainForensicsAfter(name, during, after, venue.Changes),
		Similar: explainForensicsSimilar(name, r.Similar, r.SimilarError),
		Caution: forensicsWhyCaution,
	}
	out.Headline = explainForensicsHeadline(name, during, before, out.Similar)
	return out
}

func forensicsTapeVenue(r ForensicsReport) *AroundVenue {
	if r.Tape == nil {
		return nil
	}
	if r.Tape.Combined != nil && r.Tape.Combined.Error == "" {
		return r.Tape.Combined
	}
	for i := range r.Tape.Venues {
		if r.Tape.Venues[i].Error == "" && len(r.Tape.Venues[i].Phases) > 0 {
			return &r.Tape.Venues[i]
		}
	}
	return nil
}

func explainForensicsHeadline(name string, during, before AroundPhase, similar string) string {
	if !during.Complete && !before.Complete {
		return name + " does not have a complete tape around that time."
	}
	move := during
	if !move.Complete {
		move = before
	}
	head := name + " " + forensicsMovedPhrase(move)
	if clause := forensicsVolumeClause(move); clause != "" {
		head += " " + clause
	}
	head = ensureSentence(head)
	if setup := forensicsHeadlineSetup(before, during); setup != "" {
		head += " " + setup
	}
	if lean := forensicsSimilarLeanSentence(similar); lean != "" {
		head += " " + lean
	}
	return head
}

func explainForensicsBefore(name string, before, during AroundPhase) string {
	if !before.Complete {
		return "The window before the move is incomplete, so the setup is only partly visible."
	}
	var s []string
	if before.Price.Direction == CVDDirUp || before.Price.Direction == CVDDirDown {
		s = append(s, fmt.Sprintf("Before %s moved, price was already %s — this was not a dead-still market.",
			name, forensicsDriftWord(before.Price.Direction)))
	} else {
		s = append(s, fmt.Sprintf("Before %s moved, price was mostly chopping in place.", name))
	}
	if v := forensicsVolumeStory(before, "ahead of the push"); v != "" {
		s = append(s, v)
	}
	if t := forensicsTakerStory(before, false); t != "" {
		s = append(s, t)
	}
	if b := forensicsBookStory(before); b != "" {
		s = append(s, b)
	}
	if o := forensicsPositionStory(before, during); o != "" {
		s = append(s, o)
	}
	if o := forensicsOptionsStory(before, true); o != "" {
		s = append(s, o)
	}
	if c := forensicsCurveStory(before, nil); c != "" {
		s = append(s, c)
	}
	if e := forensicsEventStory(before, true); e != "" {
		s = append(s, e)
	}
	return strings.Join(s, " ")
}

func explainForensicsDuring(name string, before, during AroundPhase, tape *AroundReport) string {
	if !during.Complete {
		return "The move window itself is incomplete (it may still be running), so the during-read is partial."
	}
	var s []string
	s = append(s, fmt.Sprintf("During the move, %s %s.", name, forensicsMovedPhrase(during)))
	if v := forensicsVolumeStory(during, "while price was moving"); v != "" {
		s = append(s, v)
	} else if before.Complete && before.Flow.Volume > 0 && during.Flow.Volume > before.Flow.Volume*1.4 {
		s = append(s, "Participation picked up versus the prior window, so this looks more like a real push than a thin drift.")
	}
	if t := forensicsTakerStory(during, true); t != "" {
		s = append(s, t)
	}
	if o := forensicsDuringPositionStory(during); o != "" {
		s = append(s, o)
	}
	if o := forensicsOptionsStory(during, false); o != "" {
		s = append(s, o)
	}
	if c := forensicsCurveStory(during, &before); c != "" {
		s = append(s, c)
	}
	if e := forensicsEventStory(during, false); e != "" {
		s = append(s, e)
	}
	if tape != nil && tape.VenueTiming != nil && tape.VenueTiming.Summary != "" {
		s = append(s, tape.VenueTiming.Summary)
	}
	if during.Flow.VWAP > 0 && during.Flow.VsVWAP == VolumeProfileVsAbove {
		s = append(s, "Price finished the window above VWAP, so the average participant was still underwater if they faded the move.")
	} else if during.Flow.VWAP > 0 && during.Flow.VsVWAP == VolumeProfileVsBelow {
		s = append(s, "Price finished the window below VWAP, so the average participant was still underwater if they bought the dip.")
	}
	return strings.Join(s, " ")
}

func explainForensicsAfter(name string, during, after AroundPhase, changes []AroundChange) string {
	if !after.Complete {
		return "The after-window is still filling in, so follow-through is not fully visible yet."
	}
	var s []string
	path := ""
	for _, ch := range changes {
		if ch.Metric == "price" {
			path = ch.Path
			break
		}
	}
	switch path {
	case AroundPathContinued:
		s = append(s, fmt.Sprintf("After the move, %s kept going the same way — the impulse did not immediately die.", name))
	case AroundPathReversed:
		s = append(s, fmt.Sprintf("After the move, %s reversed. What looked like a break often became a give-back.", name))
	case AroundPathFaded:
		s = append(s, fmt.Sprintf("After the move, %s went quiet. The push happened, then interest faded.", name))
	default:
		if after.Price.Direction == CVDDirUp || after.Price.Direction == CVDDirDown {
			s = append(s, fmt.Sprintf("After the move, price %s without a clean continuation or reversal.", forensicsDriftWord(after.Price.Direction)))
		} else {
			s = append(s, "After the move, price chopped rather than choosing a side.")
		}
	}
	if during.Complete && during.Flow.Volume > 0 && after.Flow.Volume > 0 && after.Flow.Volume < during.Flow.Volume*0.6 {
		s = append(s, "Volume dried up versus the impulse, which often means the urgent flow had already passed.")
	} else if during.Complete && after.Flow.TypicalKnown && after.Flow.VolumeRatio >= 1.5 {
		s = append(s, "Volume stayed elevated after the impulse, so the market was still working.")
	}
	if during.Complete && during.Flow.BuySellKnown && after.Flow.BuySellKnown &&
		during.Flow.Dominant != "" && after.Flow.Dominant != "" &&
		during.Flow.Dominant != TakerSideEven && after.Flow.Dominant != TakerSideEven &&
		during.Flow.Dominant != after.Flow.Dominant {
		s = append(s, "Aggressive flow flipped afterward, which is how a one-sided push often gets rejected.")
	}
	if o := forensicsAfterPositionStory(during, after); o != "" {
		s = append(s, o)
	}
	if e := forensicsEventStory(after, false); e != "" {
		s = append(s, e)
	}
	return strings.Join(s, " ")
}

func explainForensicsSimilar(name string, sim *AroundSimilarReport, simErr string) string {
	if strings.TrimSpace(simErr) != "" {
		return "Similar past moves could not be loaded, so this read stands on this tape alone."
	}
	if sim == nil {
		return ""
	}
	if len(sim.Matches) == 0 {
		if len(sim.Skipped) > 0 {
			return "Past setups that looked related did not have enough overlapping data to compare honestly, so they were left out rather than forced into the story."
		}
		return fmt.Sprintf("No recent important-move setup for %s looked similar enough to this before-window, so history does not back this read.", name)
	}
	n := sim.Events
	if n == 0 {
		n = len(sim.Matches)
	}
	h := pickForensicsWhyHorizon(sim.AfterHorizons)
	var s []string
	if h.Sample > 0 || h.Events > 0 {
		up, down := h.Up, h.Down
		total := up + down
		if total == 0 {
			total = h.Sample
		}
		if total > 0 && up*100 >= 65*total {
			s = append(s, fmt.Sprintf("When the tape looked like this before, %s more often kept rising over the next %s than it reversed.", name, h.Horizon))
		} else if total > 0 && down*100 >= 65*total {
			s = append(s, fmt.Sprintf("When the tape looked like this before, %s more often fell over the next %s than it continued higher.", name, h.Horizon))
		} else if total > 0 {
			s = append(s, fmt.Sprintf("Similar past cases split after %s — this setup has not reliably picked a side.", h.Horizon))
		}
	} else if sim.UpAfter+sim.DownAfter > 0 {
		total := sim.UpAfter + sim.DownAfter
		if sim.UpAfter*100 >= 65*total {
			s = append(s, fmt.Sprintf("When the tape looked like this before, %s more often rose after the setup than it fell.", name))
		} else if sim.DownAfter*100 >= 65*total {
			s = append(s, fmt.Sprintf("When the tape looked like this before, %s more often fell after the setup than it rose.", name))
		} else {
			s = append(s, "Similar past cases split afterward — this setup has not reliably picked a side.")
		}
	}
	top := sim.Matches[0]
	if top.AfterDirection == CVDDirUp {
		s = append(s, "The closest past case also pushed higher after the setup.")
	} else if top.AfterDirection == CVDDirDown {
		s = append(s, "The closest past case sold off after the setup.")
	}
	if n < 5 {
		s = append(s, fmt.Sprintf("That is only %d unique past event(s), so treat the comparison as a hint, not a base rate.", n))
	} else {
		s = append(s, fmt.Sprintf("This comparison uses %d unique past events.", n))
	}
	return strings.Join(s, " ")
}

func forensicsMovedPhrase(p AroundPhase) string {
	if !p.Complete {
		return "did not have a complete window"
	}
	switch p.Price.Direction {
	case CVDDirUp:
		return "rose"
	case CVDDirDown:
		return "fell"
	default:
		return "did not trend much"
	}
}

func forensicsDriftWord(dir string) string {
	switch dir {
	case CVDDirUp:
		return "drifting higher"
	case CVDDirDown:
		return "drifting lower"
	default:
		return "chopping"
	}
}

func forensicsVolumeClause(p AroundPhase) string {
	if !p.Complete || !p.Flow.TypicalKnown {
		return ""
	}
	switch p.Flow.VolumeGrade {
	case VolumeSurgeExtreme, VolumeSurgeHigh:
		return "on unusually heavy trading."
	case VolumeSurgeElevated:
		return "on busier-than-usual trading."
	default:
		if p.Flow.VolumeRatio > 0 && p.Flow.VolumeRatio < 0.7 {
			return "on lighter-than-usual trading."
		}
	}
	return ""
}

func forensicsHeadlineSetup(before, during AroundPhase) string {
	if !before.Complete {
		return ""
	}
	if o := before.Options; o != nil && o.Unusual && o.Started != AroundOptionsStartedDuring {
		s := "Listed options were already unusual before the price push."
		if o.LeadLabel != "" {
			s = "Listed options were already unusual " + o.LeadLabel + " before the price push."
		}
		if o.Together >= 3 {
			return strings.TrimSuffix(s, ".") + " Open interest, put/call, and implied vol all moved together — a stronger print than a single series."
		}
		if o.Together == 2 {
			s = strings.TrimSuffix(s, ".") + " Two options series moved together — more important than a one-off print."
		}
		if o.AfterSummary != "" {
			return strings.TrimSuffix(s, ".") + " " + o.AfterSummary
		}
		return s
	}
	if o := during.Options; o != nil && o.Unusual && o.Started == AroundOptionsStartedDuring {
		return "Listed options only became unusual while price was already moving."
	}
	if before.Book != nil && before.Book.Complete && before.Book.BidNotionalDelta < 0 && during.Complete && during.Price.Direction == CVDDirUp {
		return "Bids had already thinned, so there was less support under the market."
	}
	if before.Book != nil && before.Book.Complete && before.Book.AskNotionalDelta < 0 && during.Complete && during.Price.Direction == CVDDirDown {
		return "Offers had already thinned, so there was less resistance above."
	}
	if f := before.Futures; f != nil && f.Complete && during.Complete {
		if during.Price.Direction == CVDDirUp && f.OIDirection == CVDDirDown {
			return "Open interest was already falling, which often means covering rather than a fresh chase."
		}
		if during.Price.Direction == CVDDirUp && f.OIDirection == CVDDirUp {
			return "Open interest was already building, which often means new positions were being added."
		}
	}
	return ""
}

func forensicsSimilarLeanSentence(similar string) string {
	s := strings.ToLower(similar)
	if strings.Contains(s, "more often kept rising") || strings.Contains(s, "more often rose") {
		return "Similar past setups more often kept going up than they reversed."
	}
	if strings.Contains(s, "more often fell") {
		return "Similar past setups more often sold off afterward."
	}
	if strings.Contains(s, "split") {
		return "Similar past setups did not agree on what came next."
	}
	return ""
}

func forensicsVolumeStory(p AroundPhase, when string) string {
	if !p.Complete || !p.Flow.TypicalKnown {
		return ""
	}
	switch p.Flow.VolumeGrade {
	case VolumeSurgeExtreme:
		return fmt.Sprintf("Trading was extreme versus this coin's own typical %s — almost everyone who wanted to act was already in the tape.", when)
	case VolumeSurgeHigh:
		return fmt.Sprintf("Trading was much heavier than this coin usually sees %s, so the move had real participation.", when)
	case VolumeSurgeElevated:
		return fmt.Sprintf("Trading was already busier than usual %s.", when)
	default:
		if p.Flow.VolumeRatio > 0 && p.Flow.VolumeRatio < 0.7 {
			return fmt.Sprintf("Trading was lighter than usual %s, so a thin book can exaggerate the print.", when)
		}
	}
	return ""
}

func forensicsTakerStory(p AroundPhase, during bool) string {
	if !p.Complete || !p.Flow.BuySellKnown || p.Flow.Dominant == "" || p.Flow.Dominant == TakerSideEven {
		return ""
	}
	if p.Flow.Dominant == TakerSideBuy || p.Flow.Dominant == "buy" {
		if during {
			return "Aggressive buyers were lifting the offer — this was demand hitting the market, not only passive bids sitting there."
		}
		return "Aggressive buyers were already lifting the offer, so demand was present before the main window."
	}
	if during {
		return "Aggressive sellers were hitting the bid — this was supply hitting the market, not only offers sitting there."
	}
	return "Aggressive sellers were already hitting the bid, so supply was present before the main window."
}

func forensicsBookStory(p AroundPhase) string {
	if p.Book == nil || !p.Book.Complete {
		return ""
	}
	b := p.Book
	var bits []string
	if b.BidNotionalDelta < 0 && (b.AskNotionalDelta >= 0 || -b.BidNotionalDelta > b.AskNotionalDelta) {
		bits = append(bits, "resting bids were pulled, so there was less support under the market")
	} else if b.BidNotionalDelta > 0 && b.BidNotionalDelta > absFloat(b.AskNotionalDelta) {
		bits = append(bits, "resting bids were added, so someone was willing to defend lower")
	}
	if b.AskNotionalDelta < 0 && (b.BidNotionalDelta >= 0 || -b.AskNotionalDelta > b.BidNotionalDelta) {
		bits = append(bits, "resting offers were pulled, so the market was thinner above")
	} else if b.AskNotionalDelta > 0 && b.AskNotionalDelta > absFloat(b.BidNotionalDelta) {
		bits = append(bits, "resting offers were added, so someone was willing to cap the upside")
	}
	if b.WallsRemoved > 0 && b.WallsRemoved > b.WallsAdded {
		bits = append(bits, "large resting size came off the book")
	}
	if len(bits) == 0 {
		if strings.TrimSpace(b.Summary) != "" {
			return "The stored book changed: " + strings.TrimSuffix(strings.TrimSpace(b.Summary), ".") + "."
		}
		return ""
	}
	return "On the stored book, " + joinList(bits) + "."
}

func forensicsPositionStory(before, during AroundPhase) string {
	f := before.Futures
	if f == nil || !f.Complete {
		return ""
	}
	var s []string
	dir := during.Price.Direction
	if !during.Complete {
		dir = before.Price.Direction
	}
	switch {
	case dir == CVDDirUp && f.OIDirection == CVDDirUp:
		s = append(s, "Open interest was rising into the move, which often means new longs were being added — not only shorts covering.")
	case dir == CVDDirUp && f.OIDirection == CVDDirDown:
		s = append(s, "Open interest was falling while price was firm, which often means shorts were covering rather than a fresh long chase.")
	case dir == CVDDirDown && f.OIDirection == CVDDirUp:
		s = append(s, "Open interest was rising as price softened, which often means new shorts were being added.")
	case dir == CVDDirDown && f.OIDirection == CVDDirDown:
		s = append(s, "Open interest was falling with price, which often means longs were closing or getting forced out.")
	}
	if f.ShortLiq > 0 && f.ShortLiq > f.LongLiq*1.3 {
		s = append(s, "Short liquidations were already larger than long ones — a squeeze can force extra buying.")
	} else if f.LongLiq > 0 && f.LongLiq > f.ShortLiq*1.3 {
		s = append(s, "Long liquidations were already larger than short ones — forced selling can extend a drop.")
	}
	if f.FundingTo > f.FundingFrom && f.FundingTo > 0 {
		s = append(s, "Funding moved more positive, so the long side was paying to stay in — a crowded long can fade once the push stalls.")
	} else if f.FundingTo < f.FundingFrom && f.FundingTo < 0 {
		s = append(s, "Funding moved more negative, so the short side was paying to stay in — a crowded short can squeeze once selling dries up.")
	}
	return strings.Join(s, " ")
}

func forensicsCurveStory(p AroundPhase, prior *AroundPhase) string {
	c := p.Curve
	if c == nil || !c.Complete {
		return ""
	}
	if prior != nil && prior.Curve != nil && prior.Curve.Complete && prior.Curve.Shape != "" && prior.Curve.Shape != c.Shape {
		return ensureSentence(fmt.Sprintf("During the move the listed futures curve flipped from %s to %s", prior.Curve.Shape, c.Shape))
	}
	if p.Phase != AroundPhaseBefore {
		return ""
	}
	near, far := c.NearLabel, c.FarLabel
	if near == "" {
		near = "near"
	}
	if far == "" {
		far = "far"
	}
	gap := FormatSignedPct(absFloat(c.SpreadPct))
	switch c.Shape {
	case BasisCurveContango:
		return ensureSentence(fmt.Sprintf("The listed futures curve was in contango going into the move — far (%s) was %s%% richer than near (%s)", far, gap, near))
	case BasisCurveBackwardation:
		return ensureSentence(fmt.Sprintf("The listed futures curve was in backwardation going into the move — near (%s) was %s%% richer than far (%s)", near, gap, far))
	default:
		return ""
	}
}

func forensicsOptionsStory(p AroundPhase, before bool) string {
	o := p.Options
	if o == nil || !o.Complete {
		return ""
	}
	var bits []string
	if o.Unusual {
		switch {
		case o.Started == AroundOptionsStartedDuring:
			bits = append(bits, "Listed options only became unusual while price was already moving — this print started with the spot impulse, not ahead of it")
		case before:
			lead := "Listed options were already unusual before the move"
			if o.LeadLabel != "" {
				lead = "Listed options were already unusual " + o.LeadLabel + " before the move"
			}
			if o.Together >= 3 {
				bits = append(bits, lead+" — open interest, put/call, and implied vol all moved together, which is a stronger options print than a single series")
			} else if o.Together == 2 {
				bits = append(bits, lead+" — two options series moved together, more important than a one-off print")
			} else {
				bits = append(bits, lead+" — that kind of open-interest, put/call, or implied-vol jump sometimes shows up ahead of a spot impulse")
			}
		case o.Started == AroundOptionsStartedBefore:
			bits = append(bits, "The options print had already started before the price move and was still running")
		default:
			bits = append(bits, "Listed options stayed unusual while price was moving")
		}
	}
	if o.HasOI && o.OIDirection == CVDDirUp && absFloat(o.OIChangePct) >= aroundOptionsOIFloor {
		bits = append(bits, fmt.Sprintf("options open interest rose %s%%", FormatSignedPct(o.OIChangePct)))
	} else if o.HasOI && o.OIDirection == CVDDirDown && absFloat(o.OIChangePct) >= aroundOptionsOIFloor {
		bits = append(bits, fmt.Sprintf("options open interest fell %s%%", FormatSignedPct(o.OIChangePct)))
	}
	if o.HasPutCall && o.LeanFrom != "" && o.LeanTo != "" && o.LeanFrom != o.LeanTo {
		bits = append(bits, fmt.Sprintf("put/call lean shifted from %s toward %s", o.LeanFrom, o.LeanTo))
	} else if o.LeanTo == OptionsLeanPuts {
		bits = append(bits, "the options book leaned puts")
	} else if o.LeanTo == OptionsLeanCalls {
		bits = append(bits, "the options book leaned calls")
	}
	if o.HasDVOL && o.DVOLDelta >= aroundOptionsDVOLFloor {
		bits = append(bits, fmt.Sprintf("implied vol rose %.1f points", o.DVOLDelta))
	} else if o.HasDVOL && o.DVOLDelta <= -aroundOptionsDVOLFloor {
		bits = append(bits, fmt.Sprintf("implied vol cooled %.1f points", -o.DVOLDelta))
	}
	if before {
		if g := explainAroundGamma(*o); g != "" {
			bits = append(bits, g)
		}
	}
	if o.AfterSummary != "" {
		bits = append(bits, strings.TrimSuffix(o.AfterSummary, "."))
	}
	if len(bits) == 0 {
		return ""
	}
	return ensureSentence(strings.Join(bits, "; "))
}

func forensicsDuringPositionStory(p AroundPhase) string {
	f := p.Futures
	if f == nil || !f.Complete {
		return ""
	}
	var s []string
	switch {
	case p.Price.Direction == CVDDirUp && f.OIDirection == CVDDirUp:
		s = append(s, "Open interest rose with price, which often means new longs piled in during the push.")
	case p.Price.Direction == CVDDirUp && f.OIDirection == CVDDirDown:
		s = append(s, "Open interest fell as price rose — that is the classic short-cover / squeeze shape, not a clean new-long trend.")
	case p.Price.Direction == CVDDirDown && f.OIDirection == CVDDirUp:
		s = append(s, "Open interest rose as price fell, which often means new shorts were added into the drop.")
	case p.Price.Direction == CVDDirDown && f.OIDirection == CVDDirDown:
		s = append(s, "Open interest fell with price, which often means longs were liquidated or bailed out during the drop.")
	}
	if f.ShortLiq > 0 && f.ShortLiq > f.LongLiq*1.3 && p.Price.Direction == CVDDirUp {
		s = append(s, "Short liquidations dominated — forced buying can make a rally look stronger than the original bid.")
	} else if f.LongLiq > 0 && f.LongLiq > f.ShortLiq*1.3 && p.Price.Direction == CVDDirDown {
		s = append(s, "Long liquidations dominated — forced selling can make a drop look heavier than the original offer.")
	}
	return strings.Join(s, " ")
}

func forensicsAfterPositionStory(during, after AroundPhase) string {
	f := after.Futures
	if f == nil || !f.Complete {
		return ""
	}
	if during.Complete && during.Futures != nil && during.Futures.Complete {
		if during.Futures.OIDirection == CVDDirUp && f.OIDirection == CVDDirDown {
			return "Open interest turned down afterward, which often means the new positions were being taken off."
		}
	}
	if f.OIDirection == CVDDirDown {
		return "Open interest kept falling after the impulse, which often means the market was de-risking rather than reloading."
	}
	return ""
}

func forensicsEventStory(p AroundPhase, setup bool) string {
	if !p.Complete {
		return ""
	}
	var sweepHigh, sweepLow, absorbBid, absorbAsk bool
	for _, ev := range p.Events {
		switch ev.Kind {
		case AroundEventSweep:
			if ev.Side == "high" || ev.Side == LiquiditySweepSideHigh {
				sweepHigh = true
			} else if ev.Side == "low" || ev.Side == LiquiditySweepSideLow {
				sweepLow = true
			} else {
				sweepHigh = sweepHigh || strings.Contains(strings.ToLower(ev.Title+ev.Summary+ev.Side), "high")
				sweepLow = sweepLow || strings.Contains(strings.ToLower(ev.Title+ev.Summary+ev.Side), "low")
			}
		case AroundEventAbsorption:
			if ev.Side == "bid" || ev.Side == TakerSideBuy {
				absorbBid = true
			} else if ev.Side == "ask" || ev.Side == TakerSideSell {
				absorbAsk = true
			}
		}
	}
	var s []string
	if sweepHigh {
		if setup {
			s = append(s, "Price had already poked a prior high and come back — that can flush stops above the range and leave the tape one-sided.")
		} else {
			s = append(s, "A poke through a prior high was reclaimed, which can be a failed break or a stop flush rather than a clean breakout.")
		}
	}
	if sweepLow {
		if setup {
			s = append(s, "Price had already poked a prior low and come back — that can flush stops below the range before the next push.")
		} else {
			s = append(s, "A poke through a prior low was reclaimed, which can be a failed breakdown or a stop flush rather than a clean dump.")
		}
	}
	if absorbBid {
		s = append(s, "Large selling was absorbed without much downside — bids were defending.")
	}
	if absorbAsk {
		s = append(s, "Large buying was absorbed without much upside — offers were defending.")
	}
	return strings.Join(s, " ")
}

func pickForensicsWhyHorizon(hs []AroundSimilarHorizonStat) AroundSimilarHorizonStat {
	if len(hs) == 0 {
		return AroundSimilarHorizonStat{}
	}
	for _, h := range hs {
		if h.Horizon == "1h" && (h.Sample > 0 || h.Events > 0) {
			return h
		}
	}
	best := hs[0]
	for _, h := range hs[1:] {
		if h.Sample > best.Sample || (h.Sample == best.Sample && h.Events > best.Events) {
			best = h
		}
	}
	return best
}

func ensureSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	s = strings.TrimRight(s, ". ")
	return s + "."
}
