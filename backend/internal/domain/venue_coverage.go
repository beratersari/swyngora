package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	VenueReadFresh   = "fresh"
	VenueReadStale   = "stale"
	VenueReadMissing = "missing"

	// VenueTapeMaxAge is how old the newest aggressive print may be before
	// that venue's tape is old rather than a live market read.
	VenueTapeMaxAge = 2 * time.Minute
	// VenueBookMaxAge is how old a perpetual book snapshot may be.
	VenueBookMaxAge = 45 * time.Second
)

// VenueSample is one venue's answer while a market-wide read is built.
// OK means the venue responded. At is the newest data timestamp; zero with
// OK means the venue answered and had nothing to print.
type VenueSample struct {
	Exchange Exchange
	OK       bool
	At       time.Time
}

// VenueRead is one expected venue on a combined market read.
type VenueRead struct {
	Exchange Exchange
	Status   string // fresh | stale | missing
	AsOf     time.Time
	AgeSec   int
}

// MarketCoverage says whether every expected venue is fresh.
// Complete is false when any venue is missing or old, and the market
// verdict must not be stated.
type MarketCoverage struct {
	Complete bool
	Venues   []VenueRead
	Missing  []string
	Stale    []string
}

// CoverVenues classifies each expected venue. A venue with no successful
// sample is missing. A successful sample whose newest timestamp is older
// than maxAge is stale. A successful sample with no timestamp is fresh:
// the venue answered and printed nothing.
func CoverVenues(expected []Exchange, samples []VenueSample, now time.Time, maxAge time.Duration) MarketCoverage {
	now = now.UTC()
	if maxAge <= 0 {
		maxAge = VenueTapeMaxAge
	}
	merged := MergeVenueSamples(samples)
	out := MarketCoverage{
		Venues:  make([]VenueRead, 0, len(expected)),
		Missing: []string{},
		Stale:   []string{},
	}
	for _, ex := range expected {
		sample, ok := merged[ex]
		row := VenueRead{Exchange: ex, Status: VenueReadMissing}
		if ok && sample.OK {
			row.Status = VenueReadFresh
			if !sample.At.IsZero() {
				at := sample.At.UTC()
				row.AsOf = at
				age := now.Sub(at)
				if age < 0 {
					age = 0
				}
				row.AgeSec = int(age.Seconds())
				if age > maxAge {
					row.Status = VenueReadStale
				}
			}
		}
		switch row.Status {
		case VenueReadMissing:
			out.Missing = append(out.Missing, string(ex))
		case VenueReadStale:
			out.Stale = append(out.Stale, string(ex))
		}
		out.Venues = append(out.Venues, row)
	}
	out.Complete = len(expected) > 0 && len(out.Missing) == 0 && len(out.Stale) == 0
	return out
}

// MergeVenueSamples keeps one row per venue. The venue responded when any
// sample did. The timestamp is the newest data time among those samples.
func MergeVenueSamples(samples []VenueSample) map[Exchange]VenueSample {
	out := make(map[Exchange]VenueSample, len(samples))
	for _, s := range samples {
		if s.Exchange == "" || s.Exchange == "all" {
			continue
		}
		cur, ok := out[s.Exchange]
		if !ok {
			if !s.At.IsZero() {
				s.At = s.At.UTC()
			}
			out[s.Exchange] = s
			continue
		}
		if s.OK {
			cur.OK = true
		}
		if !s.At.IsZero() && s.At.UTC().After(cur.At) {
			cur.At = s.At.UTC()
		}
		out[s.Exchange] = cur
	}
	return out
}

// ExpectedFuturesVenues is the set a combined or single futures read must cover.
func ExpectedFuturesVenues(exchange string) []Exchange {
	switch strings.ToLower(strings.TrimSpace(exchange)) {
	case "", "all", "combined":
		return FuturesVenues()
	default:
		return []Exchange{Exchange(exchange)}
	}
}

// Explain tells which venues are missing and which are old.
// An empty string means every expected venue is fresh.
func (c MarketCoverage) Explain() string {
	if c.Complete {
		return ""
	}
	parts := []string{"No market read."}
	if len(c.Missing) > 0 {
		parts = append(parts, "Missing: "+strings.Join(c.Missing, ", ")+".")
	}
	if len(c.Stale) > 0 {
		named := make([]string, 0, len(c.Stale))
		for _, name := range c.Stale {
			label := name
			for _, v := range c.Venues {
				if string(v.Exchange) == name && v.AgeSec > 0 {
					label = fmt.Sprintf("%s (%s old)", name, formatCoverageAge(v.AgeSec))
					break
				}
			}
			named = append(named, label)
		}
		parts = append(parts, "Old: "+strings.Join(named, ", ")+".")
	}
	return strings.Join(parts, " ")
}

func formatCoverageAge(sec int) string {
	if sec < 90 {
		return fmt.Sprintf("%ds", sec)
	}
	if sec < 90*60 {
		return fmt.Sprintf("%dm", sec/60)
	}
	return fmt.Sprintf("%dh", sec/3600)
}

// CoverageFromLiquidationFeed maps socket health onto fresh, stale, or missing.
// A venue that has never been seen is missing. A venue that is down and whose
// last clock is older than the stall budget is old. A live socket is fresh.
func CoverageFromLiquidationFeed(feed LiquidationFeed, exchange string, now time.Time) MarketCoverage {
	now = now.UTC()
	samples := make([]VenueSample, 0, len(feed.Venues))
	for _, h := range feed.Venues {
		at := h.LastSeenAt
		if h.LastEventAt.After(at) {
			at = h.LastEventAt
		}
		sample := VenueSample{Exchange: Exchange(h.Exchange)}
		if h.Live {
			sample.OK = true
			sample.At = at
			samples = append(samples, sample)
			continue
		}
		if at.IsZero() || now.Sub(at.UTC()) <= liquidationFeedStall {
			samples = append(samples, sample)
			continue
		}
		sample.OK = true
		sample.At = at
		samples = append(samples, sample)
	}
	return CoverVenues(ExpectedFuturesVenues(exchange), samples, now, liquidationFeedStall)
}

// ApplyOrderCountCoverage withholds the market lean when any venue is missing or old.
func ApplyOrderCountCoverage(out *OrderCountOverview, samples []VenueSample, now time.Time) {
	if out == nil {
		return
	}
	out.Coverage = CoverVenues(ExpectedFuturesVenues(out.Exchange), samples, now, VenueTapeMaxAge)
	out.Definite = out.Coverage.Complete
	if out.Definite {
		return
	}
	msg := out.Coverage.Explain()
	out.Summary = msg
	for i := range out.Windows {
		out.Windows[i].Lean = ""
		out.Windows[i].Summary = msg
	}
}

// ApplyPrintSizeCoverage withholds the market size read when coverage is incomplete.
func ApplyPrintSizeCoverage(out *PrintSizeOverview, samples []VenueSample, now time.Time) {
	if out == nil {
		return
	}
	out.Coverage = CoverVenues(ExpectedFuturesVenues(out.Exchange), samples, now, VenueTapeMaxAge)
	out.Definite = out.Coverage.Complete
	if out.Definite {
		return
	}
	msg := out.Coverage.Explain()
	out.Summary = msg
	for i := range out.Windows {
		out.Windows[i].Read = ""
		out.Windows[i].Summary = msg
	}
}

// ApplyPerpDepthCoverage withholds the bid/ask lean when a book is missing or old.
func ApplyPerpDepthCoverage(out *PerpDepthOverview, samples []VenueSample, now time.Time) {
	if out == nil {
		return
	}
	out.Coverage = CoverVenues(ExpectedFuturesVenues(out.Exchange), samples, now, VenueBookMaxAge)
	out.Definite = out.Coverage.Complete
	if out.Definite {
		return
	}
	msg := out.Coverage.Explain()
	out.Lean = ""
	out.Summary = msg
	for i := range out.Windows {
		out.Windows[i].Kind = ""
		out.Windows[i].DepthKind = ""
		out.Windows[i].Summary = msg
	}
	for i := range out.Bands {
		out.Bands[i].Lean = ""
		out.Bands[i].Summary = msg
	}
}

// ApplyLiquidationCoverage withholds a market-wide liquidation result when a feed is down or old.
func ApplyLiquidationCoverage(out *LiquidationOverview, now time.Time) {
	if out == nil {
		return
	}
	out.Coverage = CoverageFromLiquidationFeed(out.Feed, out.Exchange, now)
	out.Definite = out.Coverage.Complete
}
