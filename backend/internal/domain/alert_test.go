package domain

import (
	"testing"
	"time"
)

func TestAlertConditionMet(t *testing.T) {
	tests := []struct {
		name   string
		cond   AlertCondition
		last   float64
		target float64
		want   bool
	}{
		{"above exact", AlertAbove, 100, 100, true},
		{"above over", AlertAbove, 101, 100, true},
		{"above under", AlertAbove, 99.9, 100, false},
		{"below exact", AlertBelow, 50, 50, true},
		{"below under", AlertBelow, 49, 50, true},
		{"below over", AlertBelow, 50.1, 50, false},
		{"unknown", AlertCondition("sideways"), 1, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AlertConditionMet(tt.cond, tt.last, tt.target); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidAlertCondition(t *testing.T) {
	if !IsValidAlertCondition("above") || !IsValidAlertCondition("below") {
		t.Fatal("expected above/below valid")
	}
	if IsValidAlertCondition("") || IsValidAlertCondition("up") {
		t.Fatal("expected invalid")
	}
}

func TestEvaluateAlert_OneTime(t *testing.T) {
	a := PriceAlert{
		Mode: AlertModeOneTime, Status: AlertStatusActive,
		Condition: AlertAbove, TargetPrice: 100,
	}
	if r := EvaluateAlert(a, 99); r.Fire {
		t.Fatal("should not fire below")
	}
	r := EvaluateAlert(a, 100)
	if !r.Fire || !r.OneTimeDone {
		t.Fatalf("%+v", r)
	}
	a.Status = AlertStatusTriggered
	if r := EvaluateAlert(a, 200); r.Fire {
		t.Fatal("triggered one_time must not fire")
	}
}

func TestEvaluateAlert_RepeatingEdgeCross(t *testing.T) {
	a := PriceAlert{
		Mode: AlertModeRepeating, Status: AlertStatusActive,
		Condition: AlertAbove, TargetPrice: 100, Armed: false,
	}
	// Already above and disarmed — no fire; stay disarmed
	r := EvaluateAlert(a, 105)
	if r.Fire || r.NewArmed {
		t.Fatalf("no fire on hot side disarmed: %+v", r)
	}
	// Move to safe side — re-arm
	r = EvaluateAlert(a, 90)
	if r.Fire || !r.NewArmed || !r.UpdateArmed {
		t.Fatalf("re-arm: %+v", r)
	}
	a.Armed = true
	// Cross up — fire and disarm
	r = EvaluateAlert(a, 101)
	if !r.Fire || r.NewArmed {
		t.Fatalf("cross fire: %+v", r)
	}
	a.Armed = false
	// Stay above — no fire
	r = EvaluateAlert(a, 110)
	if r.Fire {
		t.Fatalf("stay hot: %+v", r)
	}
	// Safe then cross again
	a.Armed = true
	r = EvaluateAlert(a, 100)
	if !r.Fire {
		t.Fatalf("second cross: %+v", r)
	}
}

func TestEvaluateAlert_RepeatingBelow(t *testing.T) {
	a := PriceAlert{
		Mode: AlertModeRepeating, Status: AlertStatusActive,
		Condition: AlertBelow, TargetPrice: 50, Armed: true,
	}
	r := EvaluateAlert(a, 49)
	if !r.Fire || r.NewArmed {
		t.Fatalf("%+v", r)
	}
	a.Armed = false
	r = EvaluateAlert(a, 40)
	if r.Fire {
		t.Fatal("stay below no re-fire")
	}
	r = EvaluateAlert(a, 51)
	if !r.NewArmed {
		t.Fatal("re-arm above target")
	}
}

func TestNormalizeAlertMode(t *testing.T) {
	m, ok := NormalizeAlertMode("")
	if !ok || m != AlertModeOneTime {
		t.Fatalf("%v %v", m, ok)
	}
	m, ok = NormalizeAlertMode("repeating")
	if !ok || m != AlertModeRepeating {
		t.Fatalf("%v %v", m, ok)
	}
	if _, ok := NormalizeAlertMode("forever"); ok {
		t.Fatal("expected invalid")
	}
}

func TestNormalizeDeliveryMode(t *testing.T) {
	m, ok := NormalizeDeliveryMode("")
	if !ok || m != DeliveryImmediate {
		t.Fatalf("%v %v", m, ok)
	}
	m, ok = NormalizeDeliveryMode("hourly_digest")
	if !ok || m != DeliveryHourlyDigest {
		t.Fatalf("%v %v", m, ok)
	}
	if _, ok := NormalizeDeliveryMode("daily"); ok {
		t.Fatal("expected invalid")
	}
}

func TestDigestHourWindow(t *testing.T) {
	t0 := time.Date(2026, 7, 28, 15, 42, 10, 0, time.UTC)
	start, end := DigestHourWindow(t0)
	if !start.Equal(time.Date(2026, 7, 28, 15, 0, 0, 0, time.UTC)) {
		t.Fatalf("start=%v", start)
	}
	if !end.Equal(time.Date(2026, 7, 28, 16, 0, 0, 0, time.UTC)) {
		t.Fatalf("end=%v", end)
	}
}

func TestInQuietHours_SameDayAndCrossMidnight(t *testing.T) {
	loc := time.UTC
	// Same-day 13:00-17:00
	mid := time.Date(2026, 7, 28, 14, 0, 0, 0, time.UTC)
	if !InQuietHours(mid, loc, "13:00", "17:00") {
		t.Fatal("expected quiet at 14:00")
	}
	if InQuietHours(time.Date(2026, 7, 28, 18, 0, 0, 0, time.UTC), loc, "13:00", "17:00") {
		t.Fatal("not quiet at 18:00")
	}
	// Cross midnight 22:00-08:00
	if !InQuietHours(time.Date(2026, 7, 28, 23, 0, 0, 0, time.UTC), loc, "22:00", "08:00") {
		t.Fatal("expected quiet at 23:00")
	}
	if !InQuietHours(time.Date(2026, 7, 29, 2, 0, 0, 0, time.UTC), loc, "22:00", "08:00") {
		t.Fatal("expected quiet at 02:00")
	}
	if InQuietHours(time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC), loc, "22:00", "08:00") {
		t.Fatal("not quiet at noon")
	}
}

func TestQuietHoursEndAfter_CrossMidnight(t *testing.T) {
	loc := time.UTC
	// 23:00 → quiet ends 08:00 next day
	at := time.Date(2026, 7, 28, 23, 15, 0, 0, time.UTC)
	end := QuietHoursEndAfter(at, loc, "22:00", "08:00")
	want := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	if !end.Equal(want) {
		t.Fatalf("got %v want %v", end, want)
	}
	// 03:00 → quiet ends 08:00 same day
	at = time.Date(2026, 7, 29, 3, 0, 0, 0, time.UTC)
	end = QuietHoursEndAfter(at, loc, "22:00", "08:00")
	want = time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	if !end.Equal(want) {
		t.Fatalf("got %v want %v", end, want)
	}
	// Outside quiet → unchanged
	at = time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	end = QuietHoursEndAfter(at, loc, "22:00", "08:00")
	if !end.Equal(at.UTC()) {
		t.Fatalf("outside quiet got %v", end)
	}
}

func TestNextAllowedDeliveryTime(t *testing.T) {
	wh := &ClientWebhook{
		QuietHoursEnabled: true,
		TimeZone:          "UTC",
		QuietStart:        "22:00",
		QuietEnd:          "08:00",
	}
	at := time.Date(2026, 7, 28, 23, 0, 0, 0, time.UTC)
	got := NextAllowedDeliveryTime(at, wh)
	want := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	wh.QuietHoursEnabled = false
	if !NextAllowedDeliveryTime(at, wh).Equal(at) {
		t.Fatal("disabled quiet should not delay")
	}
}