package domain

import (
	"testing"
	"time"
)

func TestPerpBookHistoryStep(t *testing.T) {
	if PerpBookHistoryStep(2*time.Hour) != time.Minute {
		t.Fatal("short")
	}
	if PerpBookHistoryStep(12*time.Hour) != 5*time.Minute {
		t.Fatal("day")
	}
	if PerpBookHistoryStep(3*24*time.Hour) != 30*time.Minute {
		t.Fatal("week")
	}
	if PerpBookHistoryStep(PerpDepthChartSpan) != 2*time.Hour {
		t.Fatal("month")
	}
}

func TestBuildPerpBookHistory_KeepsNewestBandInTheStep(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(3 * time.Hour)
	early := from.Add(10 * time.Minute)
	late := from.Add(40 * time.Minute)
	got := BuildPerpBookHistory("btcusdt", "all", from, to, []PerpBookBandSample{
		{Time: early, Exchange: "all", Symbol: "BTCUSDT", RangePct: 0.1, Bid: 1, Ask: 1, Mid: 100},
		{Time: early, Exchange: "all", Symbol: "BTCUSDT", RangePct: 1, Bid: 10, Ask: 4, Mid: 100},
		{Time: late, Exchange: "all", Symbol: "BTCUSDT", RangePct: 0.1, Bid: 2, Ask: 3, Mid: 101},
		{Time: late, Exchange: "all", Symbol: "BTCUSDT", RangePct: 0.5, Bid: 6, Ask: 5, Mid: 101},
		{Time: late, Exchange: "all", Symbol: "BTCUSDT", RangePct: 1, Bid: 12, Ask: 8, Mid: 101},
		{Time: late, Exchange: "all", Symbol: "BTCUSDT", RangePct: 2, Bid: 99, Ask: 99, Mid: 101},
	}, []PerpBookGap{{Time: from.Add(2 * time.Hour), Symbol: "BTCUSDT", Missing: []string{"okx"}}})
	if got.Step != time.Minute || len(got.Columns) != 2 {
		t.Fatalf("step %s cols %d", got.Step, len(got.Columns))
	}
	col := got.Columns[1]
	if col.Mid != 101 || len(col.Bands) != 3 || col.Bands[2].Bid != 12 || col.Bands[2].RangePct != 1 {
		t.Fatalf("%+v", col)
	}
	if len(got.Gaps) != 1 || got.Gaps[0].Missing[0] != "okx" {
		t.Fatalf("gaps %+v", got.Gaps)
	}
}

func TestBuildPerpBookHistory_FoldsToTwoHours(t *testing.T) {
	to := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	from := to.Add(-10 * 24 * time.Hour)
	first := from.Add(30 * time.Minute)
	second := from.Add(90 * time.Minute)
	got := BuildPerpBookHistory("BTCUSDT", "all", from, to, []PerpBookBandSample{
		{Time: first, RangePct: 1, Bid: 1, Ask: 1, Mid: 50},
		{Time: second, RangePct: 1, Bid: 4, Ask: 2, Mid: 55},
	}, nil)
	if got.Step != 2*time.Hour || len(got.Columns) != 1 || got.Columns[0].Bands[0].Bid != 4 || got.Columns[0].Mid != 55 {
		t.Fatalf("%+v", got.Columns)
	}
}

func TestAttachPerpDepthChart_DoesNotReplaceJumpTape(t *testing.T) {
	asOf := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	live := &PerpDepthOverview{AsOf: asOf, History: []PerpDepthTotalPoint{{Time: asOf.Add(-time.Hour), Total: 10}}}
	AttachPerpDepthChart(live, []PerpDepthTotalPoint{
		{Time: asOf.Add(-10 * 24 * time.Hour), Total: 5},
		{Time: asOf.Add(-time.Hour), Total: 9},
		{Time: asOf.Add(-40 * 24 * time.Hour), Total: 1},
	})
	if len(live.History) != 1 || live.History[0].Total != 10 {
		t.Fatalf("history %+v", live.History)
	}
	if len(live.Chart) != 2 || live.Chart[0].Total != 5 || live.Chart[1].Total != 9 {
		t.Fatalf("chart %+v", live.Chart)
	}
}
