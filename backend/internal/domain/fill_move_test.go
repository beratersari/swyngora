package domain

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

func TestFillMoves_PendingUntilTheMinuteCloses(t *testing.T) {
	fillAt := time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC)
	horizon := fillAt.Add(time.Hour) // 04:04:00, bar closes 04:05:00
	called := 0
	sample := func(time.Time) (float64, bool) {
		called++
		return 110, true
	}

	before := FillMoves(100, fillAt, horizon.Add(59*time.Second), sample)
	if before[0].Status != FillMovePending || before[0].Pct != nil || before[0].Price != nil {
		t.Fatalf("still forming: %+v", before[0])
	}
	if called != 0 {
		t.Fatalf("sample called while pending: %d", called)
	}

	atClose := FillMoves(100, fillAt, horizon.Add(time.Minute), sample)
	if atClose[0].Status != FillMoveReady || atClose[0].Pct == nil || *atClose[0].Pct != 10 {
		t.Fatalf("minute closed: %+v", atClose[0])
	}
	if atClose[1].Status != FillMovePending || atClose[2].Status != FillMovePending {
		t.Fatalf("later windows: %+v", atClose)
	}
	if called != 1 {
		t.Fatalf("sample calls %d", called)
	}
}

func TestFillMoves_SampleSeesTheHorizonInstant(t *testing.T) {
	fillAt := time.Date(2026, 3, 4, 5, 6, 7, 0, time.FixedZone("EET", 3*3600))
	var got time.Time
	moves := FillMoves(50, fillAt, fillAt.Add(2*time.Hour), func(h time.Time) (float64, bool) {
		got = h
		return 40, true
	})
	want := fillAt.UTC().Add(time.Hour)
	if !got.Equal(want) {
		t.Fatalf("horizon %s want %s", got, want)
	}
	if moves[0].Status != FillMoveReady || moves[0].Pct == nil || *moves[0].Pct != -20 {
		t.Fatalf("pct %+v", moves[0])
	}
	if moves[0].Window != FillMove1h || moves[1].Window != FillMove4h || moves[2].Window != FillMove24h {
		t.Fatalf("order %+v", moves)
	}
}

func TestFillMoves_EqualPricesAreReadyZero(t *testing.T) {
	fillAt := time.Date(2026, 1, 2, 0, 0, 30, 0, time.UTC)
	moves := FillMoves(100, fillAt, fillAt.Add(25*time.Hour), func(time.Time) (float64, bool) {
		return 100, true
	})
	for _, m := range moves {
		if m.Status != FillMoveReady || m.Pct == nil || *m.Pct != 0 || m.Price == nil || *m.Price != 100 {
			t.Fatalf("zero move %+v", m)
		}
	}
	raw, err := json.Marshal(moves[0])
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, part := range []string{`"pct":0`, `"price":100`, `"status":"ready"`} {
		if !strings.Contains(body, part) {
			t.Fatalf("json %s missing %s", body, part)
		}
	}
}

func TestFillMoves_MissingIsNotZero(t *testing.T) {
	fillAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	now := fillAt.Add(25 * time.Hour)
	cases := []func(time.Time) (float64, bool){
		func(time.Time) (float64, bool) { return 0, false },
		func(time.Time) (float64, bool) { return 0, true },
		func(time.Time) (float64, bool) { return -5, true },
		func(time.Time) (float64, bool) { return math.NaN(), true },
	}
	for i, sample := range cases {
		moves := FillMoves(100, fillAt, now, sample)
		for _, m := range moves {
			if m.Status != FillMoveMissing || m.Pct != nil || m.Price != nil {
				t.Fatalf("case %d %+v", i, m)
			}
		}
		raw, err := json.Marshal(moves[0])
		if err != nil {
			t.Fatal(err)
		}
		body := string(raw)
		if strings.Contains(body, `"pct"`) || strings.Contains(body, `"price"`) {
			t.Fatalf("case %d leaked numbers %s", i, body)
		}
	}
	if moves := FillMoves(100, fillAt, now, nil); moves[0].Status != FillMoveMissing {
		t.Fatalf("nil sample %+v", moves[0])
	}
}

func TestFillMoves_InvalidFillDoesNotSample(t *testing.T) {
	called := 0
	sample := func(time.Time) (float64, bool) {
		called++
		return 100, true
	}
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		price float64
		at    time.Time
	}{
		{0, now},
		{-1, now},
		{math.NaN(), now},
		{100, time.Time{}},
	} {
		moves := FillMoves(tc.price, tc.at, now.Add(48*time.Hour), sample)
		if len(moves) != 3 {
			t.Fatalf("len %d", len(moves))
		}
		for _, m := range moves {
			if m.Status != FillMoveMissing || m.Pct != nil {
				t.Fatalf("invalid %+v from %+v", m, tc)
			}
		}
	}
	if called != 0 {
		t.Fatalf("sample calls %d", called)
	}
}

func TestFillMoves_PendingJSONOmitsNumbers(t *testing.T) {
	fillAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	raw, err := json.Marshal(FillMoves(100, fillAt, fillAt, func(time.Time) (float64, bool) {
		t.Fatal("pending must not sample")
		return 0, false
	}))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if strings.Contains(body, `"pct"`) || strings.Contains(body, `"price"`) {
		t.Fatalf("pending json %s", body)
	}
}
