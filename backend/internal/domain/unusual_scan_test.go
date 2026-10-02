package domain

import (
	"math"
	"testing"
	"time"
)

func unusualReading(metric, band string, pct float64) UnusualReading {
	return UnusualReading{
		Metric: metric, Band: band, Percentile: pct, Current: pct,
		Median: 1, Samples: 30, SpanHours: 48, Summary: metric + " summary",
		Unit: UnusualUnitPctChange,
	}
}

func TestBuildUnusualScanOmitsTypicalUnknownAndPrintSize(t *testing.T) {
	rep := UnusualReport{
		Symbol: "BTCUSDT",
		Venues: []UnusualVenue{{
			Exchange: ExchangeBinance,
			Readings: []UnusualReading{
				unusualReading(UnusualMetricFunding, UnusualBandExtremeHigh, 99),
				unusualReading(UnusualMetricOpenInterest, UnusualBandTypical, 50),
				{Metric: UnusualMetricCVD, Band: UnusualBandUnknown, Reason: UnusualReasonFewSamples},
				UnusualPrintSize(),
			},
		}},
	}
	eth := UnusualReport{
		Symbol: "ETHUSDT",
		Venues: []UnusualVenue{{
			Exchange: ExchangeBybit,
			Readings: []UnusualReading{
				unusualReading(UnusualMetricSpread, UnusualBandTypical, 40),
			},
		}},
	}
	got := BuildUnusualScan("all", "all", []UnusualReport{rep, eth}, 1, time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC))
	if len(got.Hits) != 1 {
		t.Fatalf("hits %+v", got.Hits)
	}
	hit := got.Hits[0]
	if hit.Symbol != "BTCUSDT" || hit.Base != "BTC" || hit.Exchange != "binance" || hit.Metric != UnusualMetricFunding {
		t.Fatalf("%+v", hit)
	}
	if hit.Distance != 49 || got.Quiet != 1 || got.Scanned != 2 || got.Skipped != 1 || got.Universe != 3 {
		t.Fatalf("%+v", got)
	}
	if got.Note == "" || got.AsOf.IsZero() {
		t.Fatalf("note %q asOf %v", got.Note, got.AsOf)
	}
}

func TestBuildUnusualScanRanksFarthestFromTypicalFirst(t *testing.T) {
	reports := []UnusualReport{
		{
			Symbol: "ETHUSDT",
			Venues: []UnusualVenue{{
				Exchange: ExchangeBinance,
				Readings: []UnusualReading{unusualReading(UnusualMetricFunding, UnusualBandHigh, 90)},
			}},
		},
		{
			Symbol: "BTCUSDT",
			Venues: []UnusualVenue{{
				Exchange: ExchangeOKX,
				Readings: []UnusualReading{unusualReading(UnusualMetricOpenInterest, UnusualBandExtremeLow, 1)},
			}},
		},
		{
			Symbol: "SOLUSDT",
			Venues: []UnusualVenue{{
				Exchange: ExchangeBybit,
				Readings: []UnusualReading{unusualReading(UnusualMetricLiquidations, UnusualBandExtremeHigh, 99)},
			}},
		},
	}
	got := BuildUnusualScan("all", "all", reports, 0, time.Unix(10, 0).UTC())
	if len(got.Hits) != 3 {
		t.Fatalf("%+v", got.Hits)
	}
	// Percentiles 99 and 1 are the same distance from 50, so the symbol breaks the tie.
	if got.Hits[0].Symbol != "BTCUSDT" || got.Hits[1].Symbol != "SOLUSDT" || got.Hits[2].Symbol != "ETHUSDT" {
		t.Fatalf("order %s %s %s", got.Hits[0].Symbol, got.Hits[1].Symbol, got.Hits[2].Symbol)
	}
	if math.Abs(got.Hits[0].Distance-49) > 0 || math.Abs(got.Hits[1].Distance-49) > 0 || math.Abs(got.Hits[2].Distance-40) > 0 {
		t.Fatalf("distances %+v %+v %+v", got.Hits[0].Distance, got.Hits[1].Distance, got.Hits[2].Distance)
	}
}

func TestBuildUnusualScanFiltersMetric(t *testing.T) {
	rep := UnusualReport{
		Symbol: "BTCUSDT",
		Venues: []UnusualVenue{{
			Exchange: ExchangeBinance,
			Readings: []UnusualReading{
				unusualReading(UnusualMetricFunding, UnusualBandExtremeHigh, 99),
				unusualReading(UnusualMetricSpread, UnusualBandLow, 5),
			},
		}},
	}
	got := BuildUnusualScan("binance", UnusualMetricSpread, []UnusualReport{rep}, 0, time.Unix(1, 0).UTC())
	if len(got.Hits) != 1 || got.Hits[0].Metric != UnusualMetricSpread || got.Metric != UnusualMetricSpread {
		t.Fatalf("%+v", got)
	}
	if got.Quiet != 0 {
		t.Fatalf("quiet %d", got.Quiet)
	}
}

func TestParseUnusualScanMetricAndLimit(t *testing.T) {
	if got, err := ParseUnusualScanMetric(""); err != nil || got != "all" {
		t.Fatalf("%s %v", got, err)
	}
	if got, err := ParseUnusualScanMetric("CVD"); err != nil || got != UnusualMetricCVD {
		t.Fatalf("%s %v", got, err)
	}
	if _, err := ParseUnusualScanMetric("print_size"); err == nil {
		t.Fatal("print size")
	}
	if _, err := ParseUnusualScanMetric("basis"); err == nil {
		t.Fatal("basis")
	}
	if ParseUnusualScanLimit(0) != LiquidRecorderLimit || ParseUnusualScanLimit(2) != 5 || ParseUnusualScanLimit(99) != LiquidRecorderLimit {
		t.Fatalf("limits %d %d %d", ParseUnusualScanLimit(0), ParseUnusualScanLimit(2), ParseUnusualScanLimit(99))
	}
}
