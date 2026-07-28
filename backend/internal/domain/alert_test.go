package domain

import "testing"

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