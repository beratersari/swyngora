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