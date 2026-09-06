package domain

import (
	"testing"
	"time"
)

func hourlyOI(now time.Time, n int, base float64, lastJump float64) []OptionsHistoryPoint {
	out := make([]OptionsHistoryPoint, 0, n)
	for i := n - 1; i >= 0; i-- {
		v := base
		if i == 0 {
			v = base * (1 + lastJump)
		}
		out = append(out, OptionsHistoryPoint{
			Time:  now.Add(-time.Duration(i) * time.Hour),
			HasOI: true, OpenInterest: v, OpenInterestUSD: v,
			HasDVOL: true, DVOL: 50,
			HasPutCallOI: true, PutCallOI: 1.0,
		})
	}
	return out
}

func TestDetectOptionsMoves_OIJump(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	// Quiet 1h changes (~0.2%), then a 20% last-hour jump.
	pts := hourlyOI(now, 12, 100, 0.20)
	got := DetectOptionsMoves(pts, now)
	if got == nil || len(got.Findings) == 0 {
		t.Fatal("expected oi jump")
	}
	found := false
	for _, f := range got.Findings {
		if f.Kind == OptionsMoveOIJump && f.Window == OptionsHistoryWindow1h {
			found = true
			if f.Severity != OptionsFindingNotable {
				t.Fatalf("severity=%s", f.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("%+v", got.Findings)
	}
}

func TestDetectOptionsMoves_QuietIsSilent(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	pts := hourlyOI(now, 12, 100, 0.003)
	if got := DetectOptionsMoves(pts, now); got != nil {
		t.Fatalf("quiet tape should stay silent %+v", got.Findings)
	}
}

func TestDetectOptionsMoves_IVDrop(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	pts := make([]OptionsHistoryPoint, 0, 12)
	for i := 11; i >= 0; i-- {
		iv := 50.0
		if i == 0 {
			iv = 44
		} else if i == 1 {
			iv = 50.2
		}
		pts = append(pts, OptionsHistoryPoint{
			Time:    now.Add(-time.Duration(i) * time.Hour),
			HasDVOL: true, DVOL: iv,
		})
	}
	got := DetectOptionsMoves(pts, now)
	if got == nil {
		t.Fatal("expected iv drop")
	}
	found := false
	for _, f := range got.Findings {
		if f.Kind == OptionsMoveIVDrop {
			found = true
		}
	}
	if !found {
		t.Fatalf("%+v", got.Findings)
	}
}

func TestDetectOptionsMoves_NeedHistory(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	if got := DetectOptionsMoves(nil, now); got != nil {
		t.Fatal("empty")
	}
	pts := []OptionsHistoryPoint{
		{Time: now.Add(-time.Hour), HasOI: true, OpenInterest: 100},
		{Time: now, HasOI: true, OpenInterest: 200},
	}
	if got := DetectOptionsMoves(pts, now); got != nil {
		t.Fatalf("too few priors %+v", got)
	}
}
