package domain

import (
	"strings"
	"testing"
	"time"
)

func TestAroundCurveAt_ContangoBeforeMove(t *testing.T) {
	at := time.Date(2026, 9, 19, 14, 0, 0, 0, time.UTC)
	hist := []BasisCurvePoint{
		curvePointAt(at.Add(-2*time.Hour), 64000, 64600),
		curvePointAt(at.Add(-time.Hour), 64000, 64700),
	}
	live := BasisCurve{
		Complete: true, Shape: BasisCurveContango,
		Near: &BasisContractPoint{Label: "25 Sep 2026", Mark: 64000},
		Far:  &BasisContractPoint{Label: "25 Dec 2026", Mark: 64700},
	}
	got := AroundCurveAt(hist, live, ExchangeBinance, at)
	if got == nil || !got.Complete || got.Shape != BasisCurveContango {
		t.Fatalf("%+v", got)
	}
	if got.At != at.Add(-time.Hour) {
		t.Fatalf("at %s", got.At)
	}
	if !strings.Contains(got.Summary, "contango") || !strings.Contains(got.Summary, "25 Dec 2026") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestAroundCurveAt_MissingHistory(t *testing.T) {
	at := time.Date(2026, 9, 19, 14, 0, 0, 0, time.UTC)
	live := BasisCurve{
		Complete: true,
		Near:     &BasisContractPoint{Label: "Near"},
		Far:      &BasisContractPoint{Label: "Far"},
	}
	if got := AroundCurveAt(nil, live, ExchangeBinance, at); got != nil {
		t.Fatalf("%+v", got)
	}
}
